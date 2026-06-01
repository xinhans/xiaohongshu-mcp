package human

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	h := New()
	if h.DelayMin != 500*time.Millisecond {
		t.Errorf("expected DelayMin 500ms, got %v", h.DelayMin)
	}
	if h.DelayMax != 2000*time.Millisecond {
		t.Errorf("expected DelayMax 2000ms, got %v", h.DelayMax)
	}
	if h.ClickOffset != 3 {
		t.Errorf("expected ClickOffset 3, got %d", h.ClickOffset)
	}
}

func TestSetAggression(t *testing.T) {
	tests := []struct {
		level          string
		expectedMin    time.Duration
		expectedMax    time.Duration
		expectedOffset int
	}{
		{"low", 500 * time.Millisecond, 2000 * time.Millisecond, 3},
		{"medium", 300 * time.Millisecond, 1500 * time.Millisecond, 5},
		{"high", 200 * time.Millisecond, 800 * time.Millisecond, 8},
	}

	for _, tt := range tests {
		h := New()
		h.SetAggression(tt.level)
		if h.DelayMin != tt.expectedMin {
			t.Errorf("level %s: expected DelayMin %v, got %v", tt.level, tt.expectedMin, h.DelayMin)
		}
		if h.DelayMax != tt.expectedMax {
			t.Errorf("level %s: expected DelayMax %v, got %v", tt.level, tt.expectedMax, h.DelayMax)
		}
		if h.ClickOffset != tt.expectedOffset {
			t.Errorf("level %s: expected ClickOffset %d, got %d", tt.level, tt.expectedOffset, h.ClickOffset)
		}
	}
}

func TestCubicBezier(t *testing.T) {
	// t=0 时应该返回 p0
	result := cubicBezier(0, 0, 1, 2, 3)
	if result != 0 {
		t.Errorf("expected 0 at t=0, got %f", result)
	}

	// t=1 时应该返回 p3
	result = cubicBezier(1, 0, 1, 2, 3)
	if result != 3 {
		t.Errorf("expected 3 at t=1, got %f", result)
	}
}

func TestRandomFloat(t *testing.T) {
	min, max := 10.0, 20.0
	for i := 0; i < 100; i++ {
		r := randomFloat(min, max)
		if r < min || r > max {
			t.Errorf("randomFloat(%f, %f) = %f, out of range", min, max, r)
		}
	}
}
