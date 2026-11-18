//! Windows backend.
//!
//! A process can be attached to at most one console. To deliver a console
//! control event or console input to a *foreign* console we must be attached
//! to it, and attaching means detaching from our own first (which destroys it
//! if we were the last process using it). Three routes avoid that problem
//! whenever possible, see [`Route`].

use std::ffi::OsString;
use std::os::windows::process::CommandExt;
use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Mutex;
use std::time::{Duration, Instant};

use windows_sys::Win32::Foundation::{
    CloseHandle, GetLastError, ERROR_ACCESS_DENIED, ERROR_INVALID_HANDLE, ERROR_INVALID_PARAMETER,
    FALSE, GENERIC_READ, GENERIC_WRITE, HANDLE, HMODULE, HWND, INVALID_HANDLE_VALUE, LPARAM, TRUE,
    WAIT_TIMEOUT,
};
use windows_sys::Win32::Storage::FileSystem::{
    CreateFileW, FILE_SHARE_READ, FILE_SHARE_WRITE, OPEN_EXISTING,
};
use windows_sys::Win32::System::Console::{
    AttachConsole, FreeConsole, GenerateConsoleCtrlEvent, GetConsoleProcessList,
    SetConsoleCtrlHandler, WriteConsoleInputW, CTRL_BREAK_EVENT, CTRL_C_EVENT, INPUT_RECORD,
    INPUT_RECORD_0, KEY_EVENT, KEY_EVENT_RECORD, KEY_EVENT_RECORD_0,
};
use windows_sys::Win32::System::LibraryLoader::{
    GetModuleHandleExW, GetModuleHandleW, GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS,
    GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT,
};
use windows_sys::Win32::System::Threading::{
    ExitProcess, OpenProcess, TerminateProcess, WaitForSingleObject, DETACHED_PROCESS,
    PROCESS_QUERY_LIMITED_INFORMATION, PROCESS_SYNCHRONIZE, PROCESS_TERMINATE,
};
use windows_sys::Win32::UI::WindowsAndMessaging::{
    EnumChildWindows, EnumWindows, GetClassNameW, GetWindow, GetWindowThreadProcessId,
    IsWindowVisible, PostMessageW, SendMessageW, GW_OWNER, WM_CLOSE,
};

use crate::{Error, Method, Result};

/// Opaque top-level window handle (an `HWND` value).
pub type WindowHandle = usize;

/// Which executable acts as the detached helper when one is required.
#[derive(Debug, Clone)]
pub enum Helper {
    /// Re-launch the current executable (default). Requires that the code
    /// runs from an `.exe`, not from a DLL loaded into a foreign host.
    SelfExe,
    /// Launch another executable that links this crate (for example the
    /// `terminator` CLI). It must handle the helper marker before `main`,
    /// which every binary linking this crate does automatically.
    Path(PathBuf),
}

static HELPER: Mutex<Helper> = Mutex::new(Helper::SelfExe);

/// Choose which executable is used when a detached helper is required.
pub fn set_helper(helper: Helper) {
    *HELPER.lock().unwrap_or_else(|e| e.into_inner()) = helper;
}

// ---------------------------------------------------------------------------
// Helper protocol
// ---------------------------------------------------------------------------

/// First argument of a helper invocation. Deliberately unlikely to collide
/// with a real command line.
const MARKER: &str = "--terminator-internal-helper-v1";
const PROTOCOL: &str = "1";
/// How long the parent waits for the helper before assuming that the helper
/// executable does not understand the protocol and killing it.
const HELPER_TIMEOUT: Duration = Duration::from_secs(10);

// Helper exit codes. Distinct from STATUS_CONTROL_C_EXIT (0xC000013A) so a
// helper killed by the event it generated is never mistaken for success.
const H_OK: i32 = 0;
const H_BAD_ARGS: i32 = 100;
const H_NO_CONSOLE: i32 = 104;
const H_ACCESS_DENIED: i32 = 105;
const H_NO_PROCESS: i32 = 106;
const H_ATTACH_FAILED: i32 = 107;
const H_ACTION_FAILED: i32 = 109;

