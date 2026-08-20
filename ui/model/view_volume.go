package model

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bjarneo/cliamp/ui"
)

func (m Model) volumeLabel() string {
	if m.focus == focusVolume {
		return activeToggle.Render("VOL ▸ ")
	}
	return labelStyle.Render("VOL ")
}

func (m Model) renderVolumeValue() string {
	db := 0.0
	if m.player != nil {
		db = m.player.Volume()
	}
	dbStr := fmt.Sprintf("%+.0fdB", db)
	if m.focus == focusVolume {
		return m.volumeLabel() + activeToggle.Render(dbStr)
	}
	return m.volumeLabel() + dimStyle.Render(dbStr)
}

func (m Model) renderVolumeBar(barW int) string {
	if barW < 1 || m.player == nil {
		return ""
	}
	vol := m.player.Volume()
	volMin := m.player.VolumeMin()
	span := 6 - volMin
	frac := 0.0
	if span > 0 {
		frac = max(0, min(1, (vol-volMin)/span))
	}
	filled := int(frac * float64(barW))
	fillStyle := volBarStyle
	if m.focus == focusVolume {
		fillStyle = activeToggle
	}
	return fillStyle.Render(strings.Repeat("█", filled)) +
		dimStyle.Render(strings.Repeat("░", barW-filled))
}

func (m Model) renderCompactLevel() string {
	if m.levelMeter == nil {
		return ""
	}
	return m.levelMeter.Render(11)
}

func (m Model) renderLevelRow() string {
	if m.levelMeter == nil || ui.PanelWidth < 12 {
		return ""
	}
	return m.levelMeter.Render(ui.PanelWidth)
}

func (m Model) renderVolumeCluster(left string) string {
	db := 0.0
	if m.player != nil {
		db = m.player.Volume()
	}
	dbStr := fmt.Sprintf(" %+.0fdB", db)
	monoStr := ""
	if m.player != nil && m.player.Mono() {
		monoStr = " " + activeToggle.Render("[M]")
	}

	volLabel := m.volumeLabel()
	volSuffix := dimStyle.Render(dbStr) + monoStr
	if m.focus == focusVolume {
		volSuffix = activeToggle.Render(dbStr) + monoStr
	}

	leftW := lipgloss.Width(left)
	remain := max(0, ui.PanelWidth-leftW-1)
	labelW := lipgloss.Width(volLabel)
	barW := max(6, remain-labelW-lipgloss.Width(volSuffix))
	right := volLabel + m.renderVolumeBar(barW) + volSuffix
	gap := max(1, ui.PanelWidth-leftW-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}
