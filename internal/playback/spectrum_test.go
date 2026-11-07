package playback

import (
	"math"
	"testing"
	"time"
)

func TestSpectrumTracksToneAndResetsAfterSeek(t *testing.T) {
	var samples [fftSize]float64
	const bin = 40
	for i := range samples {
		samples[i] = .05 * math.Sin(2*math.Pi*bin*float64(i)/fftSize)
	}
	bands := frequencyBands(samples)
	peak := 0
	for i, v := range bands {
		if v > bands[peak] {
			peak = i
		}
	}
	if peak != bin || bands[peak] < .8 {
		t.Fatalf("tone peak=%d level=%f", peak, bands[peak])
	}
	if bands[200] > .05 {
		t.Fatal("tone leaked into unrelated frequencies")
	}
	d := &Decoder{}
	before, after := &process{}, &process{}
	d.analysis.reset(before, 0)
	pcm := make([]float32, fftSize*2)
	for i, v := range samples {
		pcm[2*i] = float32(v)
		pcm[2*i+1] = float32(v)
	}
	d.analysis.capture(before, pcm)
	if d.Spectrum(time.Second)[bin] == 0 {
		t.Fatal("decoder did not capture the tone")
	}
	d.analysis.reset(after, Rate*30)
	d.analysis.capture(before, pcm)
	if d.Spectrum(30 * time.Second)[bin] != 0 {
		t.Fatal("pre-seek process polluted the new spectrum")
	}
	d.analysis.capture(after, pcm)
	if len(d.analysis.frames) != 1 || d.analysis.frames[0].at < 30*time.Second {
		t.Fatal("new spectrum has the wrong playback time")
	}
}