/// What a helper (or an attached caller) has to do on the target console.
enum Task {
    Ctrl(u32),
    Message(String),
}

impl Task {
    fn describe(&self) -> &'static str {
        match self {
            Task::Ctrl(_) => "sending a console control event",
            Task::Message(_) => "writing to console input",
        }
    }
}

// Run before `main` by the C runtime (`.CRT$XCB` sorts before the `.CRT$XCU`
// section used by `ctor`/`inventory`, so we run first). Keeps the "forgot to
// call the helper entry point" failure mode impossible: any binary that can
// spawn a helper contains this initializer, because `spawn_helper` references
// the static below and so keeps it linked.
#[used]
#[link_section = ".CRT$XCB"]
static HELPER_CTOR: unsafe extern "C" fn() = helper_ctor;

unsafe extern "C" fn helper_ctor() {
    run_helper_if_requested();
}

/// If this process was started as a helper, perform the requested task and
/// exit. Otherwise return immediately.
///
/// Called automatically before `main`; calling it again is harmless.
pub fn run_helper_if_requested() {
    let args: Vec<OsString> = std::env::args_os().collect();
    if args.len() < 2 || args[1] != MARKER {
        return;
    }
    let code = helper_main(&args[2..]);
    // SAFETY: plain FFI call, never returns.
    unsafe { ExitProcess(code as u32) }
}

fn helper_main(args: &[OsString]) -> i32 {
    // <protocol> <ctrl|msg> <pid> <payload>
    if args.len() != 4 || args[0] != PROTOCOL {
        return H_BAD_ARGS;
    }
    let (Some(kind), Some(pid), Some(payload)) =
        (args[1].to_str(), args[2].to_str(), args[3].to_str())
    else {
        return H_BAD_ARGS;
    };
    let Ok(pid) = pid.parse::<u32>() else {
        return H_BAD_ARGS;
    };
    let task = match kind {
        "ctrl" => match payload.parse::<u32>() {
            Ok(ev) if ev == CTRL_C_EVENT || ev == CTRL_BREAK_EVENT => Task::Ctrl(ev),
            _ => return H_BAD_ARGS,
        },
        "msg" => match hex_decode(payload) {
            Some(m) => Task::Message(m),
            None => return H_BAD_ARGS,
        },
        _ => return H_BAD_ARGS,
    };

    if let Err(e) = attach(pid) {
        return match e {
            Error::ProcessNotFound(_) => H_NO_PROCESS,
            Error::NoConsole(_) => H_NO_CONSOLE,
            Error::Os(ref io) if io.raw_os_error() == Some(ERROR_ACCESS_DENIED as i32) => {
                H_ACCESS_DENIED
            }
            _ => H_ATTACH_FAILED,
        };
    }
    match run_here(&task) {
        Ok(()) => H_OK,
        Err(_) => H_ACTION_FAILED,
    }
}

fn hex_encode(s: &str) -> String {
    s.bytes().map(|b| format!("{b:02x}")).collect()
}

fn hex_decode(s: &str) -> Option<String> {
    if s.len() % 2 != 0 {
        return None;
    }
    let bytes: Option<Vec<u8>> = (0..s.len())
        .step_by(2)
        .map(|i| u8::from_str_radix(s.get(i..i + 2)?, 16).ok())
        .collect();
    String::from_utf8(bytes?).ok()
}

// ---------------------------------------------------------------------------
// Routing
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Route {
    /// Target shares our console: act on it directly.
    Direct,
    /// We have no console: attach to the target's, act, detach.
    InProcess,
    /// We have our own console: a detached helper does the attaching.
    Helper,
}

/// Serializes console attach/detach (process-wide state).
static CONSOLE_LOCK: Mutex<()> = Mutex::new(());

fn console_pids() -> Vec<u32> {
    let mut cap = 16usize;
    loop {
        let mut buf = vec![0u32; cap];
        // SAFETY: `buf` holds `cap` writable u32 values.
        let n = unsafe { GetConsoleProcessList(buf.as_mut_ptr(), cap as u32) } as usize;
        if n == 0 {
            return Vec::new(); // not attached to a console
        }
        if n <= cap {
            buf.truncate(n);
            return buf;
        }
        cap = n + 4;
    }
}

