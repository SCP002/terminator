//! `terminator` command line tool: stop processes gracefully, even on Windows.
//!
//! Linking the library registers the pre-`main` helper hook, so when this
//! executable is re-launched as a helper it never reaches the code below.

use std::process::ExitCode;
use std::time::Duration;

use terminator::{child_tree, is_running, kill, send_message, stop, wait_for_stop, Method};

const USAGE: &str = "\
terminator - stop processes gracefully, even on Windows

USAGE:
    terminator stop <PID>... [-m <METHOD>] [-t] [-w <MS>] [-f]
    terminator kill <PID>... [-t]
    terminator send <PID> <TEXT> [-n] [-e]
    terminator tree <PID>
    terminator wait <PID> [-w <MS>]

COMMANDS:
    stop    Ask processes to stop (Ctrl+C, Ctrl+Break or close window)
    kill    Terminate processes forcefully
    send    Write text to the standard input of a console process
    tree    Print all descendants of a process, deepest first
    wait    Wait until a process exits

OPTIONS:
    -m, --method <METHOD>   ctrl-c (default), ctrl-break or close-window
    -t, --tree              Apply to all descendants too, deepest first
    -w, --wait <MS>         Wait up to MS milliseconds for the process to exit
    -f, --force             If still running after the wait, kill it
                            (waits 5000 ms unless --wait is given)
    -n, --newline           Append a line ending to TEXT (CRLF on Windows)
    -e, --escapes           Interpret \\r \\n \\t \\\\ in TEXT
    -h, --help              Print this help
    -V, --version           Print version

EXIT CODES:
    0 success, 1 operation failed, 2 bad usage, 3 process still running
";

const EXIT_FAILED: u8 = 1;
const EXIT_USAGE: u8 = 2;
const EXIT_RUNNING: u8 = 3;

#[derive(Default)]
struct Opts {
    pids: Vec<u32>,
    text: Option<String>,
    method: Option<Method>,
    tree: bool,
    wait_ms: Option<u64>,
    force: bool,
    newline: bool,
    escapes: bool,
}

fn main() -> ExitCode {
    match run(std::env::args().skip(1).collect()) {
        Ok(code) => ExitCode::from(code),
        Err(msg) => {
            eprintln!("terminator: {msg}\nTry 'terminator --help'.");
            ExitCode::from(EXIT_USAGE)
        }
    }
}

fn run(args: Vec<String>) -> Result<u8, String> {
    let Some(command) = args.first().map(String::as_str) else {
        print!("{USAGE}");
        return Ok(EXIT_USAGE);
    };
    match command {
        "-h" | "--help" | "help" => {
            print!("{USAGE}");
            return Ok(0);
        }
        "-V" | "--version" => {
            println!("v{}", env!("CARGO_PKG_VERSION"));
            return Ok(0);
        }
        _ => {}
    }
    let opts = parse(command, &args[1..])?;
    match command {
        "stop" => Ok(cmd_stop(&opts)),
        "kill" => Ok(cmd_kill(&opts)),
        "send" => Ok(cmd_send(&opts)),
        "tree" => Ok(cmd_tree(&opts)),
        "wait" => Ok(cmd_wait(&opts)),
        other => Err(format!("unknown command '{other}'")),
    }
}

