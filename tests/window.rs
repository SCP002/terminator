//! Closing a GUI window (Windows). Needs an interactive desktop session.

#![cfg(windows)]

mod common;

use common::flags::CREATE_NO_WINDOW;
use common::*;

#[test]
fn close_window_posts_wm_close() {
    let scratch = Scratch::new("win");
    let mut victim = spawn_victim(&scratch, "victim", &["window"], CREATE_NO_WINDOW);
    assert!(victim.wait_line("READY"));

    let pid = victim.pid();
    assert!(terminator::main_window(pid).is_ok(), "no main window found");

    let run = run_cli(
        CREATE_NO_WINDOW,
        &["stop", &pid.to_string(), "--method", "close-window"],
    );
    assert_eq!(run.code, 0, "{}", run.stderr);
    assert!(victim.wait_exit(TIMEOUT));
    assert!(victim.lines().iter().any(|l| l == "WM_CLOSE"));
}

#[test]
fn console_only_process_has_no_main_window() {
    let scratch = Scratch::new("nowin");
    let victim = spawn_victim(&scratch, "victim", &["wait"], CREATE_NO_WINDOW);
    assert!(victim.wait_line("READY"));
    assert!(matches!(
        terminator::main_window(victim.pid()),
        Err(terminator::Error::NoWindow(_))
    ));
}

/// Stops the process on drop, so a failed assertion does not leave the app open.
struct KillOnDrop(u32);

impl Drop for KillOnDrop {
    fn drop(&mut self) {
        if terminator::is_running(self.0) {
            let _ = terminator::kill(self.0);
        }
    }
}

/// A packaged (UWP) app has no top-level window of its own: its content lives in a child
/// window of a frame owned by `ApplicationFrameHost.exe`. Closing the app by its pid has to
/// find that frame.
///
/// Opens the Windows Calculator on the desktop and needs it to be installed and not running,
/// hence `#[ignore]`:
///
/// ```text
/// cargo test --test window -- --ignored
/// ```
#[test]
#[ignore = "opens the Windows Calculator on the desktop"]
fn uwp_app_is_closed_through_its_frame_window() {
    use std::time::{Duration, Instant};
    use sysinfo::{ProcessesToUpdate, System};

    fn calculator_pids() -> Vec<u32> {
        let mut sys = System::new();
        sys.refresh_processes(ProcessesToUpdate::All, true);
        sys.processes()
            .iter()
            .filter(|(_, process)| process.name().eq_ignore_ascii_case("CalculatorApp.exe"))
            .map(|(pid, _)| pid.as_u32())
            .collect()
    }

    fn wait_until(what: &str, mut done: impl FnMut() -> bool) {
        let start = Instant::now();
        while !done() {
            assert!(
                start.elapsed() < Duration::from_secs(15),
                "timed out waiting for {what}"
            );
            std::thread::sleep(Duration::from_millis(100));
        }
    }

    assert!(
        calculator_pids().is_empty(),
        "Calculator is already running; close it first"
    );
    std::process::Command::new("explorer.exe")
        .arg(r"shell:AppsFolder\Microsoft.WindowsCalculator_8wekyb3d8bbwe!App")
        .status()
        .expect("cannot start explorer.exe");

    let mut pid = 0;
    wait_until("Calculator to start", || {
        pid = calculator_pids().first().copied().unwrap_or(0);
        pid != 0
    });
    let _cleanup = KillOnDrop(pid);
    wait_until("the frame window of Calculator", || {
        terminator::main_window(pid).is_ok()
    });

    let run = run_cli(
        CREATE_NO_WINDOW,
        &["stop", &pid.to_string(), "--method", "close-window"],
    );
    assert_eq!(run.code, 0, "{}", run.stderr);
    assert!(
        terminator::wait_for_stop(pid, Some(TIMEOUT)),
        "Calculator ignored WM_CLOSE"
    );
}