fn route_for(pid: u32) -> Route {
    let pids = console_pids();
    if pids.is_empty() {
        Route::InProcess
    } else if pids.contains(&pid) {
        Route::Direct
    } else {
        Route::Helper
    }
}

fn trace(route: Route, pid: u32) {
    if std::env::var_os("TERMINATOR_DEBUG").is_some() {
        eprintln!("terminator: pid={pid} route={route:?}");
    }
}

fn dispatch(pid: u32, task: Task) -> Result<()> {
    if !is_running(pid) {
        return Err(Error::ProcessNotFound(pid));
    }
    let route = route_for(pid);
    trace(route, pid);
    match route {
        Route::Direct => {
            let _guard = CONSOLE_LOCK.lock().unwrap_or_else(|e| e.into_inner());
            run_here(&task)
        }
        Route::InProcess => {
            let _guard = CONSOLE_LOCK.lock().unwrap_or_else(|e| e.into_inner());
            attach(pid)?;
            let result = run_here(&task);
            // SAFETY: plain FFI call; we attached above.
            unsafe { FreeConsole() };
            result
        }
        Route::Helper => spawn_helper(pid, &task),
    }
}

/// Perform `task` on the console we are currently attached to.
fn run_here(task: &Task) -> Result<()> {
    match task {
        Task::Ctrl(event) => generate_here(*event),
        Task::Message(msg) => write_input_here(msg),
    }
}

fn attach(pid: u32) -> Result<()> {
    // SAFETY: plain FFI calls.
    unsafe {
        if AttachConsole(pid) != 0 {
            return Ok(());
        }
        match GetLastError() {
            ERROR_INVALID_PARAMETER => Err(Error::ProcessNotFound(pid)),
            ERROR_INVALID_HANDLE => Err(Error::NoConsole(pid)),
            // Either we are already attached to a console or the target is
            // not accessible (for example it is elevated and we are not).
            code => Err(Error::Os(std::io::Error::from_raw_os_error(code as i32))),
        }
    }
}

// ---------------------------------------------------------------------------
// Console control events and console input
// ---------------------------------------------------------------------------

static EVENT_SEEN: AtomicBool = AtomicBool::new(false);

/// Swallows Ctrl+C / Ctrl+Break so that an event generated for the whole
/// console does not terminate the process that generated it.
///
/// A handler is used instead of `SetConsoleCtrlHandler(NULL, TRUE)` because
/// the latter is inherited by child processes started meanwhile.
unsafe extern "system" fn swallow(ctrl_type: u32) -> i32 {
    if ctrl_type == CTRL_C_EVENT || ctrl_type == CTRL_BREAK_EVENT {
        EVENT_SEEN.store(true, Ordering::SeqCst);
        TRUE
    } else {
        FALSE
    }
}

fn generate_here(event: u32) -> Result<()> {
    EVENT_SEEN.store(false, Ordering::SeqCst);
    // SAFETY: plain FFI calls; `swallow` has the required signature.
    unsafe {
        if SetConsoleCtrlHandler(Some(swallow), TRUE) == 0 {
            return Err(Error::Os(std::io::Error::last_os_error()));
        }
        let ok = GenerateConsoleCtrlEvent(event, 0);
        let err = std::io::Error::last_os_error();
        if ok != 0 {
            // The event arrives on a new thread. Wait for it so removing the
            // handler cannot race with delivery. (If this process ignores
            // Ctrl+C the handler never fires; the short timeout covers that.)
            let deadline = Instant::now() + Duration::from_millis(500);
            while !EVENT_SEEN.load(Ordering::SeqCst) && Instant::now() < deadline {
                std::thread::sleep(Duration::from_millis(1));
            }
        }
        SetConsoleCtrlHandler(Some(swallow), FALSE);
        if ok == 0 {
            return Err(Error::Os(err));
        }
    }
    Ok(())
}

const VK_RETURN: u16 = 0x0D;
const SCAN_RETURN: u16 = 0x1C;

