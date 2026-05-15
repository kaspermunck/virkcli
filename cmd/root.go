package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "virkcli",
	Short: "CLI for querying Danish company data via the VIRK API",
	Long: `CLI for querying Danish company data via the official VIRK API
(Erhvervsstyrelsen / CVR).

Commands: lookup, search, financials, person, punit.

Financial data is extracted from XBRL filings. PDF-only annual reports (common
for banks, IFRS reporters, and older filings) are recognised and listed but
their figures are not extracted.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
