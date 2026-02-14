package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gopxl/beep/v2"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/audio/analyze"
	"github.com/rimelabs/rime-cli/internal/audio/decode"
	"github.com/rimelabs/rime-cli/internal/audio/detectformat"
	"github.com/rimelabs/rime-cli/internal/audio/metadata"
	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/output/formatters"
	"github.com/rimelabs/rime-cli/internal/output/styles"
	"github.com/rimelabs/rime-cli/internal/output/visualizer"
)

type TTSState int

const (
	TTSStateConnecting TTSState = iota
	TTSStatePlaying
	TTSStateDone
)

type TTSModel struct {
	text       string
	opts       *api.TTSOptions
	output     string
	shouldPlay bool
	version    string
	baseURL    string

	state       TTSState
	err         error
	ttfb        time.Duration
	audioBuf    *bytes.Buffer
	contentType string

	analyzer    *analyze.AmplitudeAnalyzer
	sampleRate  beep.SampleRate
	numChannels int
	precision   int
	playDone    chan struct{}
	playStart   time.Time
	audioDur    time.Duration

	waveform   *visualizer.Waveform
	transcript *visualizer.Transcript
	frame      int
	termWidth  int
}

type StreamStartedMsg struct {
	Analyzer    *analyze.AmplitudeAnalyzer
	SampleRate  beep.SampleRate
	NumChannels int
	Precision   int
	PlayDone    chan struct{}
	AudioBuf    *bytes.Buffer
	TTFB        time.Duration
	ContentType string
	Err         error
}

type TTSTickMsg time.Time
type TTSQuitMsg struct{}

func NewTTSModel(text string, opts *api.TTSOptions, output string, shouldPlay bool, version string, baseURL ...string) TTSModel {
	predictedDuration := visualizer.EstimateDurationFromText(text)
	url := ""
	if len(baseURL) > 0 {
		url = baseURL[0]
	}
	return TTSModel{
		text:       text,
		opts:       opts,
		output:     output,
		shouldPlay: shouldPlay,
		version:    version,
		baseURL:    url,
		state:      TTSStateConnecting,
		waveform:   visualizer.NewWaveform(),
		transcript: visualizer.NewTranscript(text, predictedDuration),
		termWidth:  GetTerminalWidth(),
	}
}

func (m TTSModel) Init() tea.Cmd {
	return tea.Batch(
		m.startStreaming(),
		ttsTick(),
	)
}

func (m TTSModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

	case StreamStartedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, tea.Quit
		}
		m.state = TTSStatePlaying
		m.analyzer = msg.Analyzer
		m.sampleRate = msg.SampleRate
		m.numChannels = msg.NumChannels
		m.precision = msg.Precision
		m.playDone = msg.PlayDone
		m.audioBuf = msg.AudioBuf
		m.ttfb = msg.TTFB
		m.contentType = msg.ContentType
		m.playStart = time.Now()
		return m, ttsTick()

	case TTSTickMsg:
		m.frame++

		if m.state == TTSStatePlaying {
			if m.analyzer != nil {
				amp := m.analyzer.Amplitude()
				scaled := amp * 5.0
				if amp > 0.01 && scaled < 0.2 {
					scaled = 0.2
				}
				if scaled > 1.0 {
					scaled = 1.0
				}
				m.waveform.AddSample(scaled)
			}
			if m.transcript != nil {
				elapsed := time.Since(m.playStart)
				m.transcript.SetElapsed(elapsed)
			}
		}

		select {
		case <-m.playDone:
			m.state = TTSStateDone
			if m.audioBuf != nil && m.sampleRate > 0 {
				contentType := m.contentType
				if contentType == "" {
					contentType = detectformat.DetectFormat(m.audioBuf.Bytes())
				}
				if contentType == "" {
					contentType = "audio/wav"
				}

				if contentType == "audio/wav" {
					m.audioDur = analyze.CalculateDuration(m.audioBuf.Bytes(), int(m.sampleRate), m.numChannels, m.precision*8)
				} else if contentType == "audio/mpeg" || contentType == "audio/mp3" {
					m.audioDur = analyze.CalculateMP3DurationFromData(m.audioBuf.Bytes())
				} else {
					m.audioDur = time.Duration(float64(m.audioBuf.Len()) / 16000.0 * float64(time.Second))
				}

				if m.transcript != nil {
					m.transcript.SetDuration(m.audioDur)
					m.transcript.SetElapsed(m.audioDur)
				}

				var audioData []byte
				if contentType == "audio/wav" {
					audioData = metadata.FixWavHeader(m.audioBuf.Bytes())
				} else {
					audioData = m.audioBuf.Bytes()
				}

				samplesPerSecond := 20
				if m.audioDur > 0 {
					targetSamples := m.termWidth * 2
					calculatedSamplesPerSecond := float64(targetSamples) / m.audioDur.Seconds()
					if calculatedSamplesPerSecond > 0 {
						samplesPerSecond = int(calculatedSamplesPerSecond)
						if samplesPerSecond < 1 {
							samplesPerSecond = 1
						}
					}
				}

				amps, err := analyze.AnalyzeAmplitudes(audioData, samplesPerSecond)
				if err == nil && m.waveform != nil {
					scaled := analyze.ScaleAmplitudes(amps, 5.0, 0.2)
					m.waveform.SetSamples(scaled)
					m.waveform.SetProgress(1.0)
				}
			}
			return m, m.finalize()
		default:
		}
		return m, ttsTick()

	case TTSQuitMsg:
		return m, tea.Quit
	}

	return m, nil
}

