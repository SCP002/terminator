//! Shared harness for the end-to-end tests.
//!
//! Scenarios control *which console state the caller is in* by launching the
//! real `terminator` CLI with specific creation flags, and use the `victim`
//! example as the process being stopped. Everything the victim observes is
//! appended to a log file that the test polls.

#![allow(dead_code)]

use std::fs;
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};
use std::time::{Duration, Instant};

pub const TIMEOUT: Duration = Duration::from_secs(15);

#[cfg(windows)]
pub mod flags {
    /// A console exists but has no window: safe to blast with Ctrl events.
    pub const CREATE_NO_WINDOW: u32 = 0x0800_0000;
    /// The process has no console at all.
    pub const DETACHED_PROCESS: u32 = 0x0000_0008;
}

/// A unique temporary directory, removed on drop.
pub struct Scratch(PathBuf);

impl Scratch {
    pub fn new(name: &str) -> Self {
        let dir =
            std::env::temp_dir().join(format!("terminator-test-{}-{name}", std::process::id()));
        fs::create_dir_all(&dir).unwrap();
        Scratch(dir)
    }

    pub fn file(&self, name: &str) -> PathBuf {
        self.0.join(name)
    }
}

impl Drop for Scratch {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

pub fn cli_exe() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_terminator"))
}

pub fn victim_exe() -> PathBuf {
    example_exe("victim")
}

/// `target/<profile>/examples/<name>[.exe]`, built by `cargo test`.
///
/// Walks up from the test executable instead of assuming a layout: the test
/// binary lives in `target/<profile>/deps` or, with Cargo's newer layout, in
/// `target/<profile>/build/<package>/<hash>/out`.
pub fn example_exe(name: &str) -> PathBuf {
    let file = format!("{name}{}", std::env::consts::EXE_SUFFIX);
    let test_exe = std::env::current_exe().unwrap();
    test_exe
        .ancestors()
        .skip(1)
        .map(|dir| dir.join("examples").join(&file))
        .find(|candidate| candidate.is_file())
        .unwrap_or_else(|| {
            panic!(
                "examples/{file} not found above {}; run `cargo test` (it builds examples)",
                test_exe.display()
            )
        })
}

/// A running `victim` process and the log it writes.
pub struct Victim {
    pub child: Child,
    pub log: PathBuf,
}

impl Victim {
    pub fn pid(&self) -> u32 {
        self.child.id()
    }

    pub fn lines(&self) -> Vec<String> {
        fs::read_to_string(&self.log)
            .map(|s| s.lines().map(str::to_owned).collect())
            .unwrap_or_default()
    }

    /// Wait until the log contains exactly `line`.
    pub fn wait_line(&self, line: &str) -> bool {
        wait_for_line(&self.log, line, TIMEOUT)
    }

    /// Wait for the process to exit on its own. Returns whether it did.
    pub fn wait_exit(&mut self, timeout: Duration) -> bool {
        let start = Instant::now();
        while start.elapsed() < timeout {
            if matches!(self.child.try_wait(), Ok(Some(_))) {
                return true;
            }
            std::thread::sleep(Duration::from_millis(20));
        }
        false
    }
}

impl Drop for Victim {
    fn drop(&mut self) {
        // Clean up even when an assertion failed half-way (dogfooding kill).
        for pid in terminator::child_tree(self.child.id(), false) {
            let _ = terminator::kill(pid);
        }
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

/// Start `victim --out <log> <args...>`.
///
/// `creation_flags` are Windows `CreateProcess` flags and are ignored
/// elsewhere.
pub fn spawn_victim(scratch: &Scratch, name: &str, args: &[&str], creation_flags: u32) -> Victim {
    let log = scratch.file(&format!("{name}.log"));
    let mut cmd = Command::new(victim_exe());
    cmd.arg("--out")
        .arg(&log)
        .args(args)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null());
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(creation_flags);
    }
    #[cfg(not(windows))]
    let _ = creation_flags;
    let child = cmd
        .spawn()
        .unwrap_or_else(|e| panic!("cannot start {}: {e}", victim_exe().display()));
    Victim { child, log }
}

pub fn wait_for_line(log: &Path, line: &str, timeout: Duration) -> bool {
    let start = Instant::now();
    while start.elapsed() < timeout {
        if fs::read_to_string(log).is_ok_and(|s| s.lines().any(|l| l == line)) {
            return true;
        }
        std::thread::sleep(Duration::from_millis(20));
    }
    false
}

/// Result of one invocation of a program under test.
pub struct CliRun {
    pub code: i32,
    pub stdout: String,
    pub stderr: String,
}

/// Run the `terminator` CLI with route tracing enabled.
pub fn run_cli(creation_flags: u32, args: &[&str]) -> CliRun {
    run_exe(&cli_exe(), creation_flags, args)
}

/// Run any program with route tracing enabled and capture its output.
pub fn run_exe(exe: &Path, creation_flags: u32, args: &[&str]) -> CliRun {
    let mut cmd = Command::new(exe);
    cmd.env("TERMINATOR_DEBUG", "1").args(args);
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(creation_flags);
    }
    #[cfg(not(windows))]
    let _ = creation_flags;
    let out = cmd
        .output()
        .unwrap_or_else(|e| panic!("cannot run {}: {e}", exe.display()));
    CliRun {
        code: out.status.code().unwrap_or(-1),
        stdout: String::from_utf8_lossy(&out.stdout).into_owned(),
        stderr: String::from_utf8_lossy(&out.stderr).into_owned(),
    }
}
