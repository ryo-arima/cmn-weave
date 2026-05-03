//! cmn-weave agent execution core.
//!
//! All functions exposed across the FFI boundary are declared `extern "C"`.
//! They must remain panic-free: any internal failure is converted into an
//! error code before the function returns so that panics never unwind into Go.
//!
//! Memory ownership rules
//! ----------------------
//! * The caller (Go/cgo) passes a pointer to an already-allocated output
//!   struct that Rust writes into.  Rust never allocates heap memory on
//!   behalf of the caller for structs; for variable-length data (argv, env,
//!   cmdline, exe path) a fixed-capacity buffer embedded in the struct is
//!   used instead.
//! * Pointers passed from Go must be valid and non-null for the duration of
//!   the call; Rust does not retain them after the function returns.

use std::fs;
use std::os::unix::ffi::OsStrExt;

// ---------------------------------------------------------------------------
// Version
// ---------------------------------------------------------------------------

/// Returns the semantic version of the core library encoded as a packed
/// `u32`: `(major << 16) | (minor << 8) | patch`.
#[unsafe(no_mangle)]
pub extern "C" fn cmnweave_core_version() -> u32 {
    (0u32 << 16) | (1u32 << 8) | 1u32
}

// ---------------------------------------------------------------------------
// Shared constants
// ---------------------------------------------------------------------------

/// Maximum length (bytes) for path / string fields in proc structs.
pub const CMNWEAVE_BUF_LEN: usize = 4096;
/// Maximum number of argv / env entries captured.
pub const CMNWEAVE_ARGV_MAX: usize = 256;

// ---------------------------------------------------------------------------
// ProcStatus — mirrors /proc/<pid>/status
// ---------------------------------------------------------------------------

/// Subset of fields from `/proc/<pid>/status` that the agent reports back to
/// the server.
///
/// Unread / unparseable fields are left at their zero values.
#[repr(C)]
pub struct CmnweaveProcStatus {
    /// Process name (Name: field), NUL-terminated.
    pub name: [u8; 64],
    /// Single-character state (State: field): 'R','S','D','Z','T','t','I'.
    pub state: u8,
    /// Process ID.
    pub pid: i32,
    /// Parent process ID.
    pub ppid: i32,
    /// Number of threads (Threads: field).
    pub threads: i32,
    /// Virtual memory peak in kB (VmPeak:).
    pub vm_peak_kb: i64,
    /// Resident set size in kB (VmRSS:).
    pub vm_rss_kb: i64,
    /// Voluntary context switches (voluntary_ctxt_switches:).
    pub voluntary_ctxt_switches: i64,
    /// Non-voluntary context switches (nonvoluntary_ctxt_switches:).
    pub nonvoluntary_ctxt_switches: i64,
}

/// Reads `/proc/<pid>/status` and writes the parsed fields into `*out`.
///
/// Returns 0 on success, -1 if the file cannot be read, -2 on parse errors.
/// On any non-zero return the contents of `*out` are undefined.
///
/// # Safety
/// `out` must be a valid, aligned, non-null pointer to a `CmnweaveProcStatus`.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn cmnweave_read_proc_status(
    pid: i32,
    out: *mut CmnweaveProcStatus,
) -> i32 {
    let path = format!("/proc/{}/status", pid);
    let content = match fs::read_to_string(&path) {
        Ok(c) => c,
        Err(_) => return -1,
    };
    let s = unsafe { &mut *out };
    *s = unsafe { std::mem::zeroed() };
    for line in content.lines() {
        let Some((key, val)) = line.split_once(':') else { continue };
        let key = key.trim();
        let val = val.trim();
        match key {
            "Name" => copy_str_to_buf(val, &mut s.name),
            "State" => s.state = val.as_bytes().first().copied().unwrap_or(0),
            "Pid" => s.pid = val.parse().unwrap_or(0),
            "PPid" => s.ppid = val.parse().unwrap_or(0),
            "Threads" => s.threads = val.parse().unwrap_or(0),
            "VmPeak" => s.vm_peak_kb = parse_kb(val),
            "VmRSS" => s.vm_rss_kb = parse_kb(val),
            "voluntary_ctxt_switches" => {
                s.voluntary_ctxt_switches = val.parse().unwrap_or(0)
            }
            "nonvoluntary_ctxt_switches" => {
                s.nonvoluntary_ctxt_switches = val.parse().unwrap_or(0)
            }
            _ => {}
        }
    }
    0
}

// ---------------------------------------------------------------------------
// ProcStat — mirrors /proc/<pid>/stat
// ---------------------------------------------------------------------------

/// Key fields from `/proc/<pid>/stat` (space-separated).
///
/// Only the fields needed for task reporting are captured.
#[repr(C)]
pub struct CmnweaveProcStat {
    pub pid: i32,
    /// Process state character (field 3).
    pub state: u8,
    pub ppid: i32,
    /// User-mode CPU time in clock ticks (field 14, utime).
    pub utime: i64,
    /// Kernel-mode CPU time in clock ticks (field 15, stime).
    pub stime: i64,
    /// Start time in clock ticks since boot (field 22, starttime).
    pub starttime: i64,
    /// Virtual memory size in bytes (field 23, vsize).
    pub vsize: i64,
    /// RSS in pages (field 24, rss).
    pub rss_pages: i64,
    /// Exit signal number (field 38).
    pub exit_signal: i32,
}

