//! Stop processes gracefully, even on Windows.
//!
//! On Windows this sends real `CTRL_C_EVENT` / `CTRL_BREAK_EVENT` console
//! signals (or closes a GUI window) to an arbitrary process, and can write to
//! its standard input. Unlike the original Go implementation it does not ship
//! and drop a separate "proxy" executable into a temporary folder.
//!
//! Windows can attach a process to only one console at a time, so reaching a
//! foreign console sometimes needs a short-lived detached process. The crate
//! picks the cheapest route automatically:
//!
//! 1. **Direct**: the target shares the caller's console, nothing to attach.
//! 2. **In-process**: the caller has no console, so attach, send, detach.
//! 3. **Helper**: the caller has its own console; a detached helper attaches
//!    to the target console instead. By default the helper is *the current
//!    executable re-launched with a marker argument*, intercepted before
//!    `main` by a C runtime initializer that this crate registers itself. No
//!    call from `main` is needed ([`run_helper_if_requested`] exists only for
//!    unusual setups). When this code lives in a DLL, point the crate to an
//!    executable that links it with [`set_helper`].
//!
//! On Unix it sends POSIX signals and can write to a process' controlling TTY.
//!
//! Set `TERMINATOR_DEBUG=1` to print the chosen route to stderr.
//!
//! # Quick start
//!
//! ```no_run
//! use terminator::{stop, Method};
//! # fn main() -> terminator::Result<()> {
//! stop(1234, Method::CtrlC)?;
//! # Ok(())
//! # }
//! ```

mod error;
mod tree;

pub use error::{Error, Result};
pub use tree::child_tree;

use std::time::{Duration, Instant};

#[cfg_attr(windows, path = "platform/windows.rs")]
#[cfg_attr(unix, path = "platform/posix.rs")]
mod platform;

#[cfg(windows)]
pub use platform::{
    close_window, main_window, run_helper_if_requested, set_helper, Helper, WindowHandle,
};

/// Does nothing on this platform. Exists so cross-platform code can call it
/// unconditionally; see the Windows documentation for the real behavior.
#[cfg(unix)]
pub fn run_helper_if_requested() {}

/// A cross-platform way to ask a process to stop.
///
/// POSIX has no equivalent of the Windows console control events or window
/// messages, so there every method sends `SIGINT`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Method {
    /// Windows: `CTRL_C_EVENT` to the target console. POSIX: `SIGINT`.
    ///
    /// Reaches every process attached to that console which has not disabled
    /// Ctrl+C handling.
    CtrlC,
    /// Windows: `CTRL_BREAK_EVENT` to the target console. POSIX: `SIGINT`.
    ///
    /// Unlike Ctrl+C it cannot be disabled with `SetConsoleCtrlHandler(NULL,
    /// TRUE)`, so it is more reliable for processes that masked Ctrl+C.
    CtrlBreak,
    /// Windows: posts `WM_CLOSE` to the main GUI window of the process; for a packaged (UWP)
    /// app, to the frame window that hosts it. POSIX: `SIGINT`.
    CloseWindow,
}

/// Ask the process with id `pid` to stop using the given [`Method`].
///
/// Returns `Ok(())` once the request was delivered. Delivery does not
/// guarantee the process handled it or exited; use [`wait_for_stop`].
pub fn stop(pid: u32, method: Method) -> Result<()> {
    platform::stop(pid, method)
}

/// Forcibly terminate the process with id `pid`.
///
/// Windows: `TerminateProcess`. POSIX: `SIGKILL`. The process gets no chance
/// to clean up; prefer [`stop`] when a graceful shutdown is possible.
pub fn kill(pid: u32) -> Result<()> {
    platform::kill(pid)
}

/// Write `msg` to the standard input of the process with id `pid`.
///
/// Useful to answer prompts such as `Terminate batch job (Y/N)?`.
///
/// On Windows the message is injected into the target console's input buffer
/// and must end with `"\r\n"` to be submitted. On Unix it is pushed into the
/// process' controlling terminal with `TIOCSTI`, which usually requires root
/// (and is disabled by some kernels); use `"\n"` on Linux, `"\r"` on macOS.
pub fn send_message(pid: u32, msg: &str) -> Result<()> {
    platform::send_message(pid, msg)
}

/// Returns `true` while the process with id `pid` is running.
pub fn is_running(pid: u32) -> bool {
    platform::is_running(pid)
}

/// Block until the process `pid` is no longer running or `timeout` elapses
/// (forever if `None`). Returns `true` if the process stopped.
pub fn wait_for_stop(pid: u32, timeout: Option<Duration>) -> bool {
    let start = Instant::now();
    loop {
        if !platform::is_running(pid) {
            return true;
        }
        if let Some(t) = timeout {
            if start.elapsed() >= t {
                return false;
            }
        }
        std::thread::sleep(Duration::from_millis(25));
    }
}