func (m TTSModel) View() string {
	if m.err != nil {
		return ""
	}

	var b strings.Builder

	spk, modelId, lang := api.EffectiveOpts(m.opts)

	switch m.state {
	case TTSStateConnecting:
		b.WriteString(Spinner[m.frame%len(Spinner)])
		b.WriteString(" Connecting...")

	case TTSStatePlaying:
		separator := RenderSeparator(m.termWidth)

		b.WriteString(separator)
		b.WriteString("\n")
		b.WriteString(HeaderStyle.Render(fmt.Sprintf("Rime TTS: %s (%s) %s", spk, modelId, lang)))
		b.WriteString("\n")

		if m.transcript != nil {
			b.WriteString(m.transcript.Render())
			b.WriteString("\n")
		}

		elapsed := time.Since(m.playStart)
		if m.audioDur > 0 {
			b.WriteString(DimStyle.Render(fmt.Sprintf("[%s / %s]", formatters.FormatDuration(elapsed), formatters.FormatDuration(m.audioDur))))
		} else {
			b.WriteString(DimStyle.Render(fmt.Sprintf("[%s]", formatters.FormatDuration(elapsed))))
		}
		b.WriteString("\n")
		b.WriteString(m.waveform.RenderTop())
		b.WriteString("\n")
		b.WriteString(m.waveform.RenderBot())
		b.WriteString("\n")
		b.WriteString("\n")
		b.WriteString(separator)
		b.WriteString("\n")

	case TTSStateDone:
		separator := RenderSeparator(m.termWidth)

		b.WriteString(separator)
		b.WriteString("\n")
		b.WriteString(HeaderStyle.Render(fmt.Sprintf("Rime TTS: %s (%s) %s", spk, modelId, lang)))
		b.WriteString("\n")

		if m.transcript != nil {
			b.WriteString(m.transcript.Render())
			b.WriteString("\n")
		}

		if m.audioDur > 0 {
			b.WriteString(DimStyle.Render(fmt.Sprintf("[%s / %s]", formatters.FormatDuration(m.audioDur), formatters.FormatDuration(m.audioDur))))
		} else {
			elapsed := time.Since(m.playStart)
			b.WriteString(DimStyle.Render(fmt.Sprintf("[%s]", formatters.FormatDuration(elapsed))))
		}
		b.WriteString("\n")
		if m.waveform != nil {
			b.WriteString(m.waveform.RenderTop())
			b.WriteString("\n")
			b.WriteString(m.waveform.RenderBot())
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(separator)
		b.WriteString("\n")

		if m.output != "" && m.output != "-" {
			b.WriteString(styles.Successf("Audio saved to %s", m.output))
			b.WriteString("\n")
		}
		size := 0
		if m.audioBuf != nil {
			size = m.audioBuf.Len()
		}
		stats := fmt.Sprintf("TTFB: %dms | Duration: %s | Size: %s",
			m.ttfb.Milliseconds(),
			formatters.FormatDuration(m.audioDur),
			formatters.FormatBytes(size))
		b.WriteString(DimStyle.Render(stats))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *TTSModel) startStreaming() tea.Cmd {
	text := m.text
	opts := m.opts
	shouldPlay := m.shouldPlay
	version := m.version
	baseURL := m.baseURL
	return func() tea.Msg {
		apiKey, err := config.LoadAPIKey()
		if err != nil {
			return StreamStartedMsg{Err: err}
		}

		var client *api.Client
		if baseURL != "" {
			client = api.NewClient(apiKey, version, baseURL)
		} else {
			client = api.NewClient(apiKey, version)
		}
		result, err := client.TTSStream(text, opts)
		if err != nil {
			return StreamStartedMsg{Err: err}
		}

		contentType := result.ContentType
		if contentType == "" {
			peekBuf := make([]byte, 512)
			n, _ := result.Body.Read(peekBuf)
			contentType = detectformat.DetectFormat(peekBuf[:n])
			result.Body = io.NopCloser(io.MultiReader(bytes.NewReader(peekBuf[:n]), result.Body))
		}
		if contentType == "" {
			contentType = "audio/wav"
		}

		var audioBuf bytes.Buffer
		tee := io.TeeReader(result.Body, &audioBuf)

		decoder, format, err := decode.DecodeAudio(tee, contentType)
		if err != nil {
			result.Body.Close()
			return StreamStartedMsg{Err: err}
		}

		analyzer := analyze.NewAmplitudeAnalyzer(decoder)

		playDone := make(chan struct{})

		if shouldPlay {
			err = m.startPlayback(format, analyzer, result.Body, playDone)
			if err != nil {
				return StreamStartedMsg{Err: err}
			}
		} else {
			go func() {
				io.Copy(io.Discard, tee)
				result.Body.Close()
				close(playDone)
			}()
		}

		return StreamStartedMsg{
			Analyzer:    analyzer,
			SampleRate:  format.SampleRate,
			NumChannels: format.NumChannels,
			Precision:   format.Precision,
			PlayDone:    playDone,
			AudioBuf:    &audioBuf,
			TTFB:        result.TTFB,
			ContentType: contentType,
		}
	}
}

func (m *TTSModel) finalize() tea.Cmd {
	output := m.output
	text := m.text
	opts := m.opts
	audioBuf := m.audioBuf
	contentType := m.contentType
	return func() tea.Msg {
		if output != "" && output != "-" && audioBuf != nil {
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

			spk, modelId, lang := api.EffectiveOpts(opts)
			truncatedText := formatters.TruncateText(text, 50)

			if contentType == "audio/mpeg" || contentType == "audio/mp3" {
				meta := metadata.MP3Metadata{
					Artist:  "Rime AI TTS",
					Title:   fmt.Sprintf("Rime AI TTS [%s-%s-%s]: %s", spk, modelId, lang, truncatedText),
					Comment: fmt.Sprintf("[%s-%s-%s]: %s", spk, modelId, lang, text),
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
					Comment: fmt.Sprintf("[%s-%s-%s]: %s", spk, modelId, lang, text),
				}
				audioData = metadata.EmbedMetadata(audioData, meta)
			}

			os.WriteFile(output, audioData, 0644)
		}

		time.Sleep(500 * time.Millisecond)
		return TTSQuitMsg{}
	}
}

func ttsTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return TTSTickMsg(t)
	})
}

func (m TTSModel) Err() error {
	return m.err
}
