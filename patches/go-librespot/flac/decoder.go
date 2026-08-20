package flac

import (
	"fmt"
	"io"
	"sync"

	librespot "github.com/devgianlu/go-librespot"
	mewflac "github.com/mewkiz/flac"
)

// Decoder implements a FLAC decoder using a pure-Go library so go-librespot
// can compile without libflac (required on Windows).
type Decoder struct {
	sync.Mutex

	log librespot.Logger

	SampleRate int32
	Channels   int32

	gain   float32
	input  librespot.SizedReadAtSeeker
	stream *mewflac.Stream
	buf    []float32
	pos    int64
	bps    int
}

func New(log librespot.Logger, r librespot.SizedReadAtSeeker, gain float32) (*Decoder, error) {
	stream, err := mewflac.NewSeek(r)
	if err != nil {
		return nil, fmt.Errorf("flac: %w", err)
	}
	info := stream.Info
	if info == nil {
		return nil, fmt.Errorf("flac: missing stream info")
	}
	return &Decoder{
		log:        log,
		input:      r,
		stream:     stream,
		gain:       gain,
		SampleRate: int32(info.SampleRate),
		Channels:   int32(info.NChannels),
		bps:        int(info.BitsPerSample),
	}, nil
}

func (d *Decoder) Read(p []float32) (n int, err error) {
	d.Lock()
	defer d.Unlock()
	if d.stream == nil {
		return 0, fmt.Errorf("flac: decoder closed")
	}

	scale := d.gain / float32(int32(1)<<(d.bps-1))
	for n < len(p) {
		if len(d.buf) > 0 {
			copied := copy(p[n:], d.buf)
			d.buf = d.buf[copied:]
			n += copied
			d.pos += int64(copied)
			continue
		}
		frame, err := d.stream.ParseNext()
		if err != nil {
			if n > 0 && err == io.EOF {
				return n, nil
			}
			return n, err
		}
		channels := len(frame.Subframes)
		if channels == 0 {
			continue
		}
		ns := frame.Subframes[0].NSamples
		out := make([]float32, ns*channels)
		for i := 0; i < ns; i++ {
			for ch := 0; ch < channels; ch++ {
				out[i*channels+ch] = float32(frame.Subframes[ch].Samples[i]) * scale
			}
		}
		d.buf = out
	}
	return n, nil
}

func (d *Decoder) SetPositionMs(pos int64) error {
	d.Lock()
	defer d.Unlock()
	if d.stream == nil || d.SampleRate == 0 {
		return fmt.Errorf("flac: decoder closed")
	}
	sample := uint64(pos * int64(d.SampleRate) / 1000)
	start, err := d.stream.Seek(sample)
	if err != nil {
		return fmt.Errorf("could not seek to position: %w", err)
	}
	d.buf = nil
	d.pos = int64(start) * int64(d.Channels)
	return nil
}

func (d *Decoder) PositionMs() int64 {
	d.Lock()
	defer d.Unlock()
	if d.SampleRate == 0 || d.Channels == 0 {
		return 0
	}
	return (d.pos / int64(d.Channels)) * 1000 / int64(d.SampleRate)
}

func (d *Decoder) Close() error {
	d.Lock()
	defer d.Unlock()
	d.stream = nil
	return nil
}
