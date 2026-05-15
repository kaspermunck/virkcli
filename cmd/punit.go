package cmd

import (
	"fmt"
	"os"

	"github.com/kaspermunck/virkcli/virk"
	"github.com/spf13/cobra"
)

var (
	punitCVR      string
	punitRaw      bool
	punitJSON     bool
	punitEnvelope bool
)

var punitCmd = &cobra.Command{
	Use:   "punit <pNummer>",
	Short: "Look up a production unit (P-enhed) by P-number, or list P-units for a CVR",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && punitCVR == "" {
			return fmt.Errorf("provide a P-number or --cvr")
		}
		if len(args) == 1 && punitCVR != "" {
			return fmt.Errorf("pass either a P-number or --cvr, not both")
		}

		client, err := virk.NewClientFromEnv()
		if err != nil {
			return err
		}

		if punitRaw {
			var raw []byte
			if punitCVR != "" {
				return fmt.Errorf("--raw for --cvr lookups is not implemented yet")
			}
			raw, err = client.PUnitRaw(args[0])
			if err != nil {
				return err
			}
			return writePrettyJSON(raw)
		}

		if punitCVR != "" {
			list, err := client.PUnitsByCVR(punitCVR)
			if err != nil {
				return err
			}
			if punitEnvelope {
				return encodeJSON(virk.Wrap("ProductionUnitList", list))
			}
			if punitJSON {
				return encodeJSON(list)
			}
			printPUnitList(list)
			return nil
		}

		p, err := client.PUnit(args[0])
		if err != nil {
			return err
		}
		if punitEnvelope {
			return encodeJSON(virk.Wrap("ProductionUnit", p))
		}
		if punitJSON {
			return encodeJSON(p)
		}
		printPUnit(p)
		return nil
	},
}

func init() {
	punitCmd.Flags().StringVar(&punitCVR, "cvr", "", "list all production units for this CVR")
	punitCmd.Flags().BoolVar(&punitRaw, "raw", false, "print the raw Elasticsearch response as JSON")
	punitCmd.Flags().BoolVar(&punitJSON, "json", false, "print the parsed P-unit(s) as JSON")
	punitCmd.Flags().BoolVar(&punitEnvelope, "envelope", false, "emit the shared envelope (Kind=ProductionUnit or ProductionUnitList with --cvr)")
	rootCmd.AddCommand(punitCmd)
}

func printPUnit(p *virk.PUnit) {
	w := os.Stdout
	fmt.Fprintln(w, "Production unit")
	fmt.Fprintf(w, "  P-number: %s\n", p.PNumber)
	if p.Name != "" {
		fmt.Fprintf(w, "  Name:     %s\n", p.Name)
	}
	if p.ParentCVR != "" {
		fmt.Fprintf(w, "  Parent:   CVR %s\n", p.ParentCVR)
	}
	if p.Status != "" {
		fmt.Fprintf(w, "  Status:   %s\n", p.Status)
	}
	fmt.Fprintln(w, "\nAddress")
	if line := addressLine1(p.Address); line != "" {
		fmt.Fprintf(w, "  %s\n", line)
	}
	if line := addressLine2(p.Address); line != "" {
		fmt.Fprintf(w, "  %s\n", line)
	}
	if p.Address.Municipality != "" {
		fmt.Fprintf(w, "  Municipality: %s\n", p.Address.Municipality)
	}
	if p.Industry.Code != "" || p.Industry.Text != "" {
		fmt.Fprintln(w, "\nIndustry")
		if p.Industry.Code != "" {
			fmt.Fprintf(w, "  %s — %s\n", p.Industry.Code, p.Industry.Text)
		} else {
			fmt.Fprintf(w, "  %s\n", p.Industry.Text)
		}
	}
	if p.Email != "" || p.Phone != "" || p.Website != "" {
		fmt.Fprintln(w, "\nContact")
		if p.Email != "" {
			fmt.Fprintf(w, "  Email:   %s\n", p.Email)
		}
		if p.Phone != "" {
			fmt.Fprintf(w, "  Phone:   %s\n", p.Phone)
		}
		if p.Website != "" {
			fmt.Fprintf(w, "  Website: %s\n", p.Website)
		}
	}
	if p.Employees != "" {
		fmt.Fprintf(w, "\n  Employees: %s\n", p.Employees)
	}
}

func printPUnitList(list []virk.PUnit) {
	if len(list) == 0 {
		fmt.Fprintln(os.Stdout, "No production units.")
		return
	}
	fmt.Fprintf(os.Stdout, "%-12s %-10s %-20s %s\n", "P-number", "Status", "City", "Name")
	for _, p := range list {
		fmt.Fprintf(os.Stdout, "%-12s %-10s %-20s %s\n", p.PNumber, trunc(p.Status, 10), trunc(p.Address.City, 20), p.Name)
	}
}
