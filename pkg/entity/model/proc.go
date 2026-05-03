// Package model contains the domain models shared across cmn-weave components.
// proc.go defines ProcInfo, the live process information read by the agent.
package model

// ProcInfo holds live process information read from /proc/<pid>/* for a
// running task. Fields map directly to the corresponding /proc entries.
type ProcInfo struct {
	// /proc/<pid>/status fields
	Name                     string
	State                    byte
	PID                      int32
	PPID                     int32
	Threads                  int32
	VmPeakKB                 int64
	VmRSSKB                  int64
	VoluntaryCtxtSwitches    int64
	NonvoluntaryCtxtSwitches int64

	// /proc/<pid>/stat fields
	UTime      int64
	STime      int64
	StartTime  int64
	VSize      int64
	RSSPages   int64
	ExitSignal int32

	// /proc/<pid>/cmdline, /proc/<pid>/environ, /proc/<pid>/exe
	Argv    []string
	Env     []string
	ExePath string
}