fn key_record(down: bool, vk: u16, scan: u16, ch: u16) -> INPUT_RECORD {
    INPUT_RECORD {
        EventType: KEY_EVENT as u16,
        Event: INPUT_RECORD_0 {
            KeyEvent: KEY_EVENT_RECORD {
                bKeyDown: if down { TRUE } else { 0 },
                wRepeatCount: 1,
                wVirtualKeyCode: vk,
                wVirtualScanCode: scan,
                uChar: KEY_EVENT_RECORD_0 { UnicodeChar: ch },
                dwControlKeyState: 0,
            },
        },
    }
}

/// Turns text into console key events.
///
/// Ordinary characters are key-down only with no virtual key code. Line breaks (`\r`, `\n` and
/// `\r\n`) become one real Enter press (down and up, with the Return virtual key and scan code).
fn key_records(msg: &str) -> Vec<INPUT_RECORD> {
    let mut out = Vec::with_capacity(msg.len());
    let mut prev_cr = false;
    for ch in msg.encode_utf16() {
        match ch {
            0x0D | 0x0A => {
                // The `\n` of a `\r\n` pair is already covered by the Enter sent for `\r`.
                if !(ch == 0x0A && prev_cr) {
                    out.push(key_record(true, VK_RETURN, SCAN_RETURN, 0x0D));
                    out.push(key_record(false, VK_RETURN, SCAN_RETURN, 0x0D));
                }
                prev_cr = ch == 0x0D;
            }
            _ => {
                out.push(key_record(true, 0, 0, ch));
                prev_cr = false;
            }
        }
    }
    out
}

fn write_input_here(msg: &str) -> Result<()> {
    if msg.is_empty() {
        return Err(Error::Os(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "empty message",
        )));
    }
    let records = key_records(msg);
    let name: Vec<u16> = "CONIN$\0".encode_utf16().collect();
    // SAFETY: `name` is NUL-terminated; `records` outlives the call.
    unsafe {
        // CONIN$ is the input buffer of the console we are attached to, even
        // when the standard handles were redirected.
        let handle: HANDLE = CreateFileW(
            name.as_ptr(),
            GENERIC_READ | GENERIC_WRITE,
            FILE_SHARE_READ | FILE_SHARE_WRITE,
            std::ptr::null(),
            OPEN_EXISTING,
            0,
            std::ptr::null_mut(),
        );
        if handle == INVALID_HANDLE_VALUE {
            return Err(Error::Os(std::io::Error::last_os_error()));
        }
        let mut written = 0u32;
        let ok = WriteConsoleInputW(handle, records.as_ptr(), records.len() as u32, &mut written);
        let err = std::io::Error::last_os_error();
        CloseHandle(handle);
        if ok == 0 {
            return Err(Error::Os(err));
        }
    }
    Ok(())
}

// ---------------------------------------------------------------------------
// Helper process
// ---------------------------------------------------------------------------

/// Whether the code of this crate is part of the process' main executable
/// (as opposed to a DLL loaded into some other host).
fn code_lives_in_exe() -> bool {
    let mut module: HMODULE = std::ptr::null_mut();
    // SAFETY: the address belongs to this function, which is always mapped.
    unsafe {
        let ok = GetModuleHandleExW(
            GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS | GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT,
            code_lives_in_exe as *const () as *const u16,
            &mut module,
        );
        ok != 0 && module == GetModuleHandleW(std::ptr::null())
    }
}

fn helper_program() -> Result<PathBuf> {
    let helper = HELPER.lock().unwrap_or_else(|e| e.into_inner()).clone();
    match helper {
        Helper::Path(path) => Ok(path),
        Helper::SelfExe => {
            if !code_lives_in_exe() {
                return Err(Error::HelperUnavailable(
                    "terminator is loaded as a DLL into a foreign host, which cannot be \
                     re-launched; call terminator::set_helper(Helper::Path(..)) with an \
                     executable that links terminator (for example terminator.exe)"
                        .into(),
                ));
            }
            std::env::current_exe().map_err(Error::Os)
        }
    }
}

