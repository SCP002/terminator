# terminator

> Stop processes gracefully, even on Windows. Library and command line tool.

## Capabilities

On Windows it can:

* Send Ctrl + C to console applications
* Send Ctrl + Break to console applications
* Close graphical applications, including packaged UWP apps, as if their window was closed
* Send messages to standard input of console applications to answer questions such as "Y/N?"

On Linux and macOS it can:

* Send `SIGINT` to applications
* Send messages to the terminal of an application (`TIOCSTI`, usually needs root)

On every platform it can kill a process, wait for it to exit and list its descendants.

## Command line

```text
terminator stop <PID>... [-m ctrl-c|ctrl-break|close-window] [-t] [-w <MS>] [-f]
terminator kill <PID>... [-t]
terminator send <PID> <TEXT> [-n] [-e]
terminator tree <PID>
terminator wait <PID> [-w <MS>]
```

```text
terminator stop 1234 --wait 3000 --force   # Ctrl+C, then kill if still alive after 3 s
terminator stop 1234 --tree                # every descendant first, then the process
terminator send 1234 "Y" --newline         # answer "Terminate batch job (Y/N)?"
```

Exit codes: `0` success, `1` operation failed, `2` bad usage, `3` process still running.

## Library

```rust
use std::time::Duration;
use terminator::{kill, stop, wait_for_stop, Method};

fn main() -> terminator::Result<()> {
    let pid = 1234;
    stop(pid, Method::CtrlC)?;
    if !wait_for_stop(pid, Some(Duration::from_secs(3))) {
        kill(pid)?;
    }
    Ok(())
}
```

See the [`examples`](examples) folder and the API documentation (`cargo doc --open`).

## How it works on Windows

A process can be attached to only one console, and reaching a foreign console means attaching to
it. Detaching from your own console first can destroy it. The crate therefore picks the cheapest
route that is safe:

| Caller                       | Target          | Route       | What happens                                  |
|------------------------------|-----------------|-------------|-----------------------------------------------|
| shares the target's console  | same console    | `Direct`    | Generate the event right here                 |
| has no console               | any console     | `InProcess` | Attach, send, detach (under a lock)           |
| has its own console          | another console | `Helper`    | A short-lived detached helper attaches, sends |

The helper is **the current executable, re-launched** with a marker argument. A C runtime
initializer that the crate registers (`.CRT$XCB`) recognizes the marker before `main` and exits
after doing the job, so:

* there is no embedded proxy binary and nothing is written to the temporary folder;
* you do not have to call anything from `main`: any binary that can spawn a helper contains the
  initializer by construction;
* it runs before `ctor`/`inventory` style initializers.

If the code is loaded as a DLL into a foreign host (Python extension, plugin), the host cannot be
re-launched. Use `terminator::set_helper(Helper::Path(..))` to name an executable that links
terminator, for example `terminator.exe`. Without it you get `Error::HelperUnavailable` that says
so. A helper that does not understand the protocol is killed after 10 seconds and reported.

Set `TERMINATOR_DEBUG=1` to print the chosen route.

Notes:

* `child_tree` leaves out `conhost.exe` / `OpenConsole.exe`: Windows lists the console host as a
  child of every console application, but it is not part of the program's own process tree.
* Ctrl+C and Ctrl+Break reach **every** process attached to the target console that has not
  disabled them, not only the target process.
* A UWP app has no top-level window of its own, so `close-window` finds the `ApplicationFrameWindow` that hosts
  it and closes that, like the close button does.
* A target that is elevated while the caller is not may fail with "access denied".

## Tests

```text
cargo test
```

The end-to-end tests run the real CLI in each console state (own console, no console, shared
console) against a small `victim` process (`examples/victim.rs`) that records what it observed,
and check both the outcome and the route taken. They use hidden consoles, but `window.rs` needs an
interactive desktop session. `cargo test -- --ignored` additionally opens the Calculator to check
that packaged (UWP) apps can be closed by pid.

## Status

Tested on Windows 11 (x86_64) and Linux (x86_64). macOS and other Unix systems compile the same
POSIX code but are untested; ARM64 and x86 Windows are untested. Closing a UWP app was checked on the
Calculator (an ignored test, see above).
