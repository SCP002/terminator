//! POSIX signal delivery.

#![cfg(unix)]

mod common;

use common::*;
use terminator::{stop, wait_for_stop, Method};

#[test]
fn every_method_sends_sigint() {
    for (i, method) in [Method::CtrlC, Method::CtrlBreak, Method::CloseWindow]
        .into_iter()
        .enumerate()
    {
        let scratch = Scratch::new(&format!("sigint{i}"));
        let mut victim = spawn_victim(&scratch, "victim", &["wait"], 0);
        assert!(victim.wait_line("READY"));

        stop(victim.pid(), method).unwrap();

        assert!(victim.wait_exit(TIMEOUT), "{method:?}: victim did not exit");
        assert!(
            victim.lines().iter().any(|l| l == "SIGINT"),
            "{method:?}: log {:?}",
            victim.lines()
        );
        assert!(wait_for_stop(victim.pid(), Some(TIMEOUT)));
    }
}
