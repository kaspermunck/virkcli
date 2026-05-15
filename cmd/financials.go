package cmd

import (
	"fmt"
	"os"

	"github.com/kaspermunck/virkcli/virk"
	"github.com/spf13/cobra"
)

var (
	financialsRaw      bool
	financialsRawXBRL  bool
	financialsJSON     bool
	financialsEnvelope bool
	financialsYear     int
	financialsAll      bool
)

var financialsCmd = &cobra.Command{
	Use:   "financials <cvr>",
	Short: "Fetch annual report figures for a Danish company (XBRL only; PDF filings are listed but not extracted)",
	Long: `Fetch annual report figures for a Danish company from VIRK.

Extracts revenue, gross profit, profit, equity, and assets from XBRL-backed
filings. PDF-only filings (common for banks and IFRS reporters) are recognised
and listed, but their figures cannot be extracted automatically.

Use --all to see the full filing history — PDF-only years are marked with *.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateFinancialsFlags(); err != nil {
			return err
		}

		client, err := virk.NewClientFromEnv()
		if err != nil {
			return err
		}
		cvr := args[0]

		switch {
		case financialsAll && financialsRaw:
			return writeBytes(client.FinancialsRawAll(cvr))

		case financialsAll:
			list, err := client.FinancialsAll(cvr)
			if err != nil {
				return err
			}
			if financialsEnvelope {
				return encodeJSON(virk.Wrap("FinancialsHistory", list))
			}
			if financialsJSON {
				return encodeJSON(list)
			}
			printFinancialsList(list)
			return nil

		case financialsRaw, financialsRawXBRL:
			filingJSON, xbrlDoc, err := client.FinancialsRaw(cvr, financialsYear)
			if err != nil {
				return err
			}
			if financialsRawXBRL {
				_, _ = os.Stdout.Write(xbrlDoc)
				return nil
			}
			_, _ = os.Stdout.Write(filingJSON)
			fmt.Fprintln(os.Stdout)
			return nil

		default:
			var fin *virk.Financials
			if financialsYear != 0 {
				fin, err = client.FinancialsByYear(cvr, financialsYear)
			} else {
				fin, err = client.Financials(cvr)
			}
			if err != nil {
				return err
			}
			if financialsEnvelope {
				return encodeJSON(virk.Wrap("Financials", fin))
			}
			if financialsJSON {
				return encodeJSON(fin)
			}
			printFinancials(fin)
			return nil
		}
	},
}

func validateFinancialsFlags() error {
	if financialsAll && financialsYear != 0 {
		return fmt.Errorf("--all and --year are mutually exclusive")
	}
	if financialsAll && financialsRawXBRL {
		return fmt.Errorf("--all cannot be combined with --raw-xbrl (multiple documents cannot be concatenated)")
	}
	if financialsRaw && financialsRawXBRL {
		return fmt.Errorf("--raw and --raw-xbrl are mutually exclusive")
	}
	return nil
}

func init() {
	financialsCmd.Flags().BoolVar(&financialsRaw, "raw", false, "print the raw filing metadata as JSON")
	financialsCmd.Flags().BoolVar(&financialsRawXBRL, "raw-xbrl", false, "print the raw XBRL/iXBRL document")
	financialsCmd.Flags().BoolVar(&financialsJSON, "json", false, "print the parsed financials as JSON")
	financialsCmd.Flags().IntVar(&financialsYear, "year", 0, "pick the AARSRAPPORT whose fiscal year ends in this calendar year (e.g. --year 2024)")
	financialsCmd.Flags().BoolVar(&financialsAll, "all", false, "list every annual report (XBRL figures extracted; PDF-only filings marked with *)")
	financialsCmd.Flags().BoolVar(&financialsEnvelope, "envelope", false, "emit the shared envelope (Kind=Financials or FinancialsHistory with --all)")
	rootCmd.AddCommand(financialsCmd)
}

func printFinancials(f *virk.Financials) {
	fmt.Fprintln(os.Stdout, "Financials")
	if f.FiscalYearEnd != "" {
		fmt.Fprintf(os.Stdout, "  Fiscal year end:  %s\n", f.FiscalYearEnd)
	}
	if f.PDFOnly {
		fmt.Fprintln(os.Stdout, "\n  Annual report is PDF-only — no XBRL data available for extraction.")
		return
	}
	printField("Revenue", f.Revenue)
	printField("Gross profit", f.GrossProfit)
	printField("Profit", f.Profit)
	printField("Equity", f.Equity)
	printField("Assets", f.Assets)
}

func printFinancialsList(list []*virk.Financials) {
	if len(list) == 0 {
		fmt.Fprintln(os.Stdout, "No annual reports found.")
		return
	}
	fmt.Fprintf(os.Stdout, "%-14s %15s %15s %15s %15s %15s\n", "Year end", "Revenue", "Gross profit", "Profit", "Equity", "Assets")
	hasPDF := false
	for _, f := range list {
		yearEnd := f.FiscalYearEnd
		if f.PDFOnly {
			yearEnd += " *"
			hasPDF = true
		}
		fmt.Fprintf(os.Stdout, "%-14s %15s %15s %15s %15s %15s\n",
			yearEnd,
			dkkCell(f.Revenue),
			dkkCell(f.GrossProfit),
			dkkCell(f.Profit),
			dkkCell(f.Equity),
			dkkCell(f.Assets),
		)
	}
	if hasPDF {
		fmt.Fprintln(os.Stdout, "\n* PDF-only — no XBRL data available for extraction.")
	}
}

