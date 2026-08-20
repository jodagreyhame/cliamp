package vorbis

import (
	"fmt"
	"io"
	"sync"

	librespot "github.com/devgianlu/go-librespot"
	"github.com/jfreymuth/oggvorbis"
)

// Decoder implements an OggVorbis decoder using a pure-Go library so
// go-librespot can compile without libogg/libvorbis (required on Windows).
type Decoder struct {
	sync.Mutex

	log librespot.Logger

	SampleRate int32
	Channels   int32

	gain  float32
	input librespot.SizedReadAtSeeker
	r     *oggvorbis.Reader
}

func New(log librespot.Logger, r librespot.SizedReadAtSeeker, _ *MetadataPage, gain float32) (*Decoder, error) {
	dec, err := oggvorbis.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("vorbis: %w", err)
	}
	return &Decoder{
		log:        log,
		input:      r,
		r:          dec,
		gain:       gain,
		SampleRate: int32(dec.SampleRate()),
		Channels:   int32(dec.Channels()),
	}, nil
}

func (d *Decoder) Close() {
	d.Lock()
	defer d.Unlock()
	d.r = nil
}

func (d *Decoder) Read(p []float32) (n int, err error) {
	d.Lock()
	defer d.Unlock()
	if d.r == nil {
		return 0, fmt.Errorf("decoder: decoder has already been closed")
	}
	for n < len(p) {
		nn, rerr := d.r.Read(p[n:])
		if d.gain != 1 {
			for i := range p[n : n+nn] {
				p[n+i] *= d.gain
			}
		}
		n += nn
		if rerr != nil {
			return n, rerr
		}
		if nn == 0 {
			return n, io.EOF
		}
	}
	return n, nil
}

func (d *Decoder) SetPositionMs(pos int64) error {
	d.Lock()
	defer d.Unlock()
	if d.r == nil {
		return fmt.Errorf("decoder: decoder has already been closed")
	}
	sample := pos * int64(d.SampleRate) / 1000
	if err := d.r.SetPosition(sample); err != nil {
		return fmt.Errorf("failed seeking vorbis stream: %w", err)
	}
	return nil
}

func (d *Decoder) PositionMs() int64 {
	d.Lock()
	defer d.Unlock()
	if d.r == nil || d.SampleRate == 0 {
		return 0
	}
	return d.r.Position() * 1000 / int64(d.SampleRate)
}
