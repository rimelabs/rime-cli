//go:build !headless

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
	"github.com/gopxl/beep/v2/speaker"

	"github.com/rimelabs/rime-cli/internal/audio/analyze"
	"github.com/rimelabs/rime-cli/internal/audio/detectformat"
	"github.com/rimelabs/rime-cli/internal/audio/metadata"
	"github.com/rimelabs/rime-cli/internal/audio/stream"
	"github.com/rimelabs/rime-cli/internal/output/formatters"
	"github.com/rimelabs/rime-cli/internal/output/visualizer"
)

type PlayState int

const (
	PlayStateLoading PlayState = iota
	PlayStatePlaying
	PlayStateDone
)

type PlayModel struct {
	filepath string
	meta     metadata.WavMetadata
	mp3Meta  metadata.MP3Metadata
	isMP3    bool

	state     PlayState
	err       error
	audioData []byte

	streamer   beep.StreamSeekCloser
	sampleRate beep.SampleRate
	playDone   chan struct{}
	playStart  time.Time
	audioDur   time.Duration

	waveform   *visualizer.Waveform
	transcript *visualizer.Transcript
	frame      int

	parsedComment *metadata.ParsedComment
	hasComment    bool
}

type PlayLoadDoneMsg struct {
	Audio []byte
	Meta  metadata.WavMetadata
	Err   error
}

type PlayStartedMsg struct {
	Streamer   beep.StreamSeekCloser
	SampleRate beep.SampleRate
	PlayDone   chan struct{}
	AudioDur   time.Duration
}

type PlayTickMsg time.Time
type PlayQuitMsg struct{}

func NewPlayModel(filepath string) PlayModel {
	return PlayModel{
		filepath: filepath,
		state:    PlayStateLoading,
		waveform: visualizer.NewWaveform(),
	}
}

func (m PlayModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadFile(),
		playTick(),
	)
}

func (m PlayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			if m.streamer != nil {
				m.streamer.Close()
			}
			return m, tea.Quit
		}

	case PlayLoadDoneMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, tea.Quit
		}
		m.audioData = msg.Audio
		m.meta = msg.Meta

		contentType := detectformat.DetectFormat(m.audioData)
		if contentType == "audio/mpeg" || contentType == "audio/mp3" {
			m.isMP3 = true
			m.mp3Meta = metadata.ReadMP3Metadata(m.audioData)
			parsed, isValid := metadata.ParseComment(m.mp3Meta.Comment)
			m.parsedComment = parsed
			m.hasComment = isValid
			if isValid {
				m.transcript = visualizer.NewTranscript(parsed.Text, 0)
			}
		} else {
			m.isMP3 = false
			parsed, isValid := metadata.ParseComment(m.meta.Comment)
			m.parsedComment = parsed
			m.hasComment = isValid
			if isValid {
				m.transcript = visualizer.NewTranscript(parsed.Text, 0)
			}
		}

		var audioDur time.Duration
		if m.isMP3 {
			audioDur = analyze.CalculateMP3DurationFromData(m.audioData)
		} else {
			audioDur = analyze.CalculateWavDuration(m.audioData)
		}

		termWidth := GetTerminalWidth()
		samplesPerSecond := 20
		if audioDur > 0 {
			targetSamples := termWidth * 2
			calculatedSamplesPerSecond := float64(targetSamples) / audioDur.Seconds()
			if calculatedSamplesPerSecond > 0 {
				samplesPerSecond = int(calculatedSamplesPerSecond)
				if samplesPerSecond < 1 {
					samplesPerSecond = 1
				}
			}
		}

		amps, err := analyze.AnalyzeAmplitudes(m.audioData, samplesPerSecond)
		if err == nil {
			scaled := analyze.ScaleAmplitudes(amps, 5.0, 0.2)
			m.waveform.SetSamples(scaled)
		}
		return m, m.startPlayback()

	case PlayStartedMsg:
		m.state = PlayStatePlaying
		m.streamer = msg.Streamer
		m.sampleRate = msg.SampleRate
		m.playDone = msg.PlayDone
		m.audioDur = msg.AudioDur
		m.playStart = time.Now()
		m.frame = 0
		if m.hasComment && m.transcript == nil {
			m.transcript = visualizer.NewTranscript(m.parsedComment.Text, m.audioDur)
		} else if m.transcript != nil {
			m.transcript.SetDuration(m.audioDur)
		}
		return m, playTick()

	case PlayTickMsg:
		m.frame++
		if m.state == PlayStatePlaying {
			elapsed := time.Since(m.playStart)
			if m.audioDur > 0 {
				progress := float64(elapsed) / float64(m.audioDur)
				m.waveform.SetProgress(progress)
			}
			if m.transcript != nil {
				m.transcript.SetElapsed(elapsed)
			}
		}

		select {
		case <-m.playDone:
			if m.streamer != nil {
				m.streamer.Close()
			}
			m.state = PlayStateDone
			if m.waveform != nil {
				m.waveform.SetProgress(1.0)
			}
			return m, playDelayedQuit()
		default:
		}
		return m, playTick()

	case PlayQuitMsg:
		return m, tea.Quit
	}

	return m, nil
}