fn parse(command: &str, args: &[String]) -> Result<Opts, String> {
    let mut opts = Opts::default();
    let mut it = args.iter();
    while let Some(arg) = it.next() {
        let mut value = |name: &str| {
            it.next()
                .cloned()
                .ok_or_else(|| format!("option {name} needs a value"))
        };
        match arg.as_str() {
            "-m" | "--method" => {
                opts.method = Some(match value("--method")?.as_str() {
                    "ctrl-c" => Method::CtrlC,
                    "ctrl-break" => Method::CtrlBreak,
                    "close-window" => Method::CloseWindow,
                    other => return Err(format!("unknown method '{other}'")),
                })
            }
            "-t" | "--tree" => opts.tree = true,
            "-w" | "--wait" => {
                let v = value("--wait")?;
                opts.wait_ms = Some(v.parse().map_err(|_| format!("bad number '{v}'"))?);
            }
            "-f" | "--force" => opts.force = true,
            "-n" | "--newline" => opts.newline = true,
            "-e" | "--escapes" => opts.escapes = true,
            s if s.starts_with('-') && s.len() > 1 => return Err(format!("unknown option '{s}'")),
            positional => {
                if command == "send" && !opts.pids.is_empty() {
                    opts.text = Some(positional.to_owned());
                } else {
                    let pid = positional
                        .parse()
                        .map_err(|_| format!("'{positional}' is not a process id"))?;
                    opts.pids.push(pid);
                }
            }
        }
    }
    if opts.pids.is_empty() {
        return Err(format!("'{command}' needs a process id"));
    }
    if command == "send" && (opts.pids.len() != 1 || opts.text.is_none()) {
        return Err("usage: terminator send <PID> <TEXT>".into());
    }
    if matches!(command, "tree" | "wait") && opts.pids.len() != 1 {
        return Err(format!("'{command}' takes exactly one process id"));
    }
    Ok(opts)
}

/// `pid` plus, with `--tree`, all its descendants (deepest first).
fn targets(pid: u32, tree: bool) -> Vec<u32> {
    if tree {
        child_tree(pid, true)
    } else {
        vec![pid]
    }
}

fn cmd_stop(opts: &Opts) -> u8 {
    let method = opts.method.unwrap_or(Method::CtrlC);
    let mut code = 0;
    for &root in &opts.pids {
        for pid in targets(root, opts.tree) {
            if let Err(e) = stop(pid, method) {
                eprintln!("terminator: stop {pid}: {e}");
                code = EXIT_FAILED;
            }
        }
        let wait_ms = opts.wait_ms.or(opts.force.then_some(5000));
        if let (Some(ms), 0) = (wait_ms, code) {
            if !wait_for_stop(root, Some(Duration::from_millis(ms))) {
                if opts.force {
                    eprintln!("terminator: {root} still running after {ms} ms, killing");
                    for pid in targets(root, opts.tree) {
                        let _ = kill(pid);
                    }
                } else {
                    eprintln!("terminator: {root} still running after {ms} ms");
                    code = EXIT_RUNNING;
                }
            }
        }
    }
    code
}

fn cmd_kill(opts: &Opts) -> u8 {
    let mut code = 0;
    for &root in &opts.pids {
        for pid in targets(root, opts.tree) {
            if let Err(e) = kill(pid) {
                eprintln!("terminator: kill {pid}: {e}");
                code = EXIT_FAILED;
            }
        }
    }
    code
}

fn cmd_send(opts: &Opts) -> u8 {
    let mut text = opts.text.clone().unwrap_or_default();
    if opts.escapes {
        text = unescape(&text);
    }
    if opts.newline {
        text.push_str(if cfg!(windows) { "\r\n" } else { "\n" });
    }
    match send_message(opts.pids[0], &text) {
        Ok(()) => 0,
        Err(e) => {
            eprintln!("terminator: send {}: {e}", opts.pids[0]);
            EXIT_FAILED
        }
    }
}

fn cmd_tree(opts: &Opts) -> u8 {
    let pid = opts.pids[0];
    if !is_running(pid) {
        eprintln!("terminator: process {pid} does not exist");
        return EXIT_FAILED;
    }
    for child in child_tree(pid, true) {
        println!("{child}");
    }
    0
}

fn cmd_wait(opts: &Opts) -> u8 {
    let timeout = opts.wait_ms.map(Duration::from_millis);
    if wait_for_stop(opts.pids[0], timeout) {
        0
    } else {
        EXIT_RUNNING
    }
}

fn unescape(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    let mut chars = s.chars();
    while let Some(c) = chars.next() {
        if c != '\\' {
            out.push(c);
            continue;
        }
        match chars.next() {
            Some('r') => out.push('\r'),
            Some('n') => out.push('\n'),
            Some('t') => out.push('\t'),
            Some('\\') => out.push('\\'),
            Some(other) => {
                out.push('\\');
                out.push(other);
            }
            None => out.push('\\'),
        }
    }
    out
}
