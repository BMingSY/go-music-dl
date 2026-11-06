package playback

import (
	"math"
	"math/cmplx"
	"sync"
	"time"
)

const fftSize = 1024
const SpectrumBands = 128

type spectrumFrame struct {
	at    time.Duration
	bands [SpectrumBands]float32
}
type analyzer struct {
	mu      sync.Mutex
	source  *process
	cursor  int64
	samples [fftSize]float64
	used    int
	frames  []spectrumFrame
}

func (a *analyzer) reset(p *process, frame int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.source = p
	a.cursor = frame
	a.used = 0
	a.frames = nil
}
func (a *analyzer) capture(p *process, pcm []float32) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if p != a.source {
		return
	}
	for i := 0; i+1 < len(pcm); i += 2 {
		a.samples[a.used] = float64(pcm[i]+pcm[i+1]) * .5
		a.used++
		a.cursor++
		if a.used == fftSize {
			a.frames = append(a.frames, spectrumFrame{at: time.Duration(float64(a.cursor-fftSize/2) / Rate * float64(time.Second)), bands: frequencyBands(a.samples)})
			if len(a.frames) > 160 {
				copy(a.frames, a.frames[len(a.frames)-160:])
				a.frames = a.frames[:160]
			}
			a.used = 0
		}
	}
}
func (d *Decoder) Spectrum(at time.Duration) [SpectrumBands]float32 {
	a := &d.analysis
	a.mu.Lock()
	defer a.mu.Unlock()
	var result [SpectrumBands]float32
	for _, frame := range a.frames {
		result = frame.bands
		if frame.at >= at {
			break
		}
	}
	return result
}
func frequencyBands(samples [fftSize]float64) [SpectrumBands]float32 {
	var data [fftSize]complex128
	for i, v := range samples {
		data[i] = complex(v*(.5-.5*math.Cos(2*math.Pi*float64(i)/(fftSize-1))), 0)
	}
	for i, j := 1, 0; i < fftSize; i++ {
		bit := fftSize >> 1
		for j&bit != 0 {
			j ^= bit
			bit >>= 1
		}
		j ^= bit
		if i < j {
			data[i], data[j] = data[j], data[i]
		}
	}
	for size := 2; size <= fftSize; size *= 2 {
		step := cmplx.Rect(1, -2*math.Pi/float64(size))
		for start := 0; start < fftSize; start += size {
			w := complex(1, 0)
			for j := 0; j < size/2; j++ {
				u, v := data[start+j], w*data[start+j+size/2]
				data[start+j] = u + v
				data[start+j+size/2] = u - v
				w *= step
			}
		}
	}
	var result [SpectrumBands]float32
	for i := range result {
		peak := 0.0
		for j := max(1, i*fftSize/(2*SpectrumBands)); j < (i+1)*fftSize/(2*SpectrumBands); j++ {
			peak = max(peak, cmplx.Abs(data[j])/fftSize*4)
		}
		if peak > 0 {
			result[i] = float32(max(0, min(1, (20*math.Log10(peak)+70)/70)))
		}
	}
	return result
}
