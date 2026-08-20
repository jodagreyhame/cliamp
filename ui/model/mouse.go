package model

import (
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type hitRect struct {
	x, y, w int
}

func (r hitRect) contains(x, y int) bool {
	return r.w > 0 && y == r.y && x >= r.x && x < r.x+r.w
}

type hitTargets struct {
	vol, volBar, seek         hitRect
	draggingVol, draggingSeek bool
}

func volumeFromBarFrac(frac, volMin float64) float64 {
	frac = max(0, min(1, frac))
	return volMin + frac*(6-volMin)
}

func volumeBarSpan(plain string) (start, width int) {
	start = -1
	for i, r := range []rune(plain) {
		if r == '█' || r == '░' {
			if start < 0 {
				start = i
			}
			width++
			continue
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 {
		return 0, 0
	}
	return start, width
}

func isSeekLine(plain string) bool {
	trimmed := strings.TrimSpace(plain)
	if trimmed == "" {
		return false
	}
	for _, r := range trimmed {
		if r != '━' && r != '╸' {
			return false
		}
	}
	return true
}

func (m Model) recordHits(rendered string) {
	if m.hits == nil {
		return
	}
	var vol, volBar, seek hitRect
	for y, line := range strings.Split(rendered, "\n") {
		plain := ansi.Strip(line)
		if strings.Contains(plain, "VOL") {
			if start, w := volumeBarSpan(plain); w > 0 {
				volBar = hitRect{x: start, y: y, w: w}
				hitX := start
				if i := strings.Index(plain, "VOL"); i >= 0 && i < start {
					hitX = i
				}
				vol = hitRect{x: hitX, y: y, w: start + w - hitX}
			} else if i := strings.Index(plain, "VOL"); i >= 0 {
				vol = hitRect{x: i, y: y, w: lipgloss.Width(plain) - i}
				volBar = vol
			}
		}
		if isSeekLine(plain) {
			x := strings.IndexFunc(plain, func(r rune) bool { return !unicode.IsSpace(r) })
			if x < 0 {
				continue
			}
			seek = hitRect{x: x, y: y, w: lipgloss.Width(strings.TrimSpace(plain))}
		}
	}
	m.hits.vol, m.hits.volBar, m.hits.seek = vol, volBar, seek
}

func (m *Model) setVolumeFromX(x int) {
	if m.hits == nil || m.player == nil {
		return
	}
	bar := m.hits.volBar
	if bar.w <= 0 {
		bar = m.hits.vol
	}
	if bar.w <= 0 {
		return
	}
	frac := 1.0
	if bar.w > 1 {
		frac = float64(x-bar.x) / float64(bar.w-1)
	}
	m.setVolume(volumeFromBarFrac(frac, m.player.VolumeMin()))
}

func (m *Model) seekFromX(x int) tea.Cmd {
	if m.hits == nil || m.player == nil || m.hits.seek.w <= 0 {
		return nil
	}
	dur := m.player.Duration()
	if dur <= 0 {
		dur = m.cachedDur
	}
	if dur <= 0 {
		return nil
	}
	frac := 1.0
	if m.hits.seek.w > 1 {
		frac = float64(x-m.hits.seek.x) / float64(m.hits.seek.w-1)
	}
	frac = max(0, min(1, frac))
	return m.seekAbsolute(time.Duration(float64(dur) * frac))
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if m.hits == nil {
		return nil
	}
	mouse := msg.Mouse()
	switch msg.(type) {
	case tea.MouseClickMsg:
		if mouse.Button != tea.MouseLeft {
			return nil
		}
		if m.hits.vol.contains(mouse.X, mouse.Y) {
			m.hits.draggingVol = true
			m.hits.draggingSeek = false
			m.focus = focusVolume
			m.setVolumeFromX(mouse.X)
			return nil
		}
		if m.hits.seek.contains(mouse.X, mouse.Y) {
			m.hits.draggingSeek = true
			m.hits.draggingVol = false
			return m.seekFromX(mouse.X)
		}
		m.hits.draggingVol = false
		m.hits.draggingSeek = false
	case tea.MouseMotionMsg:
		if m.hits.draggingVol {
			m.setVolumeFromX(mouse.X)
		}
		if m.hits.draggingSeek {
			return m.seekFromX(mouse.X)
		}
	case tea.MouseReleaseMsg:
		m.hits.draggingVol = false
		m.hits.draggingSeek = false
	case tea.MouseWheelMsg:
		if m.hits.vol.contains(mouse.X, mouse.Y) || m.focus == focusVolume {
			switch mouse.Button {
			case tea.MouseWheelUp:
				m.changeVolume(1)
			case tea.MouseWheelDown:
				m.changeVolume(-1)
			}
		}
	}
	return nil
}
