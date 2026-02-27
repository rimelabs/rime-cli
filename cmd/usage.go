package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/config"
)

func NewUsageCmd() *cobra.Command {
	var csvOutput bool

	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show recent API usage history",
		Long:  "Displays daily character usage for Mist and Arcana models over the past week",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := config.ResolveConfigWithOptions(config.ResolveOptions{
				EnvName:    ConfigEnv,
				ConfigFile: ConfigFile,
			})
			if err != nil {
				return err
			}

			if resolved.APIKey == "" {
				return fmt.Errorf("no API key configured. Run 'rime login' or set %s", config.EnvAPIKey)
			}

			client := api.NewOptimizeClient(resolved.APIKey, Version)
			history, err := client.GetRecentUsage()
			if err != nil {
				return err
			}

			if JSONOutput {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(history)
			}

			if csvOutput {
				return writeCSV(history)
			}

			return writeTable(history)
		},
	}

	cmd.Flags().BoolVar(&csvOutput, "csv", false, "Output results as CSV")

	return cmd
}

func writeTable(history *api.UsageHistory) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", "Day", "Mist Chars", "Arcana Chars", "Total")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", "----------", "----------", "------------", "----------")

	for _, d := range history.Data {
		total := d.MistChars + d.ArcanaChars
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n",
			d.Day,
			formatNumber(d.MistChars),
			formatNumber(d.ArcanaChars),
			formatNumber(total),
		)
	}

	return w.Flush()
}

func writeCSV(history *api.UsageHistory) error {
	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"day", "mist_chars", "arcana_chars", "total"})
	for _, d := range history.Data {
		total := d.MistChars + d.ArcanaChars
		w.Write([]string{
			d.Day,
			strconv.FormatInt(d.MistChars, 10),
			strconv.FormatInt(d.ArcanaChars, 10),
			strconv.FormatInt(total, 10),
		})
	}
	w.Flush()
	return w.Error()
}

func formatNumber(n int64) string {
	if n == 0 {
		return "0"
	}

	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
