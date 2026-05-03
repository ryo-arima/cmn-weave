// linux.go implements LinuxRepository, which reads live process information
// from /proc/<pid>/* via the Rust staticlib (pkg/agent/rust) using cgo and
// maps the C structs onto domain types.
package repository

/*
#cgo CFLAGS: -I${SRCDIR}/../../grpc/auto
#cgo LDFLAGS: -L${SRCDIR}/../../../../target/release -lcmnweave_core -ldl -lpthread

#include "cmnweave_core.h"
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
	"unsafe"

	"github.com/ryo-arima/cmn-weave/pkg/entity/model"
)

// LinuxRepository reads live process information from /proc/<pid>/* via the
// Rust staticlib.
type LinuxRepository struct{}

// NewLinuxRepository creates a LinuxRepository.
func NewLinuxRepository() *LinuxRepository {
	return &LinuxRepository{}
}

// ReadProcInfo reads /proc/<pid>/{status,stat,cmdline,environ,exe} via a
// single Rust FFI call and returns a populated model.ProcInfo.
func (rcvr *LinuxRepository) ReadProcInfo(pid int32) (*model.ProcInfo, error) {
	var cInfo C.CmnweaveProcInfo
	if rc := C.cmnweave_read_proc_info(C.int32_t(pid), &cInfo); rc != 0 {
		return nil, fmt.Errorf("read /proc/%d: rc=%d", pid, rc)
	}
	info := mapProcInfo(&cInfo.status, &cInfo.stat, &cInfo.cmdline, &cInfo.environ, &cInfo.exe)
	return info, nil
}

// mapProcInfo assembles a model.ProcInfo from the five C structs populated by
// the Rust staticlib. Each struct is mapped by a dedicated helper below.
func mapProcInfo(
	s *C.CmnweaveProcStatus,
	t *C.CmnweaveProcStat,
	cl *C.CmnweaveProcCmdline,
	ev *C.CmnweaveProcEnviron,
	ex *C.CmnweaveProcExe,
) *model.ProcInfo {
	info := mapStatus(s)
	mapStat(info, t)
	mapCmdline(info, cl)
	mapEnviron(info, ev)
	mapExe(info, ex)
	return info
}

// mapStatus maps CmnweaveProcStatus fields onto a new model.ProcInfo.
func mapStatus(c *C.CmnweaveProcStatus) *model.ProcInfo {
	return &model.ProcInfo{
		Name:                     C.GoString((*C.char)(unsafe.Pointer(&c.name[0]))),
		State:                    byte(c.state),
		PID:                      int32(c.pid),
		PPID:                     int32(c.ppid),
		Threads:                  int32(c.threads),
		VmPeakKB:                 int64(c.vm_peak_kb),
		VmRSSKB:                  int64(c.vm_rss_kb),
		VoluntaryCtxtSwitches:    int64(c.voluntary_ctxt_switches),
		NonvoluntaryCtxtSwitches: int64(c.nonvoluntary_ctxt_switches),
	}
}

// mapStat maps CmnweaveProcStat fields into an existing model.ProcInfo.
func mapStat(info *model.ProcInfo, c *C.CmnweaveProcStat) {
	info.UTime      = int64(c.utime)
	info.STime      = int64(c.stime)
	info.StartTime  = int64(c.starttime)
	info.VSize      = int64(c.vsize)
	info.RSSPages   = int64(c.rss_pages)
	info.ExitSignal = int32(c.exit_signal)
}

// mapCmdline maps CmnweaveProcCmdline fields into an existing model.ProcInfo.
func mapCmdline(info *model.ProcInfo, c *C.CmnweaveProcCmdline) {
	argc := int(c.argc)
	info.Argv = make([]string, 0, argc)
	for i := range argc {
		off := int(c.offsets[i])
		info.Argv = append(info.Argv, C.GoString((*C.char)(unsafe.Pointer(&c.buf[off]))))
	}
}

// mapEnviron maps CmnweaveProcEnviron fields into an existing model.ProcInfo.
func mapEnviron(info *model.ProcInfo, c *C.CmnweaveProcEnviron) {
	envc := int(c.envc)
	info.Env = make([]string, 0, envc)
	for i := range envc {
		off := int(c.offsets[i])
		info.Env = append(info.Env, C.GoString((*C.char)(unsafe.Pointer(&c.buf[off]))))
	}
}

// mapExe maps CmnweaveProcExe fields into an existing model.ProcInfo.
func mapExe(info *model.ProcInfo, c *C.CmnweaveProcExe) {
	info.ExePath = C.GoString((*C.char)(unsafe.Pointer(&c.path[0])))
}

// ApplyProcInfo copies live proc fields from info into task and stamps
// UpdatedAt. The caller is responsible for persisting the updated task.
func ApplyProcInfo(task *model.Task, info *model.ProcInfo) {
	if info == nil || task == nil {
		return
	}
	task.State = taskStateFromByte(info.State)
	task.PID = info.PID
	task.PPID = info.PPID
	task.Threads = info.Threads
	task.VmRSS = info.VmRSSKB
	task.Signal = info.ExitSignal
	if len(info.Argv) > 0 {
		task.Argv = info.Argv
	}
	if len(info.Env) > 0 {
		task.Env = info.Env
	}
	task.UpdatedAt = time.Now()
}

// taskStateFromByte maps a /proc/<pid>/status "State:" byte to model.TaskState.
func taskStateFromByte(b byte) model.TaskState {
	switch b {
	case 'R':
		return model.TaskStateRunning
	case 'S':
		return model.TaskStateSleeping
	case 'D':
		return model.TaskStateWaiting
	case 'Z':
		return model.TaskStateZombie
	case 'T':
		return model.TaskStateStopped
	case 't':
		return model.TaskStateTraced
	case 'I':
		return model.TaskStateIdle
	default:
		return model.TaskStateUnspecified
	}
}

// ExecutionRepository runs processes on behalf of the agent.
type ExecutionRepository interface {
	// Run executes argv[0] with argv[1:] as arguments and env as additional
	// KEY=VALUE environment variables. It blocks until the process exits or
	// ctx is cancelled. Returns the process exit code, any combined output, and
	// a Go-level error for infrastructure failures.
	Run(ctx context.Context, argv []string, env []string) (exitCode int32, output string, err error)
}

// OSExecRepository implements ExecutionRepository using os/exec.
type OSExecRepository struct{}

// NewOSExecRepository creates an OSExecRepository.
func NewOSExecRepository() *OSExecRepository {
	return &OSExecRepository{}
}

// Run executes the command. env entries are appended to the inherited process environment.
func (r *OSExecRepository) Run(ctx context.Context, argv []string, env []string) (int32, string, error) {
	if len(argv) == 0 {
		return 1, "empty argv", fmt.Errorf("argv must not be empty")
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return int32(exitErr.ExitCode()), string(out), nil
		}
		return -1, "", fmt.Errorf("exec: %w", err)
	}
	return 0, string(out), nil
}
