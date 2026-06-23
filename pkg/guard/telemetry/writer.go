package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultRetentionDays = 7
	defaultMaxFileBytes  = 10 * 1024 * 1024 // 10 MiB
	defaultSubdir        = "command_guard"
)

// Emitter is the minimal interface the package-level Emit forwards to. Both
// the production Writer and the test CaptureEmitter implement it.
type Emitter interface {
	Append(Record) bool
}

// Writer is the append-only NDJSON writer with daily rotate and retention.
// Append serializes one record per line and never escalates errors: every
// failure path increments WriteErrors and returns false.
type Writer struct {
	mu              sync.Mutex
	dir             string
	retentionDays   int
	maxFileBytes    int64
	writeErrorCount uint64
	clock           func() time.Time
}

// NewWriter constructs a Writer rooted at dir. dir is created on first append
// (idempotent). Use DefaultDir() for the conventional location.
func NewWriter(dir string) *Writer {
	return &Writer{
		dir:           dir,
		retentionDays: defaultRetentionDays,
		maxFileBytes:  defaultMaxFileBytes,
		clock:         func() time.Time { return time.Now().UTC() },
	}
}

// SetMaxFileBytes overrides the rotate threshold (test helper).
func (w *Writer) SetMaxFileBytes(n int64) { w.maxFileBytes = n }

// SetRetentionDays overrides the retention window (test helper).
func (w *Writer) SetRetentionDays(n int) { w.retentionDays = n }

// SetClock overrides the time source (test helper).
func (w *Writer) SetClock(c func() time.Time) { w.clock = c }

// WriteErrors returns the cumulative write-failure counter (atomic).
func (w *Writer) WriteErrors() uint64 {
	return atomic.LoadUint64(&w.writeErrorCount)
}

// Append serializes rec and writes one NDJSON line to the current day's file.
// Validation/marshal/mkdir/open/write failures all silently increment the
// counter. Hook callers MUST ignore the return value — telemetry is fire-and-
// forget by policy.
func (w *Writer) Append(rec Record) bool {
	if err := rec.ValidateRequired(); err != nil {
		atomic.AddUint64(&w.writeErrorCount, 1)
		return false
	}
	data, err := json.Marshal(rec)
	if err != nil {
		atomic.AddUint64(&w.writeErrorCount, 1)
		return false
	}
	line := append(data, '\n')

	w.mu.Lock()
	defer w.mu.Unlock()

	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		atomic.AddUint64(&w.writeErrorCount, 1)
		return false
	}
	w.sweepRetention()
	path, err := w.targetFile(int64(len(line)))
	if err != nil {
		atomic.AddUint64(&w.writeErrorCount, 1)
		return false
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		atomic.AddUint64(&w.writeErrorCount, 1)
		return false
	}
	defer f.Close()
	if _, err := f.Write(line); err != nil {
		atomic.AddUint64(&w.writeErrorCount, 1)
		return false
	}
	return true
}

func (w *Writer) targetFile(newLineSize int64) (string, error) {
	today := w.clock().Format("2006-01-02")
	base := filepath.Join(w.dir, today+".ndjson")
	info, err := os.Stat(base)
	if errors.Is(err, fs.ErrNotExist) {
		return base, nil
	}
	if err != nil {
		return "", err
	}
	if info.Size()+newLineSize <= w.maxFileBytes {
		return base, nil
	}
	for i := 1; i < 10000; i++ {
		p := filepath.Join(w.dir, fmt.Sprintf("%s-%d.ndjson", today, i))
		st, serr := os.Stat(p)
		if errors.Is(serr, fs.ErrNotExist) {
			return p, nil
		}
		if serr != nil {
			return "", serr
		}
		if st.Size()+newLineSize <= w.maxFileBytes {
			return p, nil
		}
	}
	return "", fmt.Errorf("no rotate slot available for %s", today)
}

func (w *Writer) sweepRetention() {
	cutoff := w.clock().AddDate(0, 0, -w.retentionDays)
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".ndjson") {
			continue
		}
		trimmed := strings.TrimSuffix(name, ".ndjson")
		if len(trimmed) < 10 {
			continue
		}
		t, perr := time.Parse("2006-01-02", trimmed[:10])
		if perr != nil {
			continue
		}
		if t.Before(cutoff) {
			_ = os.Remove(filepath.Join(w.dir, name))
		}
	}
}

// --- package-level emitter ------------------------------------------------

var (
	emitterMu sync.RWMutex
	emitter   Emitter
)

// SetEmitter installs (or replaces) the package emitter. Pass nil to disable
// emit entirely (default state — production startup is expected to install a
// Writer; tests typically install a CaptureEmitter).
func SetEmitter(e Emitter) {
	emitterMu.Lock()
	emitter = e
	emitterMu.Unlock()
}

// GetEmitter returns the currently installed emitter (or nil).
func GetEmitter() Emitter {
	emitterMu.RLock()
	defer emitterMu.RUnlock()
	return emitter
}

// Emit forwards a record to the installed emitter. Returns false when no
// emitter is installed or when the emitter dropped the record. Never blocks
// the caller's decision path.
func Emit(rec Record) bool {
	emitterMu.RLock()
	e := emitter
	emitterMu.RUnlock()
	if e == nil {
		return false
	}
	return e.Append(rec)
}

// DefaultDir returns the conventional telemetry directory relative to CWD.
// The .autopus/telemetry/ prefix is already covered by the repo .gitignore.
func DefaultDir() string {
	return filepath.Join(".autopus", "telemetry", defaultSubdir)
}

// CaptureEmitter is a test helper that stores accepted records in memory. It
// validates each record (rejecting and counting invalid ones) so unit tests
// can assert both the "ok" path and the "no_secret_raw_args invariant" path.
type CaptureEmitter struct {
	mu             sync.Mutex
	records        []Record
	rejectedCount  uint64
}

// Append stores the record if it passes ValidateRequired.
func (c *CaptureEmitter) Append(rec Record) bool {
	if err := rec.ValidateRequired(); err != nil {
		atomic.AddUint64(&c.rejectedCount, 1)
		return false
	}
	c.mu.Lock()
	c.records = append(c.records, rec)
	c.mu.Unlock()
	return true
}

// Records returns a snapshot copy of the captured records.
func (c *CaptureEmitter) Records() []Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Record, len(c.records))
	copy(out, c.records)
	return out
}

// Len reports how many records have been accepted.
func (c *CaptureEmitter) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.records)
}

// Rejected reports the count of records that failed ValidateRequired.
func (c *CaptureEmitter) Rejected() uint64 {
	return atomic.LoadUint64(&c.rejectedCount)
}
