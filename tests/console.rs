//! Windows console scenarios. Each test runs the real CLI in a chosen console
//! state against a victim in its own hidden console, then checks both *what
//! the victim saw* and *which route the library took*.
//!
//! | caller console state            | target                | route     |
//! |---------------------------------|-----------------------|-----------|
//! | own console (CREATE_NO_WINDOW)  | another console       | Helper    |
//! | none (DETACHED_PROCESS)         | another console       | InProcess |
//! | shared with the target          | same console          | Direct    |

#![cfg(windows)]

mod common;

use common::flags::{CREATE_NO_WINDOW, DETACHED_PROCESS};
use common::*;

/// Stop a victim living in its own console with the CLI started in
/// `caller_flags` state; assert the route and what the victim observed.
fn stop_case(case: &str, caller_flags: u32, method: &str, route: &str, expect: &str) {
    let scratch = Scratch::new(case);
    let mut victim = spawn_victim(&scratch, "victim", &["wait"], CREATE_NO_WINDOW);
    assert!(victim.wait_line("READY"), "victim never became ready");

    let pid = victim.pid().to_string();
    let run = run_cli(
        caller_flags,
        &["stop", &pid, "--method", method, "--wait", "5000"],
    );

    assert_eq!(run.code, 0, "CLI failed: {}", run.stderr);
    assert!(
        run.stderr.contains(&format!("route={route}")),
        "expected route {route}, CLI said: {}",
        run.stderr
    );
    assert!(victim.wait_exit(TIMEOUT), "victim did not exit");
    assert!(
        victim.lines().iter().any(|l| l == expect),
        "victim log: {:?}",
        victim.lines()
    );
}

#[test]
fn ctrl_c_via_helper() {
    stop_case("c-h", CREATE_NO_WINDOW, "ctrl-c", "Helper", "CTRL_C");
}

#[test]
fn ctrl_break_via_helper() {
    stop_case(
        "b-h",
        CREATE_NO_WINDOW,
        "ctrl-break",
        "Helper",
        "CTRL_BREAK",
    );
}

#[test]
fn ctrl_c_in_process() {
    stop_case("c-i", DETACHED_PROCESS, "ctrl-c", "InProcess", "CTRL_C");
}

#[test]
fn ctrl_break_in_process() {
    stop_case(
        "b-i",
        DETACHED_PROCESS,
        "ctrl-break",
        "InProcess",
        "CTRL_BREAK",
    );
}

/// The supervisor, a `wait` victim and the CLI all share one console, so the
/// event hits everyone and the library must not attach anywhere.
fn direct_case(case: &str, method: &str, expect: &str) {
    let scratch = Scratch::new(case);
    let target_log = scratch.file("target.log");
    let cli = cli_exe();
    let supervisor = spawn_victim(
        &scratch,
        "supervisor",
        &[
            "supervisor",
            cli.to_str().unwrap(),
            method,
            target_log.to_str().unwrap(),
        ],
        CREATE_NO_WINDOW,
    );
    assert!(
        supervisor.wait_line("DONE"),
        "log: {:?}",
        supervisor.lines()
    );

    let lines = supervisor.lines();
    assert!(lines.iter().any(|l| l == "CLI_EXIT:0"), "log: {lines:?}");
    assert!(
        lines.iter().any(|l| l.contains("route=Direct")),
        "log: {lines:?}"
    );
    assert!(wait_for_line(&target_log, expect, TIMEOUT));
}

#[test]
fn ctrl_c_direct() {
    direct_case("c-d", "ctrl-c", "CTRL_C");
}

#[test]
fn ctrl_break_direct() {
    direct_case("b-d", "ctrl-break", "CTRL_BREAK");
}