/// Reads `/proc/<pid>/stat` and writes parsed fields into `*out`.
///
/// Returns 0 on success, -1 on read error, -2 on parse error.
///
/// # Safety
/// `out` must be a valid, aligned, non-null pointer to a `CmnweaveProcStat`.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn cmnweave_read_proc_stat(
    pid: i32,
    out: *mut CmnweaveProcStat,
) -> i32 {
    let path = format!("/proc/{}/stat", pid);
    let content = match fs::read_to_string(&path) {
        Ok(c) => c,
        Err(_) => return -1,
    };
    // The comm field (2nd field) may contain spaces and is wrapped in '(' ')'.
    // Skip past the closing ')' before splitting the remaining fields.
    let after_comm = match content.rfind(')') {
        Some(pos) => &content[pos + 1..],
        None => return -2,
    };
    let fields: Vec<&str> = after_comm.split_whitespace().collect();
    // After comm the field indices shift by 2 (1-based spec: field 3 = index 0 here).
    let s = unsafe { &mut *out };
    *s = unsafe { std::mem::zeroed() };
    s.pid = pid;
    s.state = fields.first().and_then(|f| f.bytes().next()).unwrap_or(0);
    s.ppid       = parse_field(&fields, 1);
    s.utime      = parse_field(&fields, 11);
    s.stime      = parse_field(&fields, 12);
    s.starttime  = parse_field(&fields, 19);
    s.vsize      = parse_field(&fields, 20);
    s.rss_pages  = parse_field(&fields, 21);
    s.exit_signal = parse_field(&fields, 35);
    0
}

// ---------------------------------------------------------------------------
// ProcCmdline — mirrors /proc/<pid>/cmdline
// ---------------------------------------------------------------------------

/// NUL-separated command line from `/proc/<pid>/cmdline`.
///
/// `argc` holds the number of entries written into `argv`; each entry is a
/// NUL-terminated C string stored in the fixed inline buffer `buf`.  The
/// `argv` array contains byte-offsets from `buf[0]` to each argument start.
#[repr(C)]
pub struct CmnweaveProcCmdline {
    /// Flat byte buffer holding all argv strings, NUL-terminated and packed.
    pub buf: [u8; CMNWEAVE_BUF_LEN],
    /// Byte offsets into `buf` for each argument; `argc` entries are valid.
    pub offsets: [u32; CMNWEAVE_ARGV_MAX],
    /// Number of valid entries in `offsets`.
    pub argc: u32,
}

/// Reads `/proc/<pid>/cmdline` and writes parsed data into `*out`.
///
/// Returns 0 on success, -1 on read error.
///
/// # Safety
/// `out` must be a valid, aligned, non-null pointer to a `CmnweaveProcCmdline`.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn cmnweave_read_proc_cmdline(
    pid: i32,
    out: *mut CmnweaveProcCmdline,
) -> i32 {
    let path = format!("/proc/{}/cmdline", pid);
    let raw = match fs::read(&path) {
        Ok(b) => b,
        Err(_) => return -1,
    };
    let s = unsafe { &mut *out };
    *s = unsafe { std::mem::zeroed() };
    let mut buf_pos: usize = 0;
    let mut argc: usize = 0;
    for arg in raw.split(|&b| b == 0) {
        if arg.is_empty() {
            continue;
        }
        if argc >= CMNWEAVE_ARGV_MAX || buf_pos + arg.len() + 1 > CMNWEAVE_BUF_LEN {
            break;
        }
        s.offsets[argc] = buf_pos as u32;
        s.buf[buf_pos..buf_pos + arg.len()].copy_from_slice(arg);
        buf_pos += arg.len() + 1; // +1 for NUL terminator (already zeroed)
        argc += 1;
    }
    s.argc = argc as u32;
    0
}

// ---------------------------------------------------------------------------
// ProcEnviron — mirrors /proc/<pid>/environ
// ---------------------------------------------------------------------------

/// NUL-separated environment from `/proc/<pid>/environ`.
/// Layout is identical to `CmnweaveProcCmdline`.
#[repr(C)]
pub struct CmnweaveProcEnviron {
    pub buf: [u8; CMNWEAVE_BUF_LEN],
    pub offsets: [u32; CMNWEAVE_ARGV_MAX],
    pub envc: u32,
}

