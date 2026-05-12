package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/insajin/autopus-adk/internal/cli/tui"
)

type runtimeProcessRow struct {
	PID     int
	PPID    int
	Command string
}

type staleRuntimeProcess struct {
	PID        int
	PPID       int
	Executable string
	Command    string
	Reason     string
}

var (
	listRuntimeProcesses      = listRuntimeProcessesPS
	runtimeProcessExecutable  = runtimeProcessExecutableDefault
	statRuntimeExecutable     = os.Stat
	terminateRuntimeProcessID = terminateRuntimeProcessDefault
)

func checkRuntimeProcessesText(w io.Writer, opts doctorOptions) bool {
	tui.SectionHeader(w, "Runtime Processes")
	staleLegacy := findStaleLegacyWorkerMCPProcesses()
	orphanedProviders := findOrphanedOrchestraProviderProcesses()
	stale := append(staleLegacy, orphanedProviders...)
	if len(stale) == 0 {
		tui.OK(w, "legacy worker MCP: no stale processes")
		tui.OK(w, "orchestra providers: no orphaned processes")
		return true
	}

	if opts.fix {
		failed := terminateStaleRuntimeProcesses(stale)
		if len(failed) == 0 {
			tui.OK(w, fmt.Sprintf("terminated %d stale runtime process(es)", len(stale)))
			return true
		}
		for _, err := range failed {
			tui.FAIL(w, err.Error())
		}
		return false
	}

	for _, proc := range stale {
		tui.SKIP(w, fmt.Sprintf("stale runtime pid=%d %s (%s)", proc.PID, runtimeProcessDisplayRef(proc), proc.Reason))
	}
	tui.Bullet(w, "Run 'auto doctor --fix' to terminate stale runtime processes.")
	return false
}

func (r *doctorJSONReport) collectRuntimeProcessChecks(opts doctorOptions) {
	staleLegacy := findStaleLegacyWorkerMCPProcesses()
	orphanedProviders := findOrphanedOrchestraProviderProcesses()
	stale := append(staleLegacy, orphanedProviders...)
	for _, proc := range stale {
		r.data.Runtime = append(r.data.Runtime, doctorRuntimeProcessPayload(proc))
	}

	if len(stale) == 0 {
		r.checks = append(r.checks, jsonCheck{
			ID:       "doctor.runtime.legacy_worker_mcp",
			Severity: "info",
			Status:   "pass",
			Detail:   "legacy worker MCP: no stale processes",
		},
			jsonCheck{
				ID:       "doctor.runtime.orchestra_provider",
				Severity: "info",
				Status:   "pass",
				Detail:   "orchestra providers: no orphaned processes",
			})
		return
	}

	if opts.fix {
		failed := terminateStaleRuntimeProcesses(stale)
		if len(failed) == 0 {
			r.checks = append(r.checks, jsonCheck{
				ID:       "doctor.runtime.legacy_worker_mcp.cleanup",
				Severity: "info",
				Status:   "pass",
				Detail:   fmt.Sprintf("terminated %d stale runtime process(es)", len(stale)),
			})
			return
		}
		for _, err := range failed {
			r.warnings = append(r.warnings, jsonMessage{
				Code:    "stale_legacy_worker_mcp_cleanup_failed",
				Message: err.Error(),
			})
		}
		r.status = jsonStatusWarn
		r.checks = append(r.checks, jsonCheck{
			ID:       "doctor.runtime.legacy_worker_mcp.cleanup",
			Severity: "warning",
			Status:   "warn",
			Detail:   fmt.Sprintf("%d stale legacy worker MCP process(es) could not be terminated", len(failed)),
		})
		return
	}

	r.status = jsonStatusWarn
	if len(staleLegacy) > 0 {
		r.warnings = append(r.warnings, jsonMessage{
			Code:    "stale_legacy_worker_mcp",
			Message: fmt.Sprintf("%d stale legacy worker MCP process(es) are still running; run 'auto doctor --fix'", len(staleLegacy)),
		})
	}
	if len(orphanedProviders) > 0 {
		r.warnings = append(r.warnings, jsonMessage{
			Code:    "orphaned_orchestra_provider",
			Message: fmt.Sprintf("%d orphaned orchestra provider process(es) are still running; run 'auto doctor --fix'", len(orphanedProviders)),
		})
	}
	r.checks = append(r.checks, jsonCheck{
		ID:       "doctor.runtime.stale_processes",
		Severity: "warning",
		Status:   "warn",
		Detail:   fmt.Sprintf("%d stale runtime process(es) detected", len(stale)),
	})
}

