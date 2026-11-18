//! Error type for the crate.

use std::fmt;

/// Convenience alias used throughout the crate.
pub type Result<T> = std::result::Result<T, Error>;

/// Something that went wrong while trying to stop or message a process.
#[derive(Debug)]
#[non_exhaustive]
pub enum Error {
    /// The target process does not exist (any more).
    ProcessNotFound(u32),
    /// The target process has no console attached, so console control events
    /// and console input cannot be delivered to it.
    NoConsole(u32),
    /// The target process has no visible top-level window to close.
    NoWindow(u32),
    /// A detached helper process was required but could not be used.
    ///
    /// The string explains what to configure (for example, a library loaded
    /// as a DLL into a foreign host that owns its own console).
    HelperUnavailable(String),
    /// The helper process ran but reported an unexpected exit code.
    HelperFailed {
        /// Raw process exit code of the helper.
        code: i32,
        /// Short description of what the helper was attempting.
        what: &'static str,
    },
    /// The operation is not supported on this platform or in this context.
    Unsupported(&'static str),
    /// An underlying OS call failed.
    Os(std::io::Error),
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Error::ProcessNotFound(pid) => write!(f, "process {pid} does not exist"),
            Error::NoConsole(pid) => write!(f, "process {pid} has no console attached"),
            Error::NoWindow(pid) => write!(f, "process {pid} has no visible main window"),
            Error::HelperUnavailable(why) => write!(f, "helper process unavailable: {why}"),
            Error::HelperFailed { code, what } => {
                write!(f, "helper process failed while {what} (exit code {code})")
            }
            Error::Unsupported(what) => write!(f, "unsupported: {what}"),
            Error::Os(e) => write!(f, "OS error: {e}"),
        }
    }
}

impl std::error::Error for Error {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Error::Os(e) => Some(e),
            _ => None,
        }
    }
}

impl From<std::io::Error> for Error {
    fn from(e: std::io::Error) -> Self {
        Error::Os(e)
    }
}
