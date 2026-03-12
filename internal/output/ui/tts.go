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
	configEnv  string
	configFile string

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

	waveform          *visualizer.Waveform
	transcript        *visualizer.Transcript
	frame             int
	termWidth         int
	rightContentWidth int
	minimal           bool
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

func NewTTSModel(text string, opts *api.TTSOptions, output string, shouldPlay bool, version string, baseURL string, configEnv string, configFile string, minimal bool) TTSModel {
	predictedDuration := visualizer.EstimateDurationFromText(text)
	var termWidth int
	var rightContentWidth int
	var waveform *visualizer.Waveform

	if minimal {
		termWidth = GetTerminalWidth(40, 0)
		waveform = visualizer.NewWaveform(termWidth)
	} else {
		termWidth = GetTerminalWidth(40, 80)
		rightContentWidth = termWidth - BoxOverhead
		waveform = visualizer.NewWaveformTwoRow(rightContentWidth)
	}

	return TTSModel{
		text:              text,
		opts:              opts,
		output:            output,
		shouldPlay:        shouldPlay,
		version:           version,
		baseURL:           baseURL,
		configEnv:         configEnv,
		configFile:        configFile,
		state:             TTSStateConnecting,
		waveform:          waveform,
		transcript:        visualizer.NewTranscript(text, predictedDuration),
		termWidth:         termWidth,
		rightContentWidth: rightContentWidth,
		minimal:           minimal,
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
				m.waveform.AddSample(m.analyzer.Amplitude())
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
					targetSamples := m.waveform.Width()
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
					m.waveform.SetSamples(amps)
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
		b.WriteString(Spinner[m.frame%len(Spinner)] + " Connecting...\n")

	case TTSStatePlaying, TTSStateDone:
		stats := m.buildStats()
		statsLine := strings.Join(stats, DimStyle.Render(" | "))
		if m.minimal {
			labels := [][2]string{
				{"model", modelId},
				{"speaker", spk},
				{"lang", lang},
			}
			b.WriteString(RenderMinimalView("Rime TTS", m.waveform, m.transcript, m.text, m.termWidth, labels, statsLine))
		} else {
			header := RenderLabeledHeader(spk, modelId, lang)
			var elapsedStr string
			if m.state == TTSStateDone && m.audioDur > 0 {
				elapsedStr = formatters.FormatDuration(m.audioDur)
			} else if !m.playStart.IsZero() {
				elapsedStr = formatters.FormatDuration(time.Since(m.playStart))
			}
			rightLines := RenderRightPanel(header, m.rightContentWidth, m.transcript, elapsedStr, "", m.waveform)
			b.WriteString(RenderBoxLayout(m.rightContentWidth, rightLines))
			if len(stats) > 0 {
				b.WriteString(statsLine + "\n")
			}
		}

		if m.state == TTSStateDone && m.output != "" && m.output != "-" {
			b.WriteString(styles.Successf("Audio saved to %s", m.output) + "\n")
		}
	}

	return b.String()
}

func (m *TTSModel) startStreaming() tea.Cmd {
	text := m.text
	opts := m.opts
	shouldPlay := m.shouldPlay
	version := m.version
	baseURL := m.baseURL
	configEnv := m.configEnv
	configFile := m.configFile
	return func() tea.Msg {
		resolved, err := config.ResolveConfigWithOptions(config.ResolveOptions{
			EnvName:        configEnv,
			APIURLOverride: baseURL,
			ConfigFile:     configFile,
		})
		if err != nil {
			return StreamStartedMsg{Err: err}
		}

		client := api.NewClient(api.ClientOptions{
			APIKey:           resolved.APIKey,
			APIURL:           resolved.APIURL,
			AuthHeaderPrefix: resolved.AuthHeaderPrefix,
			Version:          version,
		})
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

func (m TTSModel) buildStats() []string {
	var stats []string
	if m.ttfb > 0 {
		stats = append(stats, DimStyle.Render("TTFB: ")+fmt.Sprintf("%dms", m.ttfb.Milliseconds()))
	}
	var dur time.Duration
	if m.state == TTSStateDone && m.audioDur > 0 {
		dur = m.audioDur
	} else if !m.playStart.IsZero() {
		dur = time.Since(m.playStart)
	}
	if dur > 0 {
		stats = append(stats, DimStyle.Render("Duration: ")+formatters.FormatDuration(dur))
	}
	if m.audioBuf != nil && m.audioBuf.Len() > 0 {
		stats = append(stats, DimStyle.Render("Size: ")+formatters.FormatBytes(m.audioBuf.Len()))
	}
	return stats
}

func ttsTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return TTSTickMsg(t)
	})
}

func (m TTSModel) Err() error {
	return m.err
}
