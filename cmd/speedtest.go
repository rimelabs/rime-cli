package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/output/styles"
)

type SpeedtestResult struct {
	Environment string        `json:"environment"`
	APIURL      string        `json:"api_url"`
	TTFB        time.Duration `json:"ttfb_ns"`
	TTFBMs      float64       `json:"ttfb_ms"`
	Error       string        `json:"error,omitempty"`
}

func NewSpeedtestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "speedtest",
		Short: "Measure TTFB for all configured endpoints",
		Long:  "Performs a TTS request against all configured endpoints and reports the time to first byte (TTFB) for each",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg *config.Config
			var err error

			if ConfigFile != "" {
				cfg, err = config.LoadConfigFromPath(ConfigFile)
			} else {
				cfg, err = config.LoadConfig()
			}
			if err != nil {
				return err
			}

			if cfg == nil {
				return fmt.Errorf("no config file found. Run 'rime config init' to create one")
			}

			envs := cfg.ListEnvironments()
			if len(envs) == 0 {
				return fmt.Errorf("no environments configured")
			}

			text := fmt.Sprintf("good %s from Rime AI!", getGreeting())
			opts := &api.TTSOptions{
				Speaker: "astra",
				ModelID: "arcana",
				Lang:    "eng",
			}

			results := make([]SpeedtestResult, 0, len(envs))

			if !JSONOutput && !Quiet {
				fmt.Printf("%-15s %-50s %s\n", "ENV", "URL", "TTFB")
				fmt.Println(strings.Repeat("-", 80))
			}

			for _, envName := range envs {
				env, err := cfg.ResolveEnvironment(envName)
				if err != nil {
					result := SpeedtestResult{
						Environment: envName,
						Error:       err.Error(),
					}
					results = append(results, result)
					if !JSONOutput && !Quiet {
						fmt.Printf("%-15s %-50s %s\n", envName, "(error)", styles.Error(err.Error()))
					}
					continue
				}

				client := api.NewClientWithOptions(api.ClientOptions{
					APIKey:           env.GetAPIKey(),
					APIURL:           env.APIURL,
					AuthHeaderPrefix: getAuthPrefix(env),
					Version:          Version,
				})

				streamResult, err := client.TTSStream(text, opts)
				if err != nil {
					result := SpeedtestResult{
						Environment: envName,
						APIURL:      env.APIURL,
						Error:       err.Error(),
					}
					results = append(results, result)
					if !JSONOutput && !Quiet {
						fmt.Printf("%-15s %-50s %s\n", envName, truncateURL(env.APIURL, 50), styles.Error(err.Error()))
					}
					continue
				}

				streamResult.Body.Close()

				result := SpeedtestResult{
					Environment: envName,
					APIURL:      env.APIURL,
					TTFB:        streamResult.TTFB,
					TTFBMs:      float64(streamResult.TTFB.Microseconds()) / 1000.0,
				}
				results = append(results, result)

				if !JSONOutput && !Quiet {
					ttfbStr := formatTTFB(streamResult.TTFB)
					fmt.Printf("%-15s %-50s %s\n", envName, truncateURL(env.APIURL, 50), styles.Success(ttfbStr))
				}
			}

			if JSONOutput {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(results)
			}

			if !Quiet {
				fastest := findFastest(results)
				if fastest != nil {
					fmt.Printf("\n%s %s (%s)\n", styles.Success("Fastest:"), fastest.Environment, formatTTFB(fastest.TTFB))
				}
			}

			return nil
		},
	}

	return cmd
}

func getAuthPrefix(env *config.Environment) string {
	if env.AuthHeaderPrefix != nil {
		return *env.AuthHeaderPrefix
	}
	return "Bearer"
}

func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}

func formatTTFB(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000.0)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func findFastest(results []SpeedtestResult) *SpeedtestResult {
	var fastest *SpeedtestResult
	for i := range results {
		if results[i].Error != "" || results[i].TTFB == 0 {
			continue
		}
		if fastest == nil || results[i].TTFB < fastest.TTFB {
			fastest = &results[i]
		}
	}
	return fastest
}