func (m PlayModel) View() string {
	if m.err != nil {
		return ""
	}

	var b strings.Builder

	switch m.state {
	case PlayStateLoading:
		b.WriteString(Spinner[m.frame%len(Spinner)])
		b.WriteString(" Loading...")

	case PlayStatePlaying:
		width := GetTerminalWidth()
		separator := RenderSeparator(width)

		b.WriteString(separator)
		b.WriteString("\n")

		if m.hasComment {
			header := fmt.Sprintf("Rime TTS [%s-%s-%s]", m.parsedComment.Speaker, m.parsedComment.ModelID, m.parsedComment.Language)
			b.WriteString(HeaderStyle.Render(header))
			b.WriteString("\n")
		} else {
			b.WriteString(HeaderStyle.Render(m.filepath))
			b.WriteString("\n")
		}

		if m.transcript != nil {
			b.WriteString(m.transcript.Render())
			b.WriteString("\n")
		}

		elapsed := time.Since(m.playStart)
		b.WriteString(DimStyle.Render(fmt.Sprintf("[%s / %s]", formatters.FormatDuration(elapsed), formatters.FormatDuration(m.audioDur))))
		b.WriteString("\n")

		b.WriteString(m.waveform.RenderTop())
		b.WriteString("\n")
		b.WriteString(m.waveform.RenderBot())
		b.WriteString("\n")
		b.WriteString("\n")
		b.WriteString(separator)
		b.WriteString("\n")

	case PlayStateDone:
		width := GetTerminalWidth()
		separator := RenderSeparator(width)

		b.WriteString(separator)
		b.WriteString("\n")

		if m.hasComment {
			header := fmt.Sprintf("Rime TTS [%s-%s-%s]", m.parsedComment.Speaker, m.parsedComment.ModelID, m.parsedComment.Language)
			b.WriteString(HeaderStyle.Render(header))
			b.WriteString("\n")
		} else {
			b.WriteString(HeaderStyle.Render(m.filepath))
			b.WriteString("\n")
		}

		if m.transcript != nil {
			b.WriteString(m.transcript.Render())
			b.WriteString("\n")
		}

		b.WriteString(DimStyle.Render(fmt.Sprintf("[%s / %s]", formatters.FormatDuration(m.audioDur), formatters.FormatDuration(m.audioDur))))
		b.WriteString("\n")

		b.WriteString(m.waveform.RenderTop())
		b.WriteString("\n")
		b.WriteString(m.waveform.RenderBot())
		b.WriteString("\n")
		b.WriteString("\n")
		b.WriteString(separator)
		b.WriteString("\n")
	}

	return b.String()
}

func (m *PlayModel) loadFile() tea.Cmd {
	filepath := m.filepath
	return func() tea.Msg {
		data, err := os.ReadFile(filepath)
		if err != nil {
			return PlayLoadDoneMsg{Err: err}
		}

		meta := metadata.ReadMetadata(data)
		return PlayLoadDoneMsg{Audio: data, Meta: meta}
	}
}

func (m *PlayModel) startPlayback() tea.Cmd {
	audioData := m.audioData
	isMP3 := m.isMP3
	return func() tea.Msg {
		reader := bytes.NewReader(audioData)
		var streamer beep.StreamSeekCloser
		var format beep.Format
		var err error
		if isMP3 {
			streamer, format, err = stream.DecodeMP3Streaming(io.NopCloser(reader))
		} else {
			decoder, f, decodeErr := stream.DecodeStreaming(reader)
			if decodeErr != nil {
				return PlayQuitMsg{}
			}
			streamer = &wavStreamerAdapter{decoder: decoder, rc: io.NopCloser(reader)}
			format = f
		}

		if err != nil {
			return PlayQuitMsg{}
		}

		var audioDur time.Duration

		if isMP3 {
			audioDur = analyze.CalculateMP3Duration(streamer, format.SampleRate)
			if audioDur == 0 {
				audioDur = analyze.CalculateMP3DurationFromData(audioData)
			}
		} else {
			if streamer.Len() > 0 {
				audioDur = format.SampleRate.D(streamer.Len())
			} else {
				audioDur = analyze.CalculateWavDuration(audioData)
			}
		}

		err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		if err != nil {
			streamer.Close()
			return PlayQuitMsg{}
		}

		playDone := make(chan struct{})
		speaker.Play(beep.Seq(streamer, beep.Callback(func() {
			close(playDone)
		})))

		return PlayStartedMsg{
			Streamer:   streamer,
			SampleRate: format.SampleRate,
			PlayDone:   playDone,
			AudioDur:   audioDur,
		}
	}
}

type wavStreamerAdapter struct {
	decoder *stream.StreamingDecoder
	rc      io.ReadCloser
}

func (w *wavStreamerAdapter) Stream(samples [][2]float64) (n int, ok bool) {
	return w.decoder.Stream(samples)
}

func (w *wavStreamerAdapter) Err() error {
	return w.decoder.Err()
}

func (w *wavStreamerAdapter) Len() int {
	return -1
}

func (w *wavStreamerAdapter) Position() int {
	return -1
}

func (w *wavStreamerAdapter) Seek(p int) error {
	return fmt.Errorf("seek not supported")
}

func (w *wavStreamerAdapter) Close() error {
	if w.rc != nil {
		return w.rc.Close()
	}
	return nil
}

func playTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return PlayTickMsg(t)
	})
}

func playDelayedQuit() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return PlayQuitMsg{}
	})
}

func (m PlayModel) Err() error {
	return m.err
}
