package ui

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestLevelMeterKeepsChannelsIndependent(t *testing.T) {
	m := NewLevelMeter()
	samples := [][2]float64{{0.75, 0.25}, {-0.75, -0.25}}
	m.Tick(time.Unix(1, 0), true, func(dst [][2]float64) int {
		return copy(dst, samples)
	})

	if m.targetLevel[0] <= m.targetLevel[1] {
		t.Fatalf("target levels = %v, want left greater than right", m.targetLevel)
	}
	if !m.Animating() {
		t.Fatal("meter with energy should be animating")
	}

	plain := ansi.Strip(m.Render(24))
	if !strings.Contains(plain, "L") || !strings.Contains(plain, "R") {
		t.Fatalf("render = %q, want L and R channel labels", plain)
	}
}

func TestLevelMeterDecaysWhenSilent(t *testing.T) {
	m := NewLevelMeter()
	samples := [][2]float64{{0.8, 0.8}}
	m.Tick(time.Unix(1, 0), true, func(dst [][2]float64) int {
		return copy(dst, samples)
	})
	if m.level[0] <= 0 {
		t.Fatalf("level after signal = %v, want energy", m.level)
	}

	// Advance far enough that hold expires and the needle falls.
	now := time.Unix(1, 0)
	for range 40 {
		now = now.Add(50 * time.Millisecond)
		m.Tick(now, false, nil)
	}
	if m.level[0] > 0.05 || m.level[1] > 0.05 {
		t.Fatalf("level after silence = %v, want decay toward 0", m.level)
	}
}

func TestLevelMeterRenderFitsWidth(t *testing.T) {
	m := NewLevelMeter()
	m.level = [2]float64{0.4, 0.2}
	m.peak = [2]float64{0.6, 0.3}
	for _, width := range []int{0, 4, 8, 12, 24, 40} {
		got := ansi.Strip(m.Render(width))
		if n := len([]rune(got)); n > width {
			t.Fatalf("width %d render = %q (%d runes)", width, got, n)
		}
	}
}

func TestLevelMeterUnityIsFullScale(t *testing.T) {
	if got := stereoDBLevel(1); math.Abs(got-1) > 1e-9 {
		t.Fatalf("unity = %v, want 1", got)
	}
	if got := stereoDBLevel(0); got != 0 {
		t.Fatalf("silence = %v, want 0", got)
	}
}
