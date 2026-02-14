package visualizer

import "testing"

func TestWaveform_SetSamples(t *testing.T) {
	w := NewWaveform()
	samples := []float64{0.0, 0.5, 1.0, 0.8}

	w.SetSamples(samples)

	if len(w.samples) != len(samples) {
		t.Errorf("SetSamples() length = %d, expected %d", len(w.samples), len(samples))
	}
	if w.playhead != 0 {
		t.Error("SetSamples() should reset playhead to 0")
	}
}

func TestWaveform_SetProgress(t *testing.T) {
	w := NewWaveform()
	w.SetSamples([]float64{0.0, 0.5, 1.0, 0.8})

	w.SetProgress(0.5)
	if w.playhead != 2 {
		t.Errorf("SetProgress(0.5) playhead = %d, expected 2", w.playhead)
	}

	w.SetProgress(1.0)
	if w.playhead != 4 {
		t.Errorf("SetProgress(1.0) playhead = %d, expected 4", w.playhead)
	}

	w.SetProgress(-0.5)
	if w.playhead != 0 {
		t.Errorf("SetProgress(-0.5) should clamp to 0, got %d", w.playhead)
	}

	w.SetProgress(2.0)
	if w.playhead != 4 {
		t.Errorf("SetProgress(2.0) should clamp to max, got %d", w.playhead)
	}
}

func TestWaveform_RenderBasic(t *testing.T) {
	w := NewWaveform()
	w.SetSamples([]float64{0.0, 1.0, 0.0, 1.0})

	top := w.RenderTop()
	bot := w.RenderBot()

	if top == "" {
		t.Error("RenderTop() returned empty string")
	}
	if bot == "" {
		t.Error("RenderBot() returned empty string")
	}
}

func TestWaveform_RenderEmpty(t *testing.T) {
	w := NewWaveform()

	top := w.RenderTop()
	bot := w.RenderBot()

	if top != "" {
		t.Errorf("RenderTop() with no samples = %q, expected empty", top)
	}
	if bot != "" {
		t.Errorf("RenderBot() with no samples = %q, expected empty", bot)
	}
}

func TestWaveform_AddSample(t *testing.T) {
	w := NewWaveform()

	w.AddSample(0.5)
	if len(w.samples) != 1 {
		t.Errorf("AddSample() length = %d, expected 1", len(w.samples))
	}

	w.AddSample(0.8)
	if len(w.samples) != 2 {
		t.Errorf("AddSample() length = %d, expected 2", len(w.samples))
	}
}
