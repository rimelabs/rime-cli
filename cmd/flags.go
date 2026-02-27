package cmd

import (
	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/spf13/pflag"
)

type modelParamFlags struct {
	// Arcana/ArcanaV2 only
	Temperature       float64
	TopP              float64
	RepetitionPenalty float64
	MaxTokens         int

	// Both families
	SamplingRate int
	SpeedAlpha   float64

	// Mist/MistV2 only
	PauseBetweenBrackets     bool
	PhonemizeBetweenBrackets bool
	InlineSpeedAlpha         string
	NoTextNormalization      bool
	SaveOovs                 bool
}

func (f *modelParamFlags) register(flags *pflag.FlagSet) {
	// Arcana/ArcanaV2 params
	flags.Float64Var(&f.Temperature, "temperature", 0.5, "Sampling temperature (arcana/arcanav2 only, 0–1)")
	flags.Float64Var(&f.TopP, "top-p", 1.0, "Top-p nucleus sampling (arcana/arcanav2 only, 0–1)")
	flags.Float64Var(&f.RepetitionPenalty, "repetition-penalty", 1.5, "Repetition penalty (arcana/arcanav2 only, 1–2)")
	flags.IntVar(&f.MaxTokens, "max-tokens", 1200, "Max output tokens (arcana/arcanav2 only, 200–5000)")

	// Shared params
	flags.IntVar(&f.SamplingRate, "sampling-rate", 0, "Output sampling rate in Hz (arcana: 8000/16000/22050/24000/44100/48000/96000; mist: 4000–44100)")
	flags.Float64Var(&f.SpeedAlpha, "speed-alpha", 1.0, "Speed multiplier, must be >0 (both model families)")

	// Mist/MistV2 params
	flags.BoolVar(&f.PauseBetweenBrackets, "pause-between-brackets", false, "Insert pause at bracketed markers (mist/mistv2 only)")
	flags.BoolVar(&f.PhonemizeBetweenBrackets, "phonemize-between-brackets", false, "Phonemize text in brackets (mist/mistv2 only)")
	flags.StringVar(&f.InlineSpeedAlpha, "inline-speed-alpha", "", "Comma-separated per-segment speed values (mist/mistv2 only)")
	flags.BoolVar(&f.NoTextNormalization, "no-text-normalization", false, "Disable text normalization (mist/mistv2 only)")
	flags.BoolVar(&f.SaveOovs, "save-oovs", false, "Save out-of-vocabulary words (mist/mistv2 only)")
}

func (f *modelParamFlags) applyChanged(flags *pflag.FlagSet, opts *api.TTSOptions) {
	if flags.Changed("temperature") {
		opts.Temperature = &f.Temperature
	}
	if flags.Changed("top-p") {
		opts.TopP = &f.TopP
	}
	if flags.Changed("repetition-penalty") {
		opts.RepetitionPenalty = &f.RepetitionPenalty
	}
	if flags.Changed("max-tokens") {
		opts.MaxTokens = &f.MaxTokens
	}
	if flags.Changed("sampling-rate") {
		opts.SamplingRate = &f.SamplingRate
	}
	if flags.Changed("speed-alpha") {
		opts.SpeedAlpha = &f.SpeedAlpha
	}
	if flags.Changed("pause-between-brackets") {
		opts.PauseBetweenBrackets = &f.PauseBetweenBrackets
	}
	if flags.Changed("phonemize-between-brackets") {
		opts.PhonemizeBetweenBrackets = &f.PhonemizeBetweenBrackets
	}
	if flags.Changed("inline-speed-alpha") {
		opts.InlineSpeedAlpha = &f.InlineSpeedAlpha
	}
	if flags.Changed("no-text-normalization") {
		opts.NoTextNormalization = &f.NoTextNormalization
	}
	if flags.Changed("save-oovs") {
		opts.SaveOovs = &f.SaveOovs
	}
}