/// Answer an input prompt in a foreign console.
fn message_case(case: &str, caller_flags: u32, route: &str) {
    let scratch = Scratch::new(case);
    let mut victim = spawn_victim(&scratch, "victim", &["readline"], CREATE_NO_WINDOW);
    assert!(victim.wait_line("READY"));

    let pid = victim.pid().to_string();
    let run = run_cli(caller_flags, &["send", &pid, "hello world", "--newline"]);

    assert_eq!(run.code, 0, "CLI failed: {}", run.stderr);
    assert!(
        run.stderr.contains(&format!("route={route}")),
        "{}",
        run.stderr
    );
    assert!(victim.wait_exit(TIMEOUT));
    assert!(
        victim.lines().iter().any(|l| l == "LINE:hello world"),
        "victim log: {:?}",
        victim.lines()
    );
}

#[test]
fn message_via_helper() {
    message_case("m-h", CREATE_NO_WINDOW, "Helper");
}

#[test]
fn message_in_process() {
    message_case("m-i", DETACHED_PROCESS, "InProcess");
}

#[test]
fn message_with_non_ascii_text_survives_the_helper_protocol() {
    let scratch = Scratch::new("m-u");
    let mut victim = spawn_victim(&scratch, "victim", &["readline"], CREATE_NO_WINDOW);
    assert!(victim.wait_line("READY"));
    let pid = victim.pid().to_string();
    let run = run_cli(
        CREATE_NO_WINDOW,
        &["send", &pid, "Привет, мир", "--newline"],
    );
    assert_eq!(run.code, 0, "{}", run.stderr);
    assert!(victim.wait_exit(TIMEOUT));
    assert!(victim.lines().iter().any(|l| l == "LINE:Привет, мир"));
}

/// What a library loaded from a DLL has to do: name an executable that links
/// terminator (here the CLI) instead of re-launching the host.
#[test]
fn external_helper_executable_works() {
    let stop_child = example_exe("stop_child");
    let cli = cli_exe();
    let run = run_exe(
        &stop_child,
        CREATE_NO_WINDOW,
        &["--hidden", "--helper", cli.to_str().unwrap()],
    );
    assert_eq!(run.code, 0, "stdout: {} stderr: {}", run.stdout, run.stderr);
    assert!(run.stderr.contains("route=Helper"), "{}", run.stderr);
    assert!(run.stdout.contains("exited gracefully"), "{}", run.stdout);
}

/// A helper executable that does not link terminator must not be trusted:
/// the failure is reported instead of anything being silently assumed.
#[test]
fn helper_that_ignores_the_protocol_is_reported() {
    let stop_child = example_exe("stop_child");
    let not_a_helper = victim_exe(); // does not link terminator; exits 64
    let run = run_exe(
        &stop_child,
        CREATE_NO_WINDOW,
        &["--hidden", "--helper", not_a_helper.to_str().unwrap()],
    );
    assert_eq!(run.code, 1, "stdout: {} stderr: {}", run.stdout, run.stderr);
    assert!(
        run.stderr.contains("helper process failed"),
        "{}",
        run.stderr
    );
    assert!(run.stderr.contains("exit code 64"), "{}", run.stderr);
}

#[test]
fn process_without_console_is_reported() {
    let scratch = Scratch::new("noc");
    let victim = spawn_victim(&scratch, "victim", &["wait"], DETACHED_PROCESS);
    assert!(victim.wait_line("READY"));
    let pid = victim.pid().to_string();
    let run = run_cli(CREATE_NO_WINDOW, &["stop", &pid]);
    assert_eq!(run.code, 1);
    assert!(run.stderr.contains("no console"), "{}", run.stderr);
}

#[test]
fn dead_process_is_reported() {
    let scratch = Scratch::new("dead");
    let mut victim = spawn_victim(&scratch, "victim", &["exit"], CREATE_NO_WINDOW);
    assert!(victim.wait_exit(TIMEOUT));
    let pid = victim.pid().to_string();
    let run = run_cli(CREATE_NO_WINDOW, &["stop", &pid]);
    assert_eq!(run.code, 1);
    assert!(run.stderr.contains("does not exist"), "{}", run.stderr);
}
