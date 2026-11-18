//! Start a console program in its own console and stop it gracefully.
//!
//! ```text
//! cargo run --example stop_child
//! cargo run --example stop_child -- --hidden --helper path\to\terminator.exe
//! ```
//!
//! The child is `ping -t` (Windows) or `sleep` (Unix), so no helper scripts
//! are needed. On Windows the child gets its own console, which makes this
//! the case that used to need the kamikaze proxy.
//!
//! Options:
//! * `--hidden`       give the child a console without a window
//! * `--helper <EXE>` (Windows) use another executable that links terminator
//!   as the detached helper, as a library loaded from a DLL would have to

use std::process::{Command, Stdio};
use std::time::Duration;

use terminator::{kill, stop, wait_for_stop, Method};

fn main() {
    let mut hidden = false;
    let mut helper: Option<String> = None;
    let mut args = std::env::args().skip(1);
    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--hidden" => hidden = true,
            "--helper" => helper = args.next(),
            other => {
                eprintln!("unknown argument '{other}'");
                std::process::exit(2);
            }
        }
    }

    #[cfg(windows)]
    if let Some(path) = &helper {
        terminator::set_helper(terminator::Helper::Path(path.into()));
    }
    #[cfg(not(windows))]
    let _ = (hidden, &helper);

    let mut cmd;
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        const CREATE_NEW_CONSOLE: u32 = 0x0000_0010;
        const CREATE_NO_WINDOW: u32 = 0x0800_0000;
        cmd = Command::new("ping");
        cmd.args(["-t", "127.0.0.1"]).creation_flags(if hidden {
            CREATE_NO_WINDOW
        } else {
            CREATE_NEW_CONSOLE
        });
    }
    #[cfg(not(windows))]
    {
        cmd = Command::new("sleep");
        cmd.arg("1000");
    }
    let mut child = cmd
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
        .expect("cannot start the child");
    let pid = child.id();
    println!("started {pid}");
    std::thread::sleep(Duration::from_millis(500));

    if let Err(e) = stop(pid, Method::CtrlC) {
        eprintln!("cannot stop {pid}: {e}");
        let _ = kill(pid);
        let _ = child.wait();
        std::process::exit(1);
    }
    println!("Ctrl+C delivered to {pid}");

    if wait_for_stop(pid, Some(Duration::from_secs(5))) {
        println!("{pid} exited gracefully");
    } else {
        println!("{pid} ignored Ctrl+C, killing it");
        let _ = kill(pid);
    }
    let _ = child.wait();
}
