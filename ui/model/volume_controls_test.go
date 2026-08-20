package model

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/bjarneo/cliamp/playlist"
	"github.com/bjarneo/cliamp/ui"
)

func newVolumeTestModel() Model {
	player := &playbackFakeEngine{volume: -6, volumeMin: -50}
	m := New(player, playlist.New(), nil, "", nil, nil, nil, &recordingConfigSaver{})
	m.layout.tier = layoutFull
	m.width = 80
	m.height = 24
	m.recomputeLayout()
	return m
}

func TestHandleVolumeKeyAdjustsWhenFocused(t *testing.T) {
	m := newVolumeTestModel()
	m.focus = focusVolume

	if cmd := m.handleKey(tea.KeyPressMsg{Code: tea.KeyRight}); cmd != nil {
		t.Fatalf("handleKey(right) cmd = %v, want nil", cmd)
	}
	if got := m.player.Volume(); got != -5 {
		t.Fatalf("volume after right = %v, want -5", got)
	}
	if got := m.volumeSaveAfter; got != volumeSaveDebounce {
		t.Fatalf("volumeSaveAfter = %v, want %v", got, volumeSaveDebounce)
	}

	if cmd := m.handleKey(tea.KeyPressMsg{Code: tea.KeyLeft}); cmd != nil {
		t.Fatalf("handleKey(left) cmd = %v, want nil", cmd)
	}
	if got := m.player.Volume(); got != -6 {
		t.Fatalf("volume after left = %v, want -6", got)
	}
}

func TestTabCycleIncludesVolume(t *testing.T) {
	m := newVolumeTestModel()
	m.focus = focusPlaylist

	m.handleKey(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focus != focusEQ {
		t.Fatalf("focus after first Tab = %v, want EQ", m.focus)
	}
	m.handleKey(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focus != focusVolume {
		t.Fatalf("focus after second Tab = %v, want Volume", m.focus)
	}
	m.handleKey(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focus != focusSpeed {
		t.Fatalf("focus after third Tab = %v, want Speed", m.focus)
	}
}

func TestMinimalLayoutRejectsVolumeFocus(t *testing.T) {
	m := newVolumeTestModel()
	m.focus = focusVolume
	m.width = 40
	m.height = 10
	m.recomputeLayout()
	m.normalizeMainFocus()
	if m.focus != focusPlaylist {
		t.Fatalf("focus = %v, want playlist at minimal size", m.focus)
	}
}

func TestRenderControlsShowsLevelAndVolume(t *testing.T) {
	old := ui.PanelWidth
	ui.PanelWidth = 80
	t.Cleanup(func() { ui.PanelWidth = old })

	m := newVolumeTestModel()
	m.focus = focusVolume
	plain := ansi.Strip(m.renderControls())
	if !strings.Contains(plain, "VOL") {
		t.Fatalf("controls = %q, want VOL adjuster", plain)
	}
	if !strings.Contains(plain, "-6dB") {
		t.Fatalf("controls = %q, want current volume", plain)
	}
	level := ansi.Strip(m.renderLevelRow())
	if !strings.Contains(level, "L") || !strings.Contains(level, "R") {
		t.Fatalf("level row = %q, want L/R level meter", level)
	}
}

func TestFlushPendingVolumeSavePersists(t *testing.T) {
	saver := &recordingConfigSaver{}
	player := &playbackFakeEngine{volume: -12, volumeMin: -50}
	m := New(player, playlist.New(), nil, "", nil, nil, nil, saver)
	m.changeVolume(2)
	m.flushPendingVolumeSave()

	if got := saver.values["volume"]; got != "-10" {
		t.Fatalf("saved volume = %q, want -10", got)
	}
	if m.volumeSaveAfter != 0 {
		t.Fatalf("volumeSaveAfter = %v, want 0", m.volumeSaveAfter)
	}
}

func TestVolumeDeltaFromWindowsKeyNames(t *testing.T) {
	tests := []struct {
		msg  tea.KeyPressMsg
		want float64
	}{
		{msg: tea.KeyPressMsg{Text: "+"}, want: 1},
		{msg: tea.KeyPressMsg{Text: "="}, want: 1},
		{msg: tea.KeyPressMsg{Text: "-"}, want: -1},
		{msg: tea.KeyPressMsg{Code: '+'}, want: 1},
	}
	for _, tt := range tests {
		got, ok := volumeDeltaFromKey(tt.msg)
		if !ok || got != tt.want {
			t.Fatalf("volumeDeltaFromKey(%q) = %v, %v, want %v, true", tt.msg.String(), got, ok, tt.want)
		}
	}
	m := newVolumeTestModel()
	m.handleKey(tea.KeyPressMsg{Text: "+"})
	if got := m.player.Volume(); got != -5 {
		t.Fatalf("volume after + = %v, want -5", got)
	}
	m.handleKey(tea.KeyPressMsg{Text: "-"})
	if got := m.player.Volume(); got != -6 {
		t.Fatalf("volume after - = %v, want -6", got)
	}
}

func TestCompactControlsShowLevelAndVolume(t *testing.T) {
	old := ui.PanelWidth
	ui.PanelWidth = 50
	t.Cleanup(func() { ui.PanelWidth = old })

	m := newVolumeTestModel()
	m.layout.tier = layoutCompact
	m.focus = focusVolume
	plain := ansi.Strip(m.renderCompactControls())
	if !strings.Contains(plain, "VOL") || !strings.Contains(plain, "-6dB") {
		t.Fatalf("compact controls = %q, want volume adjuster", plain)
	}
	if !strings.Contains(plain, "L") || !strings.Contains(plain, "R") {
		t.Fatalf("compact controls = %q, want L/R level meter", plain)
	}
}
