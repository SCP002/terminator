//! A small process that reports what happens to it. Used as the target in
//! the end-to-end tests and in the other examples.
//!
//! ```text
//! victim --out <FILE> <MODE> [ARGS]
//!
//! MODES
//!   wait               Report READY, then the first stop request and exit
//!   readline           Report READY, read one line from the console/stdin
//!   window             (Windows) Show a window, report WM_CLOSE
//!   chain <N>          Spawn a chain of N nested victims for tree tests
//!   supervisor <CLI> <METHOD> <TARGET_OUT>
//!                      (Windows) Share a console with a `wait` victim and
//!                      run the CLI against it from inside that console
//!   exit               Exit immediately
//! ```
//!
//! Everything it observes is appended, one line at a time, to `--out`:
//! `READY`, `PID:<n>`, `CTRL_C`, `CTRL_BREAK`, `SIGINT`, `SIGTERM`,
//! `WM_CLOSE`, `LINE:<text>`, `CLI_EXIT:<code>`, `DONE`.

use std::fs::OpenOptions;
use std::io::Write;
use std::path::PathBuf;
use std::sync::OnceLock;
use std::time::Duration;

static OUT: OnceLock<PathBuf> = OnceLock::new();

fn record(line: &str) {
    if let Some(path) = OUT.get() {
        if let Ok(mut f) = OpenOptions::new().create(true).append(true).open(path) {
            let _ = writeln!(f, "{line}");
        }
    }
}

fn main() {
    let mut args = std::env::args().skip(1);
    let mut mode = None;
    let mut rest = Vec::new();
    while let Some(a) = args.next() {
        match a.as_str() {
            "--out" => {
                let path = args.next().expect("--out needs a value");
                OUT.set(PathBuf::from(path)).unwrap();
            }
            _ if mode.is_none() => mode = Some(a),
            _ => rest.push(a),
        }
    }
    match mode.as_deref() {
        Some("wait") => wait_mode(),
        Some("readline") => readline_mode(),
        Some("chain") => chain_mode(rest[0].parse().expect("chain needs a number")),
        Some("exit") => {}
        #[cfg(windows)]
        Some("window") => win::window_mode(),
        #[cfg(windows)]
        Some("supervisor") => win::supervisor_mode(&rest),
        other => {
            eprintln!("victim: unknown mode {other:?}");
            std::process::exit(64);
        }
    }
}

fn wait_mode() -> ! {
    install_stop_handlers();
    record("READY");
    loop {
        std::thread::sleep(Duration::from_millis(20));
        poll_stop_flag();
    }
}

fn readline_mode() {
    record("READY");
    let line = read_input_line();
    record(&format!("LINE:{}", line.trim_end()));
}

/// Read one line. On Windows read the console itself: standard input may be
/// redirected.
#[cfg(windows)]
fn read_input_line() -> String {
    {
        use std::os::windows::io::AsRawHandle;
        use windows_sys::Win32::System::Console::ReadConsoleW;
        let con = OpenOptions::new()
            .read(true)
            .write(true)
            .open("CONIN$")
            .expect("open CONIN$");
        // ReadConsoleW returns UTF-16; ReadFile would return bytes in the
        // console's OEM code page and mangle anything non-ASCII.
        let mut buf = [0u16; 1024];
        let mut read = 0u32;
        // SAFETY: `buf` and `read` outlive the call; the handle is valid.
        let ok = unsafe {
            ReadConsoleW(
                con.as_raw_handle() as _,
                buf.as_mut_ptr().cast(),
                buf.len() as u32,
                &mut read,
                std::ptr::null(),
            )
        };
        assert!(ok != 0, "ReadConsoleW failed");
        String::from_utf16_lossy(&buf[..read as usize])
    }
}

#[cfg(unix)]
fn read_input_line() -> String {
    let mut line = String::new();
    std::io::stdin().read_line(&mut line).unwrap();
    line
}

fn chain_mode(n: u32) {
    record(&format!("PID:{}", std::process::id()));
    if n == 0 {
        wait_mode();
    }
    let out = OUT.get().expect("--out is required for chain");
    let exe = std::env::current_exe().unwrap();
    let mut child = std::process::Command::new(exe)
        .arg("--out")
        .arg(out)
        .arg("chain")
        .arg((n - 1).to_string())
        .spawn()
        .unwrap();
    let _ = child.wait();
}

/// Poll `out` until it contains `needle` (used by the supervisor).
#[cfg(windows)]
fn wait_for_line(path: &std::path::Path, needle: &str, timeout: Duration) -> bool {
    let start = std::time::Instant::now();
    while start.elapsed() < timeout {
        if std::fs::read_to_string(path).is_ok_and(|s| s.lines().any(|l| l == needle)) {
            return true;
        }
        std::thread::sleep(Duration::from_millis(20));
    }
    false
}

// ---------------------------------------------------------------------------
// Stop detection
// ---------------------------------------------------------------------------

#[cfg(windows)]
fn install_stop_handlers() {
    use windows_sys::Win32::System::Console::SetConsoleCtrlHandler;
    // SAFETY: plain FFI calls. Re-enable Ctrl+C in case it was inherited as
    // disabled, then install our handler.
    unsafe {
        SetConsoleCtrlHandler(None, 0);
        SetConsoleCtrlHandler(Some(win::on_ctrl), 1);
    }
}

#[cfg(windows)]
fn poll_stop_flag() {}

