package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/audio/metadata"
	"github.com/rimelabs/rime-cli/internal/audio/playback"
	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/output/styles"
	"github.com/rimelabs/rime-cli/internal/output/ui"
	"github.com/rimelabs/rime-cli/internal/tts"
)

func NewTTSCmd() *cobra.Command {
	var output string
	var play bool
	var spk string
	var modelId string
	var lang string
	var format string
	var apiURL string
	var modelParams modelParamFlags

	cmd := &cobra.Command{
		Use:   "tts TEXT",
		Short: "Synthesize text to speech",
		Long: `Convert text to speech audio.

Supports both WAV and MP3 formats. The format is automatically selected based on the model:
- mistv2 model outputs MP3 format
- All other models output WAV format

Use --format to override the default format selection.

The CLI handles format detection, metadata embedding, and playback for both formats.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := args[0]

			if !playback.IsPlaybackEnabled() {
				if output == "" {
					return fmt.Errorf("output file required in headless build (use -o FILE or -o - for stdout)")
				}
			}

			shouldPlay := playback.IsPlaybackEnabled() && (play || output == "")

			if spk == "" {
				return fmt.Errorf("--speaker is required. You can use --speaker astra")
			}
			if modelId == "" {
				return fmt.Errorf("--model-id is required. You can use --model-id %s", api.ModelIDArcana)
			}
			if !api.IsValidModelID(modelId) {
				return fmt.Errorf("invalid modelId: %s (valid options: %s, %s, %s, %s)", modelId, api.ModelIDArcana, api.ModelIDArcanaV2, api.ModelIDMistV2, api.ModelIDMist)
			}
			if !api.IsValidLang(lang, modelId) {
				return fmt.Errorf("invalid language %q for model %s (valid: %s)", lang, modelId, strings.Join(api.ValidLangsForModel(modelId), ", "))
			}

			modelIdLower := strings.ToLower(modelId)
			if (modelIdLower == api.ModelIDMist || modelIdLower == api.ModelIDMistV2) && format != "mp3" {
				return fmt.Errorf("%s and %s models require --format mp3. Please specify --format mp3", api.ModelIDMist, api.ModelIDMistV2)
			}

			var audioFormat string
			if format != "" {
				switch format {
				case "wav":
					audioFormat = "audio/wav"
				case "mp3":
					audioFormat = "audio/mp3"
				default:
					return fmt.Errorf("unsupported format: %s (supported: wav, mp3)", format)
				}
			}

			opts := &api.TTSOptions{
				Speaker:     spk,
				ModelID:     modelId,
				Lang:        lang,
				AudioFormat: audioFormat,
			}
			modelParams.applyChanged(cmd.Flags(), opts)
			if err := api.ValidateModelParams(opts); err != nil {
				return err
			}

			if output == "-" {
				resolved, err := config.ResolveConfigWithOptions(config.ResolveOptions{
					EnvName:        ConfigEnv,
					APIURLOverride: apiURL,
					ConfigFile:     ConfigFile,
				})
				if err != nil {
					return err
				}
				client := api.NewClient(api.ClientOptions{
					APIKey:           resolved.APIKey,
					APIURL:           resolved.APIURL,
					AuthHeaderPrefix: resolved.AuthHeaderPrefix,
					Version:          Version,
				})
				audioData, err := client.TTS(text, opts)
				if err != nil {
					return err
				}
				contentType := opts.AudioFormat
				if contentType == "" {
					contentType = api.GetAudioFormat(opts.ModelID)
				}
				if contentType == "audio/wav" {
					audioData = metadata.FixWavHeader(audioData)
				}
				_, err = os.Stdout.Write(audioData)
				return err
			}

			if Quiet || JSONOutput || !term.IsTerminal(int(os.Stdout.Fd())) {
				runOpts := tts.RunOptions{
					Text:       text,
					TTSOptions: opts,
					Output:     output,
					Play:       shouldPlay,
					Quiet:      Quiet,
					JSON:       JSONOutput,
					Version:    Version,
					BaseURL:    apiURL,
					ConfigEnv:  ConfigEnv,
					ConfigFile: ConfigFile,
				}
				return tts.RunNonInteractive(runOpts)
			}

			if output == "" && shouldPlay {
				fmt.Fprintln(os.Stderr, styles.Dim("Playing audio (use -o to save)"))
			}

			p := tea.NewProgram(ui.NewTTSModel(text, opts, output, shouldPlay, Version, apiURL, ConfigEnv, ConfigFile))
			m, err := p.Run()
			if err != nil {
				return err
			}

			ttsM := m.(ui.TTSModel)
			if ttsM.Err() != nil {
				return ttsM.Err()
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (or - for stdout)")
	if playback.IsPlaybackEnabled() {
		cmd.Flags().BoolVarP(&play, "play", "p", false, "Play audio (default if no output specified)")
	}
	cmd.Flags().StringVarP(&spk, "speaker", "s", "", "Voice speaker to use (required)")
	cmd.Flags().StringVarP(&modelId, "model-id", "m", "", fmt.Sprintf("Model ID (required, e.g., %s, %s, %s, %s)", api.ModelIDArcana, api.ModelIDArcanaV2, api.ModelIDMistV2, api.ModelIDMist))
	cmd.Flags().StringVar(&modelId, "modelId", "", "")
	cmd.Flags().MarkHidden("modelId")
	cmd.Flags().StringVarP(&lang, "lang", "l", "eng", "Language code (e.g., eng, es, fra). Valid codes depend on model.")
	cmd.Flags().StringVarP(&format, "format", "f", "", "Audio format: wav or mp3 (overrides model default)")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "API URL (default: $RIME_API_URL or https://users.rime.ai/v1/rime-tts)")

	modelParams.register(cmd.Flags())

	return cmd
}
