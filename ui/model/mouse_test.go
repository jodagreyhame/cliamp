package model

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestVolumeFromBarFrac(t *testing.T) {
	tests := []struct {
		frac, min, want float64
	}{
		{0, -50, -50},
		{1, -50, 6},
		{0.5, -50, -22},
		{-1, -30, -30},
		{2, -30, 6},
	}
	for _, tt := range tests {
		if got := volumeFromBarFrac(tt.frac, tt.min); got != tt.want {
			t.Fatalf("volumeFromBarFrac(%v, %v) = %v, want %v", tt.frac, tt.min, got, tt.want)
		}
	}
}

func TestHandleMouseDragSetsVolume(t *testing.T) {
	m := newVolumeTestModel()
	m.hits.vol = hitRect{x: 10, y: 8, w: 20}
	m.hits.volBar = hitRect{x: 14, y: 8, w: 11}

	if cmd := m.handleMouse(tea.MouseClickMsg{X: 14, Y: 8, Button: tea.MouseLeft}); cmd != nil {
		t.Fatalf("click cmd = %v, want nil", cmd)
	}
	if m.focus != focusVolume {
		t.Fatalf("focus = %v, want volume", m.focus)
	}
	if got := m.player.Volume(); got != -50 {
		t.Fatalf("volume after left click = %v, want -50", got)
	}
	if !m.hits.draggingVol {
		t.Fatal("expected draggingVol after click")
	}

	m.handleMouse(tea.MouseMotionMsg{X: 24, Y: 8, Button: tea.MouseLeft})
	if got := m.player.Volume(); got != 6 {
		t.Fatalf("volume after drag right = %v, want 6", got)
	}

	m.handleMouse(tea.MouseReleaseMsg{X: 24, Y: 8, Button: tea.MouseLeft})
	if m.hits.draggingVol {
		t.Fatal("draggingVol still set after release")
	}
}

func TestHandleMouseWheelAdjustsVolume(t *testing.T) {
	m := newVolumeTestModel()
	m.focus = focusVolume
	m.handleMouse(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if got := m.player.Volume(); got != -5 {
		t.Fatalf("volume after wheel up = %v, want -5", got)
	}
	m.handleMouse(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if got := m.player.Volume(); got != -6 {
		t.Fatalf("volume after wheel down = %v, want -6", got)
	}
}

func TestViewRecordsVolumeBarHit(t *testing.T) {
	m := newVolumeTestModel()
	m.View()
	if m.hits.volBar.w < 6 {
		t.Fatalf("volBar = %+v, want a draggable bar", m.hits.volBar)
	}
	plain := ansi.Strip(m.renderControls())
	if !strings.Contains(plain, "█") && !strings.Contains(plain, "░") {
		t.Fatalf("controls = %q, want volume bar cells", plain)
	}
}

func TestVolumeBarSpanStopsAtGap(t *testing.T) {
	start, w := volumeBarSpan("VOL ██████░░░░ -6dB")
	if start != 4 || w != 10 {
		t.Fatalf("volumeBarSpan = (%d, %d), want (4, 10)", start, w)
	}
	if isSeekLine("━━━━╸━━━") != true {
		t.Fatal("isSeekLine(bar) = false")
	}
	if isSeekLine("VOL ██████") {
		t.Fatal("isSeekLine(volume) = true")
	}
}