#[cfg(unix)]
static STOP_SIGNAL: std::sync::atomic::AtomicI32 = std::sync::atomic::AtomicI32::new(0);

#[cfg(unix)]
extern "C" fn on_signal(sig: libc::c_int) {
    STOP_SIGNAL.store(sig, std::sync::atomic::Ordering::SeqCst);
}

#[cfg(unix)]
fn install_stop_handlers() {
    // SAFETY: the handler only stores into an atomic.
    unsafe {
        libc::signal(libc::SIGINT, on_signal as *const () as libc::sighandler_t);
        libc::signal(libc::SIGTERM, on_signal as *const () as libc::sighandler_t);
    }
}

#[cfg(unix)]
fn poll_stop_flag() {
    match STOP_SIGNAL.load(std::sync::atomic::Ordering::SeqCst) {
        0 => {}
        libc::SIGINT => {
            record("SIGINT");
            std::process::exit(0);
        }
        _ => {
            record("SIGTERM");
            std::process::exit(0);
        }
    }
}

// ---------------------------------------------------------------------------
// Windows-only modes
// ---------------------------------------------------------------------------

#[cfg(windows)]
mod win {
    use super::*;
    use windows_sys::Win32::Foundation::{HWND, LPARAM, LRESULT, WPARAM};
    use windows_sys::Win32::System::Console::{
        SetConsoleCtrlHandler, CTRL_BREAK_EVENT, CTRL_C_EVENT,
    };
    use windows_sys::Win32::System::LibraryLoader::GetModuleHandleW;
    use windows_sys::Win32::UI::WindowsAndMessaging::{
        CreateWindowExW, DefWindowProcW, DestroyWindow, DispatchMessageW, GetMessageW,
        PostQuitMessage, RegisterClassW, ShowWindow, TranslateMessage, MSG, SW_SHOWNOACTIVATE,
        WM_CLOSE, WM_DESTROY, WNDCLASSW, WS_OVERLAPPEDWINDOW,
    };

    pub unsafe extern "system" fn on_ctrl(ctrl_type: u32) -> i32 {
        let name = match ctrl_type {
            CTRL_C_EVENT => "CTRL_C",
            CTRL_BREAK_EVENT => "CTRL_BREAK",
            _ => return 0,
        };
        record(name);
        std::process::exit(0);
    }

    unsafe extern "system" fn wndproc(hwnd: HWND, msg: u32, wp: WPARAM, lp: LPARAM) -> LRESULT {
        match msg {
            WM_CLOSE => {
                record("WM_CLOSE");
                DestroyWindow(hwnd);
                0
            }
            WM_DESTROY => {
                PostQuitMessage(0);
                0
            }
            _ => DefWindowProcW(hwnd, msg, wp, lp),
        }
    }

    pub fn window_mode() {
        let class: Vec<u16> = "TerminatorVictim\0".encode_utf16().collect();
        let title: Vec<u16> = "terminator victim\0".encode_utf16().collect();
        // SAFETY: standard window class registration and message loop.
        unsafe {
            let hinst = GetModuleHandleW(std::ptr::null());
            let mut wc: WNDCLASSW = std::mem::zeroed();
            wc.lpfnWndProc = Some(wndproc);
            wc.hInstance = hinst;
            wc.lpszClassName = class.as_ptr();
            RegisterClassW(&wc);
            let hwnd = CreateWindowExW(
                0,
                class.as_ptr(),
                title.as_ptr(),
                WS_OVERLAPPEDWINDOW,
                100,
                100,
                320,
                120,
                std::ptr::null_mut(),
                std::ptr::null_mut(),
                hinst,
                std::ptr::null(),
            );
            ShowWindow(hwnd, SW_SHOWNOACTIVATE);
            record("READY");
            let mut msg: MSG = std::mem::zeroed();
            while GetMessageW(&mut msg, std::ptr::null_mut(), 0, 0) > 0 {
                TranslateMessage(&msg);
                DispatchMessageW(&msg);
            }
        }
    }

    /// Runs inside console K together with a `wait` victim and the CLI, so
    /// the library must pick the *direct* route.
    pub fn supervisor_mode(rest: &[String]) {
        let (cli, method, target_out) = (&rest[0], &rest[1], PathBuf::from(&rest[2]));
        // The event is generated for the whole console, so survive it.
        // SAFETY: handler returns TRUE for both events.
        unsafe extern "system" fn survive(_: u32) -> i32 {
            1
        }
        unsafe { SetConsoleCtrlHandler(Some(survive), 1) };

        let exe = std::env::current_exe().unwrap();
        // No creation flags: the child inherits this console.
        let mut target = std::process::Command::new(exe)
            .arg("--out")
            .arg(&target_out)
            .arg("wait")
            .spawn()
            .unwrap();
        if !wait_for_line(&target_out, "READY", Duration::from_secs(10)) {
            record("TARGET_NOT_READY");
            let _ = target.kill();
            let _ = target.wait();
            return;
        }
        // Piped output does not change the console the CLI is attached to.
        let output = std::process::Command::new(cli)
            .env("TERMINATOR_DEBUG", "1")
            .args(["stop", &target.id().to_string(), "--method", method])
            .args(["--wait", "5000"])
            .output()
            .unwrap();
        for line in String::from_utf8_lossy(&output.stderr).lines() {
            record(&format!("CLI_LOG:{line}"));
        }
        record(&format!("CLI_EXIT:{}", output.status.code().unwrap_or(-1)));
        let _ = target.wait();
        record("DONE");
    }
}
