package ui

import (
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// LevelMeter is a compact stereo RMS/peak meter for the player chrome.
// It is independent of the visualizer mode so output level stays visible
// even when the spectrum is hidden.
type LevelMeter struct {
	samples     [][2]float64
	level       [2]float64
	peak        [2]float64
	hold        [2]time.Duration
	targetLevel [2]float64
	targetPeak  [2]float64
	lastTick    time.Time
	samplesAt   time.Time
}

// NewLevelMeter returns an idle stereo level meter.
func NewLevelMeter() *LevelMeter {
	return &LevelMeter{samples: make([][2]float64, stereoSampleWindow)}
}

// Tick advances ballistics. stereoSamplesInto may be nil; playing=false
// decays the needles toward silence.
func (m *LevelMeter) Tick(now time.Time, playing bool, stereoSamplesInto func([][2]float64) int) {
	if m == nil {
		return
	}
	if playing && stereoSamplesInto != nil {
		m.sample(now, stereoSamplesInto)
	} else {
		m.targetLevel = [2]float64{}
		m.targetPeak = [2]float64{}
		m.samplesAt = time.Time{}
	}
	m.advance(now)
}

// Animating reports whether the meter is still moving or showing energy.
func (m *LevelMeter) Animating() bool {
	if m == nil {
		return false
	}
	for channel := range 2 {
		if m.level[channel] > stereoEpsilon || m.peak[channel] > stereoEpsilon ||
			math.Abs(m.level[channel]-m.targetLevel[channel]) > stereoEpsilon {
			return true
		}
	}
	return false
}

// Render draws a single-row L/R meter that fits in width cells.
func (m *LevelMeter) Render(width int) string {
	if m == nil || width <= 0 {
		return ""
	}
	label := "LVL "
	if width <= len(label)+6 {
		label = ""
	}
	inner := width - len(label)
	if inner < 6 {
		return strings.Repeat(" ", width)
	}
	// Two meters plus one gap: "L…… R……"
	leftW := inner / 2
	rightW := inner - leftW - 1
	if rightW < 3 {
		rightW = 3
		leftW = inner - rightW - 1
	}
	if leftW < 3 {
		return renderStereoMeter("", (m.level[0]+m.level[1])/2, max(m.peak[0], m.peak[1]), width)
	}
	left := renderStereoMeter("L", m.level[0], m.peak[0], leftW)
	right := renderStereoMeter("R", m.level[1], m.peak[1], rightW)
	out := label + left + " " + right
	if w := lipgloss.Width(out); w < width {
		out += strings.Repeat(" ", width-w)
	}
	return out
}

func (m *LevelMeter) sample(now time.Time, stereoSamplesInto func([][2]float64) int) {
	if !m.samplesAt.IsZero() && !now.IsZero() && now.Sub(m.samplesAt) < TickAnalyze {
		return
	}
	n := stereoSamplesInto(m.samples)
	m.targetLevel, m.targetPeak = stereoMetrics(m.samples[:n])
	if !now.IsZero() {
		m.samplesAt = now
	}
}

func (m *LevelMeter) advance(now time.Time) {
	dt := TickAnim
	if !now.IsZero() && !m.lastTick.IsZero() {
		dt = now.Sub(m.lastTick)
	}
	if dt <= 0 || dt > maxSmoothDtFrames*TickAnim {
		dt = TickAnim
	}
	m.lastTick = now
	dtSeconds := dt.Seconds()

	for channel := range 2 {
		rate := stereoFallRate
		if m.targetLevel[channel] > m.level[channel] {
			rate = stereoRiseRate
		}
		m.level[channel] += (m.targetLevel[channel] - m.level[channel]) * (1 - math.Exp(-rate*dtSeconds))

		switch {
		case m.targetPeak[channel] > m.peak[channel]:
			m.peak[channel] = m.targetPeak[channel]
			m.hold[channel] = stereoPeakHold
		case m.hold[channel] > 0:
			m.hold[channel] = max(0, m.hold[channel]-dt)
		default:
			m.peak[channel] = max(m.level[channel], m.peak[channel]-stereoPeakFallRate*dtSeconds)
		}
	}
}
