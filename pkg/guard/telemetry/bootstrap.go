package telemetry

import "sync"

// bootstrapOnce gates the default-emitter install so concurrent calls from
// worker and orchestra startup paths collapse into a single installation.
var bootstrapOnce sync.Once

// EnsureDefault installs a Writer-backed default emitter on first call. It is
// idempotent (sync.Once) and concurrency-safe: subsequent calls — including
// the second startup path's call — become no-ops.
//
// If an emitter is already installed (e.g. tests pre-installed a CaptureEmitter
// before invoking the hook), EnsureDefault yields and does NOT overwrite it.
//
// Disk side-effects: NONE here. NewWriter only stores the directory string;
// mkdir / file create happen lazily inside Writer.Append, which is reached only
// when the command-guard hook actually emits in a non-disabled mode. As a
// result, mode==disabled environments never create the telemetry directory or
// any NDJSON file just because EnsureDefault was called.
func EnsureDefault() {
	bootstrapOnce.Do(func() {
		emitterMu.Lock()
		defer emitterMu.Unlock()
		if emitter != nil {
			return
		}
		emitter = NewWriter(DefaultDir())
	})
}

// ResetBootstrapForTest resets the bootstrap gate so a subsequent EnsureDefault
// call can re-install. TEST-ONLY: production code never calls this. The reset
// is not concurrency-safe with live EnsureDefault calls and assumes the test
// is serial.
func ResetBootstrapForTest() {
	bootstrapOnce = sync.Once{}
}
