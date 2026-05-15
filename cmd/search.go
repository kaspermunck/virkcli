package cmd

import (
	"fmt"
	"os"

	"github.com/kaspermunck/virkcli/virk"
	"github.com/spf13/cobra"
)

var (
	searchCity     string
	searchLimit    int
	searchActive   bool
	searchRaw      bool
	searchJSON     bool
	searchEnvelope bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for Danish companies by name",
	Long:  "Search for Danish companies by name (fuzzy). Use --city to narrow by postal district and --active to filter to status NORMAL.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := ""
		if len(args) == 1 {
			query = args[0]
		}
		if query == "" && searchCity == "" {
			return fmt.Errorf("provide a search query or --city")
		}

		client, err := virk.NewClientFromEnv()
		if err != nil {
			return err
		}
		opts := virk.SearchOpts{
			Query:  query,
			City:   searchCity,
			Active: searchActive,
			Limit:  searchLimit,
		}

		if searchRaw {
			raw, err := client.SearchRaw(opts)
			if err != nil {
				return err
			}
			return writePrettyJSON(raw)
		}

		hits, err := client.Search(opts)
		if err != nil {
			return err
		}
		if searchEnvelope {
			return encodeJSON(virk.Wrap("CompanySearch", hits))
		}
		if searchJSON {
			return encodeJSON(hits)
		}
		printSearchHits(hits)
		return nil
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchCity, "city", "", "filter by postal district (postdistrikt)")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "max number of results")
	searchCmd.Flags().BoolVar(&searchActive, "active", false, "only include companies with sammensatStatus NORMAL")
	searchCmd.Flags().BoolVar(&searchRaw, "raw", false, "print the raw Elasticsearch response as JSON")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "print the parsed hits as JSON")
	searchCmd.Flags().BoolVar(&searchEnvelope, "envelope", false, "emit the shared envelope (Kind=CompanySearch)")
	rootCmd.AddCommand(searchCmd)
}

func printSearchHits(hits []virk.SearchHit) {
	if len(hits) == 0 {
		fmt.Fprintln(os.Stdout, "No matches.")
		return
	}
	fmt.Fprintf(os.Stdout, "%-10s %-6s %-10s %-20s %s\n", "CVR", "Form", "Status", "City", "Name")
	for _, h := range hits {
		fmt.Fprintf(os.Stdout, "%-10s %-6s %-10s %-20s %s\n", h.CVR, trunc(h.Form, 6), trunc(h.Status, 10), trunc(h.City, 20), h.Name)
	}
}