fn spawn_helper(pid: u32, task: &Task) -> Result<()> {
    // Keep the pre-main initializer linked into every binary that can spawn
    // a helper (otherwise the linker may drop the otherwise unreferenced static).
    std::hint::black_box(&HELPER_CTOR);

    let program = helper_program()?;
    let (kind, payload) = match task {
        Task::Ctrl(event) => ("ctrl", event.to_string()),
        Task::Message(msg) => ("msg", hex_encode(msg)),
    };
    let mut child = Command::new(&program)
        .arg(MARKER)
        .arg(PROTOCOL)
        .arg(kind)
        .arg(pid.to_string())
        .arg(payload)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        // No console of its own, so it is free to attach to the target's.
        .creation_flags(DETACHED_PROCESS)
        .spawn()
        .map_err(Error::Os)?;

    let deadline = Instant::now() + HELPER_TIMEOUT;
    let status = loop {
        match child.try_wait().map_err(Error::Os)? {
            Some(status) => break status,
            None if Instant::now() >= deadline => {
                let _ = child.kill();
                let _ = child.wait();
                return Err(Error::HelperUnavailable(format!(
                    "{} did not finish within {:?}; it probably does not link terminator \
                     (the helper executable must handle the helper marker before main)",
                    program.display(),
                    HELPER_TIMEOUT
                )));
            }
            None => std::thread::sleep(Duration::from_millis(2)),
        }
    };

    match status.code().unwrap_or(-1) {
        H_OK => Ok(()),
        H_NO_PROCESS => Err(Error::ProcessNotFound(pid)),
        H_NO_CONSOLE => Err(Error::NoConsole(pid)),
        H_ACCESS_DENIED => Err(Error::Os(std::io::Error::from_raw_os_error(
            ERROR_ACCESS_DENIED as i32,
        ))),
        code => Err(Error::HelperFailed {
            code,
            what: task.describe(),
        }),
    }
}

// ---------------------------------------------------------------------------
// Windows (GUI)
// ---------------------------------------------------------------------------

struct EnumState {
    pid: u32,
    found: Option<usize>,
}

unsafe extern "system" fn enum_proc(hwnd: HWND, lparam: LPARAM) -> i32 {
    let state = &mut *(lparam as *mut EnumState);
    let mut window_pid = 0u32;
    GetWindowThreadProcessId(hwnd, &mut window_pid);
    if window_pid == state.pid && is_main_window(hwnd) {
        state.found = Some(hwnd as usize);
        return FALSE; // stop enumerating
    }
    TRUE
}

fn is_main_window(hwnd: HWND) -> bool {
    // SAFETY: plain FFI calls.
    unsafe { GetWindow(hwnd, GW_OWNER).is_null() && IsWindowVisible(hwnd) != 0 }
}

/// Class of the top-level windows that host packaged (UWP) apps. They are owned by
/// `ApplicationFrameHost.exe`, not by the app, which keeps its content in a child window of
/// the frame.
const UWP_FRAME_CLASS: &str = "ApplicationFrameWindow";

fn has_class(hwnd: HWND, expected: &str) -> bool {
    let mut buf = [0u16; 64];
    // SAFETY: `buf` is writable for its whole length.
    let len = unsafe { GetClassNameW(hwnd, buf.as_mut_ptr(), buf.len() as i32) };
    len > 0 && String::from_utf16_lossy(&buf[..len as usize]) == expected
}

struct FrameState {
    pid: u32,
    owns_child: bool,
    frame: Option<usize>,
}

unsafe extern "system" fn child_proc(hwnd: HWND, lparam: LPARAM) -> i32 {
    let state = &mut *(lparam as *mut FrameState);
    let mut window_pid = 0u32;
    GetWindowThreadProcessId(hwnd, &mut window_pid);
    if window_pid == state.pid {
        state.owns_child = true;
        return FALSE; // stop enumerating children
    }
    TRUE
}

unsafe extern "system" fn frame_proc(hwnd: HWND, lparam: LPARAM) -> i32 {
    let state = &mut *(lparam as *mut FrameState);
    if has_class(hwnd, UWP_FRAME_CLASS) {
        state.owns_child = false;
        EnumChildWindows(hwnd, Some(child_proc), lparam);
        if state.owns_child {
            state.frame = Some(hwnd as usize);
            return FALSE; // stop enumerating
        }
    }
    TRUE
}

