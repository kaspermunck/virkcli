package cmd

import (
	"fmt"
	"os"

	"github.com/kaspermunck/virkcli/virk"
	"github.com/spf13/cobra"
)

var (
	ejerActiveOnly bool
	ejerLimit      int
	ejerRaw        bool
	ejerJSON       bool
	ejerEnvelope   bool
)

var ejerCmd = &cobra.Command{
	Use:   "ejer <cvr>",
	Short: "Find companies in which the given CVR is registered as a deltager (reverse ownership)",
	Long: `Reverse-ownership lookup: list every Danish company in which <cvr> appears as
a deltager — owner, stifter, board member, or auditor.

The CVR system records each owner relation on the *owned* company's record, so
a holding company's portfolio is otherwise invisible without already knowing
the portfolio companies. This command bridges that gap.

Use --active-only to exclude relations that have ended.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := virk.NewClientFromEnv()
		if err != nil {
			return err
		}
		opts := virk.EjerOpts{
			CVR:        args[0],
			ActiveOnly: ejerActiveOnly,
			Limit:      ejerLimit,
		}

		if ejerRaw {
			raw, err := client.EjerRaw(opts)
			if err != nil {
				return err
			}
			return writePrettyJSON(raw)
		}

		hits, err := client.Ejer(opts)
		if err != nil {
			return err
		}
		if ejerEnvelope {
			return encodeJSON(virk.Wrap("EjerList", hits))
		}
		if ejerJSON {
			return encodeJSON(hits)
		}
		printEjerHits(hits)
		return nil
	},
}

func init() {
	ejerCmd.Flags().BoolVar(&ejerActiveOnly, "active-only", false, "only include relations that have not ended")
	ejerCmd.Flags().IntVar(&ejerLimit, "limit", 50, "max number of target companies to return")
	ejerCmd.Flags().BoolVar(&ejerRaw, "raw", false, "print the raw Elasticsearch response as JSON")
	ejerCmd.Flags().BoolVar(&ejerJSON, "json", false, "print the parsed hits as JSON")
	ejerCmd.Flags().BoolVar(&ejerEnvelope, "envelope", false, "emit the shared envelope (Kind=EjerList)")
	rootCmd.AddCommand(ejerCmd)
}

func printEjerHits(hits []virk.EjerHit) {
	if len(hits) == 0 {
		fmt.Fprintln(os.Stdout, "No relations found.")
		return
	}
	fmt.Fprintf(os.Stdout, "%-10s %-30s %-22s %-10s %s\n", "CVR", "Name", "Role", "Ownership", "Active")
	for _, h := range hits {
		active := "yes"
		if !h.Active {
			active = "no"
			if h.EndedAt != "" {
				active = fmt.Sprintf("no (ended %s)", h.EndedAt)
			}
		}
		ownership := h.OwnershipPct
		if ownership != "" {
			ownership += "%"
		}
		fmt.Fprintf(os.Stdout, "%-10s %-30s %-22s %-10s %s\n",
			h.CVR,
			trunc(h.Name, 30),
			trunc(h.Role, 22),
			ownership,
			active,
		)
	}
}
