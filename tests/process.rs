//! Platform-independent behavior: kill, wait, process tree, error reporting.

mod common;

use std::time::Duration;

use common::*;
use terminator::{child_tree, is_running, kill, stop, wait_for_stop, Error, Method};

#[cfg(windows)]
const NEW_CONSOLE: u32 = common::flags::CREATE_NO_WINDOW;
#[cfg(not(windows))]
const NEW_CONSOLE: u32 = 0;

#[test]
fn kill_terminates_the_process() {
    let scratch = Scratch::new("kill");
    let victim = spawn_victim(&scratch, "victim", &["wait"], NEW_CONSOLE);
    assert!(victim.wait_line("READY"));
    assert!(is_running(victim.pid()));

    kill(victim.pid()).unwrap();
    assert!(wait_for_stop(victim.pid(), Some(TIMEOUT)));
    assert!(!is_running(victim.pid()));
}

#[test]
fn wait_for_stop_times_out_on_a_live_process() {
    let scratch = Scratch::new("wait");
    let victim = spawn_victim(&scratch, "victim", &["wait"], NEW_CONSOLE);
    assert!(victim.wait_line("READY"));
    assert!(!wait_for_stop(
        victim.pid(),
        Some(Duration::from_millis(200))
    ));
}

#[test]
fn stopping_a_dead_process_reports_process_not_found() {
    let scratch = Scratch::new("dead");
    let mut victim = spawn_victim(&scratch, "victim", &["exit"], NEW_CONSOLE);
    assert!(victim.wait_exit(TIMEOUT));
    assert!(matches!(
        stop(victim.pid(), Method::CtrlC),
        Err(Error::ProcessNotFound(_))
    ));
    assert!(matches!(kill(victim.pid()), Err(Error::ProcessNotFound(_))));
}

/// root -> middle -> leaf, started by `victim chain 2`.
fn spawn_chain(scratch: &Scratch) -> (Victim, [u32; 3]) {
    let victim = spawn_victim(scratch, "chain", &["chain", "2"], NEW_CONSOLE);
    assert!(victim.wait_line("READY"), "log: {:?}", victim.lines());
    let pids: Vec<u32> = victim
        .lines()
        .iter()
        .filter_map(|l| l.strip_prefix("PID:")?.parse().ok())
        .collect();
    assert_eq!(pids.len(), 3, "log: {:?}", victim.lines());
    (victim, [pids[0], pids[1], pids[2]])
}

#[test]
fn child_tree_lists_descendants_deepest_first() {
    let scratch = Scratch::new("tree");
    let (_victim, [root, middle, leaf]) = spawn_chain(&scratch);

    assert_eq!(child_tree(root, false), vec![leaf, middle]);
    assert_eq!(child_tree(root, true), vec![leaf, middle, root]);
    assert_eq!(child_tree(leaf, false), Vec::<u32>::new());
}

#[test]
fn killing_a_tree_bottom_up_stops_every_process() {
    let scratch = Scratch::new("killtree");
    let (_victim, [root, middle, leaf]) = spawn_chain(&scratch);

    for pid in child_tree(root, true) {
        kill(pid).unwrap();
    }
    for pid in [leaf, middle, root] {
        assert!(wait_for_stop(pid, Some(TIMEOUT)), "{pid} still running");
    }
}

#[test]
fn cli_tree_prints_the_same_order() {
    let scratch = Scratch::new("clitree");
    let (_victim, [root, middle, leaf]) = spawn_chain(&scratch);

    let run = std::process::Command::new(cli_exe())
        .args(["tree", &root.to_string()])
        .output()
        .unwrap();
    let printed: Vec<u32> = String::from_utf8_lossy(&run.stdout)
        .lines()
        .map(|l| l.parse().unwrap())
        .collect();
    assert_eq!(printed, vec![leaf, middle, root]);
}

#[test]
fn cli_reports_bad_usage_with_exit_code_two() {
    let run = std::process::Command::new(cli_exe())
        .args(["stop", "not-a-pid"])
        .output()
        .unwrap();
    assert_eq!(run.status.code(), Some(2));
}
