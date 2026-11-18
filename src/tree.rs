//! Process tree helpers.

use std::collections::HashMap;

use sysinfo::{ProcessRefreshKind, ProcessesToUpdate, System};

/// Returns the ids of all descendants of the process `pid`, deepest first.
///
/// The order is suitable for stopping a tree bottom-up: every process appears
/// after all of its own descendants. Siblings keep the order in which they
/// were started (oldest first). If `with_root` is `true`, `pid` itself is
/// appended last.
///
/// Returns an empty vector (or just `[pid]` with `with_root`) when the process
/// has no children or does not exist.
///
/// On Windows the console host (`conhost.exe`, `OpenConsole.exe`) that the
/// system starts for every console application shows up as a child of that
/// application. It is plumbing, not part of the program's own process tree,
/// and stopping it would only destroy the console, so it is left out.
pub fn child_tree(pid: u32, with_root: bool) -> Vec<u32> {
    let mut sys = System::new();
    sys.refresh_processes_specifics(ProcessesToUpdate::All, true, ProcessRefreshKind::new());

    // parent -> [(start_time, child)]
    let mut children: HashMap<u32, Vec<(u64, u32)>> = HashMap::new();
    for (child, process) in sys.processes() {
        if is_console_host(process) {
            continue;
        }
        if let Some(parent) = process.parent() {
            let (parent, child) = (parent.as_u32(), child.as_u32());
            if parent != child {
                children
                    .entry(parent)
                    .or_default()
                    .push((process.start_time(), child));
            }
        }
    }
    for list in children.values_mut() {
        list.sort_unstable();
    }

    let mut out = Vec::new();
    collect(pid, &children, &mut out, 0);
    if with_root {
        out.push(pid);
    }
    out
}

#[cfg(windows)]
fn is_console_host(process: &sysinfo::Process) -> bool {
    let name = process.name().to_string_lossy().to_ascii_lowercase();
    name == "conhost.exe" || name == "openconsole.exe"
}

#[cfg(not(windows))]
fn is_console_host(_process: &sysinfo::Process) -> bool {
    false
}

fn collect(pid: u32, children: &HashMap<u32, Vec<(u64, u32)>>, out: &mut Vec<u32>, depth: usize) {
    // Windows reuses PIDs and keeps stale parent ids, which can form cycles.
    if depth > 256 {
        return;
    }
    if let Some(list) = children.get(&pid) {
        for &(_, child) in list {
            if out.contains(&child) || child == pid {
                continue;
            }
            collect(child, children, out, depth + 1);
            out.push(child);
        }
    }
}
