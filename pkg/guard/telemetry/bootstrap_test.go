package telemetry

import (
	"os"
	"sync"
	"testing"
)

// preserveEmitter snapshots the emitter slot, clears it, resets the bootstrap
// gate, and restores both on test cleanup. Tests exercising EnsureDefault MUST
// call this so a prior test's installed emitter does not skew the result.
func preserveEmitter(t *testing.T) {
	t.Helper()
	prev := GetEmitter()
	ResetBootstrapForTest()
	SetEmitter(nil)
	t.Cleanup(func() {
		ResetBootstrapForTest()
		SetEmitter(prev)
	})
}

func TestBootstrap_InstallsDefaultEmitterOnce(t *testing.T) {
	preserveEmitter(t)
	if GetEmitter() != nil {
		t.Fatal("emitter must be nil before EnsureDefault")
	}
	EnsureDefault()
	first := GetEmitter()
	if first == nil {
		t.Fatal("EnsureDefault must install an emitter")
	}
	if _, ok := first.(*Writer); !ok {
		t.Errorf("default emitter must be *Writer, got %T", first)
	}
	// second call must not replace the installed emitter (sync.Once)
	EnsureDefault()
	if GetEmitter() != first {
		t.Error("second EnsureDefault must not replace the installed emitter")
	}
}

func TestBootstrap_NoOpWhenEmitterAlreadySet(t *testing.T) {
	preserveEmitter(t)
	cap := &CaptureEmitter{}
	SetEmitter(cap)
	EnsureDefault()
	if GetEmitter() != cap {
		t.Errorf("EnsureDefault must not overwrite a pre-installed emitter")
	}
}

func TestBootstrap_NoDiskSideEffectOnInit(t *testing.T) {
	preserveEmitter(t)
	dir := DefaultDir()
	// Snapshot existence before. If the directory already exists from a prior
	// run, we cannot prove non-creation; skip in that case.
	if _, err := os.Stat(dir); err == nil {
		t.Skipf("%s already exists; cannot prove non-creation in this env", dir)
	}
	EnsureDefault()
	if _, err := os.Stat(dir); err == nil {
		t.Errorf("EnsureDefault must not create %s on init", dir)
	}
	// emitter is installed, but still no directory until Append fires.
	if GetEmitter() == nil {
		t.Fatal("EnsureDefault must install an emitter")
	}
	if _, err := os.Stat(dir); err == nil {
		t.Errorf("install-only EnsureDefault must not create %s", dir)
	}
}

func TestBootstrap_ConcurrentSafe(t *testing.T) {
	preserveEmitter(t)
	const N = 32
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			EnsureDefault()
		}()
	}
	wg.Wait()
	if GetEmitter() == nil {
		t.Fatal("emitter must be installed after concurrent EnsureDefault")
	}
	if _, ok := GetEmitter().(*Writer); !ok {
		t.Errorf("concurrent EnsureDefault must install *Writer")
	}
}

func TestBootstrap_AfterResetReinstalls(t *testing.T) {
	preserveEmitter(t)
	EnsureDefault()
	first := GetEmitter()
	if first == nil {
		t.Fatal("first EnsureDefault must install")
	}
	// Reset the gate AND clear the slot — only then can a fresh install occur.
	ResetBootstrapForTest()
	SetEmitter(nil)
	EnsureDefault()
	second := GetEmitter()
	if second == nil {
		t.Fatal("EnsureDefault after reset must reinstall")
	}
	if second == first {
		t.Errorf("after reset+clear, the new emitter should be a fresh instance")
	}
}

func TestBootstrap_DoesNotCallAppend(t *testing.T) {
	preserveEmitter(t)
	// Pre-install a CaptureEmitter; EnsureDefault must skip it without calling Append.
	cap := &CaptureEmitter{}
	SetEmitter(cap)
	EnsureDefault()
	if cap.Len() != 0 {
		t.Errorf("EnsureDefault must not call Append on the installed emitter, got %d records", cap.Len())
	}
}