func findStaleLegacyWorkerMCPProcesses() []staleRuntimeProcess {
	rows, err := listRuntimeProcesses()
	if err != nil {
		return nil
	}

	var stale []staleRuntimeProcess
	for _, row := range rows {
		if !isLegacyWorkerMCPCommand(row.Command) {
			continue
		}
		exe, err := runtimeProcessExecutable(row.PID)
		if err != nil || strings.TrimSpace(exe) == "" {
			continue
		}
		exe = normalizeExecutablePath(exe)
		if ok, reason := isStaleAutoExecutable(exe); ok {
			stale = append(stale, staleRuntimeProcess{
				PID:        row.PID,
				PPID:       row.PPID,
				Executable: exe,
				Command:    row.Command,
				Reason:     reason,
			})
		}
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i].PID < stale[j].PID })
	return stale
}

func terminateStaleRuntimeProcesses(stale []staleRuntimeProcess) []error {
	var failed []error
	for _, proc := range stale {
		if err := terminateRuntimeProcessID(proc.PID); err != nil {
			failed = append(failed, fmt.Errorf("terminate stale runtime process pid=%d: %w", proc.PID, err))
		}
	}
	return failed
}

func runtimeProcessDisplayRef(proc staleRuntimeProcess) string {
	if strings.TrimSpace(proc.Executable) != "" {
		return "exe=" + proc.Executable
	}
	if strings.TrimSpace(proc.Command) != "" {
		return "cmd=" + proc.Command
	}
	return "cmd=<unknown>"
}

func isLegacyWorkerMCPCommand(command string) bool {
	command = strings.Join(strings.Fields(command), " ")
	return strings.Contains(command, "auto worker mcp-serve") ||
		strings.Contains(command, "auto worker mcp-server")
}

func isStaleAutoExecutable(path string) (bool, string) {
	base := filepath.Base(path)
	switch base {
	case "auto.old":
		return true, "old auto binary left alive after self-update"
	case "auto":
		if _, err := statRuntimeExecutable(path); errors.Is(err, os.ErrNotExist) {
			return true, "auto binary path no longer exists"
		}
	}
	return false, ""
}

func normalizeExecutablePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimSuffix(path, " (deleted)")
	return filepath.Clean(path)
}

func listRuntimeProcessesPS() ([]runtimeProcessRow, error) {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,command=").Output()
	if err != nil {
		return nil, err
	}

	var rows []runtimeProcessRow
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		command := strings.Join(parts[2:], " ")
		rows = append(rows, runtimeProcessRow{PID: pid, PPID: ppid, Command: command})
	}
	return rows, nil
}

func runtimeProcessExecutableDefault(pid int) (string, error) {
	if runtime.GOOS == "linux" {
		return os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "exe"))
	}

	out, err := exec.Command("lsof", "-w", "-a", "-p", strconv.Itoa(pid), "-d", "txt", "-Fn").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "n") {
			continue
		}
		path := strings.TrimPrefix(line, "n")
		base := filepath.Base(normalizeExecutablePath(path))
		if base == "auto" || base == "auto.old" {
			return path, nil
		}
	}
	return "", fmt.Errorf("auto executable not found for pid %d", pid)
}

func terminateRuntimeProcessDefault(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}
