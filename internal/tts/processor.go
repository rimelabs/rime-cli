package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/audio/analyze"
	"github.com/rimelabs/rime-cli/internal/audio/detectformat"
	"github.com/rimelabs/rime-cli/internal/audio/metadata"
	"github.com/rimelabs/rime-cli/internal/audio/playback"
	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/output/formatters"
	"github.com/rimelabs/rime-cli/internal/output/styles"
)

const (
	defaultSampleRate    = 24000
	defaultNumChannels   = 1
	defaultBitsPerSample = 16
)

type Result struct {
	TTFBMs     int64  `json:"ttfb_ms"`
	DurationMs int64  `json:"duration_ms"`
	SizeBytes  int    `json:"size_bytes"`
	OutputFile string `json:"output_file,omitempty"`
	Text       string `json:"text"`
	Speaker    string `json:"speaker"`
	ModelID    string `json:"model_id"`
	Lang       string `json:"lang"`
}

type RunOptions struct {
	Text       string
	TTSOptions *api.TTSOptions
	Output     string
	Play       bool
	Quiet      bool
	JSON       bool
	Version    string
	BaseURL    string
	ConfigEnv  string
	ConfigFile string
}

func RunNonInteractive(opts RunOptions) error {
	resolved, err := config.ResolveConfigWithOptions(config.ResolveOptions{
		EnvName:        opts.ConfigEnv,
		APIURLOverride: opts.BaseURL,
		ConfigFile:     opts.ConfigFile,
	})
	if err != nil {
		return err
	}

	client := api.NewClientWithOptions(api.ClientOptions{
		APIKey:           resolved.APIKey,
		APIURL:           resolved.APIURL,
		AuthHeaderPrefix: resolved.AuthHeaderPrefix,
		Version:          opts.Version,
	})
	result, err := client.TTSStream(opts.Text, opts.TTSOptions)
	if err != nil {
		return err
	}
	defer result.Body.Close()

	var audioBuf bytes.Buffer
	_, err = io.Copy(&audioBuf, result.Body)
	if err != nil {
		return err
	}

	contentType := result.ContentType
	if contentType == "" {
		contentType = detectformat.DetectFormat(audioBuf.Bytes())
	}
	if contentType == "" {
		contentType = "audio/wav"
	}

	var audioData []byte
	if contentType == "audio/wav" {
		audioData = metadata.FixWavHeader(audioBuf.Bytes())
	} else {
		audioData = audioBuf.Bytes()
	}

	if opts.Output != "" && opts.Output != "-" {
		spk, modelId, lang := api.EffectiveOpts(opts.TTSOptions)
		truncatedText := formatters.TruncateText(opts.Text, 50)

		if contentType == "audio/mpeg" || contentType == "audio/mp3" {
			meta := metadata.MP3Metadata{
				Artist:  "Rime AI TTS",
				Title:   fmt.Sprintf("Rime AI TTS [%s-%s-%s]: %s", spk, modelId, lang, truncatedText),
				Comment: fmt.Sprintf("[%s-%s-%s]: %s", spk, modelId, lang, opts.Text),
			}
			var embedErr error
			audioData, embedErr = metadata.EmbedMP3Metadata(audioData, meta)
			if embedErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to embed MP3 metadata: %v\n", embedErr)
			}
		} else {
			meta := metadata.WavMetadata{
				Artist:  "Rime AI TTS",
				Name:    fmt.Sprintf("Rime AI TTS [%s-%s-%s]: %s", spk, modelId, lang, truncatedText),
				Comment: fmt.Sprintf("[%s-%s-%s]: %s", spk, modelId, lang, opts.Text),
			}
			audioData = metadata.EmbedMetadata(audioData, meta)
		}

		if err := os.WriteFile(opts.Output, audioData, 0644); err != nil {
			return err
		}
		if !opts.Quiet && !opts.JSON {
			fmt.Fprintln(os.Stderr, styles.Successf("Audio saved to %s", opts.Output))
		}
	}

	if opts.Play && !opts.JSON {
		if err := playback.PlayAudioData(audioData, contentType); err != nil {
			return err
		}
	}

	if opts.JSON {
		audioDur := calculateDuration(audioData, contentType)
		spk, modelId, lang := api.EffectiveOpts(opts.TTSOptions)
		ttsResult := Result{
			TTFBMs:     result.TTFB.Milliseconds(),
			DurationMs: audioDur.Milliseconds(),
			SizeBytes:  len(audioData),
			OutputFile: opts.Output,
			Text:       opts.Text,
			Speaker:    spk,
			ModelID:    modelId,
			Lang:       lang,
		}
		return json.NewEncoder(os.Stdout).Encode(ttsResult)
	}

	if !opts.Quiet {
		audioDur := calculateDuration(audioData, contentType)
		stats := fmt.Sprintf("TTFB: %dms | Duration: %s | Size: %s",
			result.TTFB.Milliseconds(),
			formatters.FormatDuration(audioDur),
			formatters.FormatBytes(len(audioData)))
		fmt.Fprintln(os.Stderr, styles.Dim(stats))
	}

	return nil
}

func calculateDuration(audioData []byte, contentType string) time.Duration {
	if contentType == "audio/mpeg" || contentType == "audio/mp3" {
		return analyze.CalculateMP3DurationFromData(audioData)
	}
	return analyze.CalculateDuration(audioData, defaultSampleRate, defaultNumChannels, defaultBitsPerSample)
}