/// Returns the main window of process `pid`.
///
/// For a classic application that is its visible, unowned top-level window. A packaged (UWP)
/// app has no such window: the frame that hosts it belongs to `ApplicationFrameHost.exe`, so
/// the frame containing a window of `pid` is returned instead. Closing that frame closes the
/// app, exactly like its close button.
pub fn main_window(pid: u32) -> Result<WindowHandle> {
    let mut state = EnumState { pid, found: None };
    // SAFETY: `state` outlives the synchronous EnumWindows call.
    unsafe { EnumWindows(Some(enum_proc), &mut state as *mut EnumState as LPARAM) };
    if let Some(window) = state.found {
        return Ok(window);
    }

    let mut frames = FrameState {
        pid,
        owns_child: false,
        frame: None,
    };
    // SAFETY: `frames` outlives the synchronous enumeration.
    unsafe { EnumWindows(Some(frame_proc), &mut frames as *mut FrameState as LPARAM) };
    frames.frame.ok_or(Error::NoWindow(pid))
}

/// Sends `WM_CLOSE` to `window`, as if its close button was pressed.
///
/// With `wait` the call blocks until the window procedure handled the message (for example
/// until the user answers a "save changes?" dialog).
pub fn close_window(window: WindowHandle, wait: bool) -> Result<()> {
    let hwnd = window as HWND;
    let msg = WM_CLOSE;
    // SAFETY: plain FFI calls; an invalid handle just makes them fail.
    let ok = unsafe {
        if wait {
            SendMessageW(hwnd, msg, 0, 0) == 0
        } else {
            PostMessageW(hwnd, msg, 0, 0) != 0
        }
    };
    if ok {
        Ok(())
    } else {
        Err(Error::Os(std::io::Error::last_os_error()))
    }
}

// ---------------------------------------------------------------------------
// Public platform API
// ---------------------------------------------------------------------------

pub(crate) fn stop(pid: u32, method: Method) -> Result<()> {
    match method {
        Method::CtrlC => dispatch(pid, Task::Ctrl(CTRL_C_EVENT)),
        Method::CtrlBreak => dispatch(pid, Task::Ctrl(CTRL_BREAK_EVENT)),
        Method::CloseWindow => {
            if !is_running(pid) {
                return Err(Error::ProcessNotFound(pid));
            }
            close_window(main_window(pid)?, false)
        }
    }
}

pub(crate) fn send_message(pid: u32, msg: &str) -> Result<()> {
    dispatch(pid, Task::Message(msg.to_owned()))
}

pub(crate) fn kill(pid: u32) -> Result<()> {
    // An exited process whose handle is still open elsewhere can be opened
    // but not terminated (ERROR_ACCESS_DENIED), which would be misleading.
    if !is_running(pid) {
        return Err(Error::ProcessNotFound(pid));
    }
    // SAFETY: plain FFI calls; the handle is closed on every path.
    unsafe {
        let handle = OpenProcess(PROCESS_TERMINATE, FALSE, pid);
        if handle.is_null() {
            return Err(match GetLastError() {
                ERROR_INVALID_PARAMETER => Error::ProcessNotFound(pid),
                code => Error::Os(std::io::Error::from_raw_os_error(code as i32)),
            });
        }
        let ok = TerminateProcess(handle, 1);
        let err = std::io::Error::last_os_error();
        CloseHandle(handle);
        if ok == 0 {
            return Err(Error::Os(err));
        }
    }
    Ok(())
}

pub(crate) fn is_running(pid: u32) -> bool {
    // SAFETY: plain FFI calls; the handle is closed on every path.
    unsafe {
        let handle = OpenProcess(
            PROCESS_QUERY_LIMITED_INFORMATION | PROCESS_SYNCHRONIZE,
            FALSE,
            pid,
        );
        if handle.is_null() {
            // Access denied means the process exists but is not ours.
            return GetLastError() == ERROR_ACCESS_DENIED;
        }
        let running = WaitForSingleObject(handle, 0) == WAIT_TIMEOUT;
        CloseHandle(handle);
        running
    }
}