/// Reads `/proc/<pid>/environ` and writes parsed data into `*out`.
///
/// Returns 0 on success, -1 on read error.
///
/// # Safety
/// `out` must be a valid, aligned, non-null pointer to a `CmnweaveProcEnviron`.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn cmnweave_read_proc_environ(
    pid: i32,
    out: *mut CmnweaveProcEnviron,
) -> i32 {
    let path = format!("/proc/{}/environ", pid);
    let raw = match fs::read(&path) {
        Ok(b) => b,
        Err(_) => return -1,
    };
    let s = unsafe { &mut *out };
    *s = unsafe { std::mem::zeroed() };
    let mut buf_pos: usize = 0;
    let mut envc: usize = 0;
    for entry in raw.split(|&b| b == 0) {
        if entry.is_empty() {
            continue;
        }
        if envc >= CMNWEAVE_ARGV_MAX || buf_pos + entry.len() + 1 > CMNWEAVE_BUF_LEN {
            break;
        }
        s.offsets[envc] = buf_pos as u32;
        s.buf[buf_pos..buf_pos + entry.len()].copy_from_slice(entry);
        buf_pos += entry.len() + 1;
        envc += 1;
    }
    s.envc = envc as u32;
    0
}

// ---------------------------------------------------------------------------
// ProcExe — mirrors /proc/<pid>/exe (readlink)
// ---------------------------------------------------------------------------

/// Resolved path of the executable from `/proc/<pid>/exe`.
#[repr(C)]
pub struct CmnweaveProcExe {
    /// NUL-terminated absolute path. Truncated to `CMNWEAVE_BUF_LEN - 1` bytes
    /// if the path exceeds the buffer.
    pub path: [u8; CMNWEAVE_BUF_LEN],
    /// Actual byte length of `path` (excluding NUL terminator).
    pub len: u32,
}

/// Resolves `/proc/<pid>/exe` and writes the path into `*out`.
///
/// Returns 0 on success, -1 on error.
///
/// # Safety
/// `out` must be a valid, aligned, non-null pointer to a `CmnweaveProcExe`.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn cmnweave_read_proc_exe(
    pid: i32,
    out: *mut CmnweaveProcExe,
) -> i32 {
    let link = format!("/proc/{}/exe", pid);
    let target = match fs::read_link(&link) {
        Ok(p) => p,
        Err(_) => return -1,
    };
    let bytes = target.as_os_str().as_bytes();
    let s = unsafe { &mut *out };
    *s = unsafe { std::mem::zeroed() };
    let copy_len = bytes.len().min(CMNWEAVE_BUF_LEN - 1);
    s.path[..copy_len].copy_from_slice(&bytes[..copy_len]);
    s.len = copy_len as u32;
    0
}

// ---------------------------------------------------------------------------
// ProcInfo — aggregate of all five /proc/<pid>/* reads
// ---------------------------------------------------------------------------

/// Aggregate of all live process information for a single PID.
///
/// Each sub-struct can also be read individually via the dedicated functions
/// above; this composite type allows Go to make a single FFI call.
#[repr(C)]
pub struct CmnweaveProcInfo {
    pub status:  CmnweaveProcStatus,
    pub stat:    CmnweaveProcStat,
    pub cmdline: CmnweaveProcCmdline,
    pub environ: CmnweaveProcEnviron,
    pub exe:     CmnweaveProcExe,
}

/// Reads all `/proc/<pid>/*` sources into `*out` with a single call.
///
/// `status` and `stat` are mandatory: if either fails the function returns
/// their error code immediately and `*out` is in an undefined state.
/// `cmdline`, `environ`, and `exe` are best-effort: failures are silently
/// ignored and the corresponding sub-structs are left zeroed.
///
/// Returns 0 on success, or the non-zero rc from the first mandatory failure.
///
/// # Safety
/// `out` must be a valid, aligned, non-null pointer to a `CmnweaveProcInfo`.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn cmnweave_read_proc_info(
    pid: i32,
    out: *mut CmnweaveProcInfo,
) -> i32 {
    let s = unsafe { &mut *out };
    *s = unsafe { std::mem::zeroed() };

    // Mandatory reads — return immediately on error.
    let rc = unsafe { cmnweave_read_proc_status(pid, &mut s.status) };
    if rc != 0 {
        return rc;
    }
    let rc = unsafe { cmnweave_read_proc_stat(pid, &mut s.stat) };
    if rc != 0 {
        return rc;
    }

    // Best-effort reads — ignore failures.
    unsafe { cmnweave_read_proc_cmdline(pid, &mut s.cmdline) };
    unsafe { cmnweave_read_proc_environ(pid, &mut s.environ) };
    unsafe { cmnweave_read_proc_exe(pid, &mut s.exe) };

    0
}

// ---------------------------------------------------------------------------
// Helpers (not exported)
// ---------------------------------------------------------------------------

fn copy_str_to_buf(src: &str, dst: &mut [u8]) {
    let bytes = src.as_bytes();
    let len = bytes.len().min(dst.len() - 1);
    dst[..len].copy_from_slice(&bytes[..len]);
    // remaining bytes are already zero (struct is zeroed on entry)
}

/// Parse a `"1234 kB"` value string, returning the numeric part.
fn parse_kb(s: &str) -> i64 {
    s.split_whitespace()
        .next()
        .and_then(|n| n.parse().ok())
        .unwrap_or(0)
}

/// Parse the field at `index` in a split stat line, returning 0 on failure.
fn parse_field<T: std::str::FromStr + Default>(fields: &[&str], index: usize) -> T {
    fields.get(index).and_then(|f| f.parse().ok()).unwrap_or_default()
}

