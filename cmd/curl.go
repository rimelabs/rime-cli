package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/config"
)

type CurlOptions struct {
	Text       string
	Speaker    string
	ModelID    string
	Lang       string
	ShowKey    bool
	Oneline    bool
	APIKey     string
	APIURL     string
	AuthPrefix string
}

func audioFormatToExt(acceptHeader string) string {
	if acceptHeader == "audio/mp3" {
		return "mp3"
	}
	return "wav"
}

func generateCurlCommand(opts CurlOptions, modelOpts *api.TTSOptions) (string, error) {
	reqBody := api.TTSRequest{
		Text:    opts.Text,
		Speaker: opts.Speaker,
		ModelID: opts.ModelID,
		Lang:    opts.Lang,

		RepetitionPenalty:        modelOpts.RepetitionPenalty,
		Temperature:              modelOpts.Temperature,
		TopP:                     modelOpts.TopP,
		MaxTokens:                modelOpts.MaxTokens,
		SamplingRate:             modelOpts.SamplingRate,
		SpeedAlpha:               modelOpts.SpeedAlpha,
		PauseBetweenBrackets:     modelOpts.PauseBetweenBrackets,
		PhonemizeBetweenBrackets: modelOpts.PhonemizeBetweenBrackets,
		InlineSpeedAlpha:         modelOpts.InlineSpeedAlpha,
		NoTextNormalization:      modelOpts.NoTextNormalization,
		SaveOovs:                 modelOpts.SaveOovs,
	}

	var jsonBody []byte
	var err error
	if opts.Oneline {
		jsonBody, err = json.Marshal(reqBody)
	} else {
		jsonBody, err = json.MarshalIndent(reqBody, "", "  ")
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	authHeader := "$RIME_CLI_API_KEY"
	if opts.ShowKey && opts.APIKey != "" {
		authHeader = opts.APIKey
	}

	apiURL := opts.APIURL
	if apiURL == "" {
		apiURL = api.GetAPIURL()
	}

	authPrefix := opts.AuthPrefix
	if authPrefix == "" {
		authPrefix = "Bearer"
	}

	acceptHeader := api.GetAudioFormat(opts.ModelID)
	outputFile := "output." + audioFormatToExt(acceptHeader)

	var b strings.Builder
	jsonStr := strings.ReplaceAll(string(jsonBody), "'", "'\\''")

	if opts.Oneline {
		b.WriteString(fmt.Sprintf("curl -X POST '%s' -H 'Accept: %s' -H 'Authorization: %s %s' -H 'Content-Type: application/json' -o '%s' -f -d '%s'", apiURL, acceptHeader, authPrefix, authHeader, outputFile, jsonStr))
	} else {
		b.WriteString("curl --request POST \\\n")
		b.WriteString(fmt.Sprintf("  --url '%s' \\\n", apiURL))
		b.WriteString(fmt.Sprintf("  --header 'Accept: %s' \\\n", acceptHeader))
		b.WriteString(fmt.Sprintf("  --header 'Authorization: %s %s' \\\n", authPrefix, authHeader))
		b.WriteString("  --header 'Content-Type: application/json' \\\n")
		b.WriteString(fmt.Sprintf("  --output '%s' \\\n", outputFile))
		b.WriteString("  --fail \\\n")
		b.WriteString(fmt.Sprintf("  --data '%s'", jsonStr))
	}

	return b.String(), nil
}

func NewCurlCmd() *cobra.Command {
	var spk string
	var modelId string
	var lang string
	var showKey bool
	var oneline bool
	var apiURL string
	var modelParams modelParamFlags

	cmd := &cobra.Command{
		Use:   "curl TEXT",
		Short: "Generate curl command for TTS request",
		Long: `Generate a curl command for making TTS API requests.

Run without arguments to see an example:
  rime curl

For easy copy-paste (single line):
  rime curl --oneline --show-key

Or provide your own text:
  rime curl "your text here" --speaker astra --model-id arcana`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := ""
			if len(args) > 0 {
				text = args[0]

				if spk == "" {
					return fmt.Errorf("--speaker is required when providing text")
				}

				if modelId == "" {
					return fmt.Errorf("--model-id is required when providing text")
				}
			} else {
				text = "Hello from Rime um lemme know if you can hear me!"

				if spk == "" {
					spk = "astra"
				}

				if modelId == "" {
					modelId = api.ModelIDArcana
				}
			}

			if modelId != "" {
				if !api.IsValidModelID(modelId) {
					return fmt.Errorf("invalid modelId: %s (valid options: %s, %s, %s, %s)", modelId, api.ModelIDArcana, api.ModelIDArcanaV2, api.ModelIDMistV2, api.ModelIDMist)
				}
				if !api.IsValidLang(lang, modelId) {
					return fmt.Errorf("invalid language %q for model %s (valid: %s)", lang, modelId, strings.Join(api.ValidLangsForModel(modelId), ", "))
				}
			}

			resolved, err := config.ResolveConfigWithOptions(config.ResolveOptions{
				EnvName:        ConfigEnv,
				APIURLOverride: apiURL,
				ConfigFile:     ConfigFile,
			})
			if err != nil {
				return err
			}

			curlOpts := CurlOptions{
				Text:       text,
				Speaker:    spk,
				ModelID:    modelId,
				Lang:       lang,
				ShowKey:    showKey,
				Oneline:    oneline,
				APIURL:     resolved.APIURL,
				AuthPrefix: resolved.AuthHeaderPrefix,
			}

			if showKey {
				curlOpts.APIKey = resolved.APIKey
			}

			ttsOpts := &api.TTSOptions{ModelID: modelId}
			modelParams.applyChanged(cmd.Flags(), ttsOpts)
			if err := api.ValidateModelParams(ttsOpts); err != nil {
				return err
			}

			curlCmd, err := generateCurlCommand(curlOpts, ttsOpts)
			if err != nil {
				return err
			}

			fmt.Println(curlCmd)

			if !showKey && term.IsTerminal(int(os.Stdout.Fd())) {
				fmt.Fprintln(os.Stderr, "# Tip: use --show-key or export RIME_CLI_API_KEY")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&spk, "speaker", "s", "", "Voice speaker to use (required)")
	cmd.Flags().StringVarP(&modelId, "model-id", "m", "", fmt.Sprintf("Model ID (required, e.g., %s, %s, %s, %s)", api.ModelIDArcana, api.ModelIDArcanaV2, api.ModelIDMistV2, api.ModelIDMist))
	cmd.Flags().StringVar(&modelId, "modelId", "", "")
	cmd.Flags().MarkHidden("modelId")
	cmd.Flags().StringVarP(&lang, "lang", "l", "eng", "Language code (e.g., eng, es, fra). Valid codes depend on model.")
	cmd.Flags().BoolVar(&showKey, "show-key", false, "Include actual API key (default: $RIME_CLI_API_KEY)")
	cmd.Flags().BoolVar(&oneline, "oneline", false, "Output as single line (easier to copy-paste)")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "API URL (default: $RIME_API_URL or https://users.rime.ai/v1/rime-tts)")

	modelParams.register(cmd.Flags())

	return cmd
}
