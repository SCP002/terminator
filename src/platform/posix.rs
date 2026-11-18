//! POSIX backend (Linux, macOS, other Unix).

use std::fs::OpenOptions;
use std::os::unix::io::AsRawFd;
use std::process::Command;

use sysinfo::{Pid, ProcessRefreshKind, ProcessStatus, ProcessesToUpdate, System};

use crate::{Error, Method, Result};

fn os_error() -> Error {
    Error::Os(std::io::Error::last_os_error())
}

fn send_signal(pid: u32, sig: libc::c_int) -> Result<()> {
    let pid_t = pid as libc::pid_t;
    // SAFETY: plain FFI call. Guard against pid values that would address a
    // process group (0) or every process we may signal (negative).
    if pid_t <= 0 {
        return Err(Error::ProcessNotFound(pid));
    }
    if unsafe { libc::kill(pid_t, sig) } == 0 {
        return Ok(());
    }
    match std::io::Error::last_os_error().raw_os_error() {
        Some(libc::ESRCH) => Err(Error::ProcessNotFound(pid)),
        _ => Err(os_error()),
    }
}

pub(crate) fn stop(pid: u32, _method: Method) -> Result<()> {
    // Every method maps to SIGINT on POSIX.
    send_signal(pid, libc::SIGINT)
}

pub(crate) fn kill(pid: u32) -> Result<()> {
    send_signal(pid, libc::SIGKILL)
}

pub(crate) fn is_running(pid: u32) -> bool {
    let pid_t = pid as libc::pid_t;
    if pid_t <= 0 {
        return false;
    }
    // SAFETY: signal 0 only checks for existence and permission.
    let exists = unsafe { libc::kill(pid_t, 0) } == 0
        || std::io::Error::last_os_error().raw_os_error() == Some(libc::EPERM);
    if !exists {
        return false;
    }
    // A zombie has exited but not been reaped by its parent yet.
    let mut sys = System::new();
    let target = Pid::from_u32(pid);
    sys.refresh_processes_specifics(
        ProcessesToUpdate::Some(&[target]),
        false,
        ProcessRefreshKind::new(),
    );
    !matches!(
        sys.process(target).map(|p| p.status()),
        Some(ProcessStatus::Zombie) | None
    )
}

/// Returns the TTY device of process `pid` (for example `/dev/pts/3`).
fn terminal_of(pid: u32) -> Result<String> {
    // `ps` is the one portable way to learn the controlling terminal on both
    // Linux (`pts/3`) and macOS (`ttys003`).
    let out = Command::new("ps")
        .args(["-o", "tty=", "-p", &pid.to_string()])
        .output()
        .map_err(Error::Os)?;
    let tty = String::from_utf8_lossy(&out.stdout).trim().to_owned();
    if !out.status.success() || tty.is_empty() {
        return Err(Error::ProcessNotFound(pid));
    }
    if tty == "?" || tty == "??" || tty == "-" {
        return Err(Error::NoConsole(pid));
    }
    Ok(format!("/dev/{tty}"))
}

pub(crate) fn send_message(pid: u32, msg: &str) -> Result<()> {
    let device = terminal_of(pid)?;
    let tty = OpenOptions::new()
        .write(true)
        .open(&device)
        .map_err(Error::Os)?;
    for byte in msg.bytes() {
        let ch = byte as libc::c_char;
        // SAFETY: TIOCSTI takes a pointer to one character.
        if unsafe { libc::ioctl(tty.as_raw_fd(), libc::TIOCSTI as _, &ch) } != 0 {
            return Err(os_error());
        }
    }
    Ok(())
}
