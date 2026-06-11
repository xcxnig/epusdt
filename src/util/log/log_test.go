package log

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestSetLevelUpdatesAtomicLevelAndLogsChange(t *testing.T) {
	oldSugar := Sugar
	oldLevel := atomicLevel.Level()
	t.Cleanup(func() {
		Sugar = oldSugar
		atomicLevel.SetLevel(oldLevel)
	})

	core, observed := observer.New(zapcore.WarnLevel)
	Sugar = zap.New(core).Sugar()
	atomicLevel.SetLevel(zapcore.ErrorLevel)

	if err := SetLevel(" DEBUG "); err != nil {
		t.Fatalf("SetLevel(DEBUG): %v", err)
	}
	if got := CurrentLevel(); got != "debug" {
		t.Fatalf("CurrentLevel = %q, want debug", got)
	}
	logs := observed.FilterMessage("[log] level changed: error -> debug").All()
	if len(logs) != 1 {
		t.Fatalf("level change logs = %d, want 1; all=%v", len(logs), observed.All())
	}

	if err := SetLevel("debug"); err != nil {
		t.Fatalf("SetLevel(debug no-op): %v", err)
	}
	logs = observed.FilterMessage("[log] level changed: error -> debug").All()
	if len(logs) != 1 {
		t.Fatalf("no-op level change emitted extra logs: %d", len(logs))
	}

	if err := SetLevel("fatal"); err == nil {
		t.Fatal("SetLevel(fatal) succeeded, want error")
	}
}
