// Package model contains domain types shared across all cmn-weave components.
package model

import "time"

// TaskState reflects the Linux process state exposed via /proc/<pid>/status
// ("State:" field) plus lifecycle states managed by the server.
//
// Mapping to /proc/<pid>/status State characters:
//   R → TaskStateRunning
//   S → TaskStateSleeping
//   D → TaskStateWaiting   (uninterruptible sleep)
//   Z → TaskStateZombie
//   T → TaskStateStopped   (stopped by signal)
//   t → TaskStateTraced    (stopped by debugger)
//   I → TaskStateIdle
type TaskState string

const (
	// Server-managed states (no corresponding proc state).
	TaskStateUnspecified TaskState = "UNSPECIFIED"
	TaskStatePending     TaskState = "PENDING"    // queued, not yet started
	TaskStateCancelled   TaskState = "CANCELLED"  // cancelled before or after start
	TaskStateFailed      TaskState = "FAILED"     // exited with non-zero code or error
	TaskStateSucceeded   TaskState = "SUCCEEDED"  // exited with code 0

	// Linux proc states (mirrors /proc/<pid>/status "State:" values).
	TaskStateRunning  TaskState = "R" // Running
	TaskStateSleeping TaskState = "S" // Sleeping in an interruptible wait
	TaskStateWaiting  TaskState = "D" // Waiting in uninterruptible disk sleep
	TaskStateZombie   TaskState = "Z" // Zombie
	TaskStateStopped  TaskState = "T" // Stopped (on a signal)
	TaskStateTraced   TaskState = "t" // Tracing stop
	TaskStateIdle     TaskState = "I" // Idle kernel thread
)

// Task represents a unit of work dispatched by a client and executed by a
// remote agent as a Linux process. The proc-level fields are populated by the
// agent by reading /proc/<PID>/* after the process is spawned.
type Task struct {
	// ID is the server-assigned UUIDv4 that uniquely identifies the task.
	ID string

	// NodeID is the UUID of the Node (agent) executing this task.
	NodeID string

	// State is the current lifecycle / proc state of the task.
	State TaskState

	// ---- Dispatch input ----

	// Argv holds the command and its arguments as dispatched (mirrors
	// /proc/<PID>/cmdline split by NUL bytes).
	Argv []string

	// Env holds the KEY=VALUE environment variables supplied at dispatch time
	// (mirrors /proc/<PID>/environ split by NUL bytes).
	Env []string

	// Cwd is the working directory requested at dispatch time (mirrors
	// /proc/<PID>/cwd resolved symlink).
	Cwd string

	// ---- Runtime proc fields (populated by the agent) ----

	// PID is the Linux process ID assigned by the kernel.
	PID int32

	// PPID is the parent process ID (/proc/<PID>/status "PPid:").
	PPID int32

	// Threads is the number of threads in the process group
	// (/proc/<PID>/status "Threads:").
	Threads int32

	// VmRSS is the resident set size in kilobytes
	// (/proc/<PID>/status "VmRSS:").
	VmRSS int64

	// ExitCode is the process exit code (/proc/<PID>/stat field 52 or
	// waitpid). Zero means success; only meaningful when State is
	// Succeeded or Failed.
	ExitCode int32

	// Signal is the signal number that terminated the process, if any.
	// Zero if the process exited normally.
	Signal int32

	// ErrorMessage contains a human-readable description for failures that
	// have no meaningful exit code (e.g. exec failure, I/O error).
	ErrorMessage string

	// ---- Lifecycle timestamps ----

	// CreatedAt is the time the task row was first inserted.
	CreatedAt time.Time

	// UpdatedAt is the time the task row was last modified.
	UpdatedAt time.Time

	// StartedAt is the time the agent spawned the process.
	// Zero if the task has not started yet.
	StartedAt *time.Time

	// FinishedAt is the time the process exited.
	// Zero if the task is still running.
	FinishedAt *time.Time
}
