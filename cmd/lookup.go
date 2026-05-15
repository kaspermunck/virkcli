package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/kaspermunck/virkcli/virk"
	"github.com/spf13/cobra"
)

var (
	lookupRaw      bool
	lookupJSON     bool
	lookupEnvelope bool
)

var lookupCmd = &cobra.Command{
	Use:   "lookup <cvr>",
	Short: "Look up a Danish company by CVR number",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := virk.NewClientFromEnv()
		if err != nil {
			return err
		}

		if lookupRaw {
			raw, err := client.LookupRaw(args[0])
			if err != nil {
				return err
			}
			return writePrettyJSON(raw)
		}

		company, err := client.Lookup(args[0])
		if err != nil {
			return err
		}

		if lookupEnvelope {
			return encodeJSON(virk.Wrap("Company", company))
		}
		if lookupJSON {
			return encodeJSON(company)
		}

		printCompany(company)
		return nil
	},
}

func init() {
	lookupCmd.Flags().BoolVar(&lookupRaw, "raw", false, "print the raw Elasticsearch response as JSON")
	lookupCmd.Flags().BoolVar(&lookupJSON, "json", false, "print the parsed company as JSON")
	lookupCmd.Flags().BoolVar(&lookupEnvelope, "envelope", false, "emit the shared envelope (Kind=Company)")
	rootCmd.AddCommand(lookupCmd)
}

func printCompany(c *virk.Company) {
	w := os.Stdout
	fmt.Fprintln(w, "Company")
	fmt.Fprintf(w, "  Name:     %s\n", c.Name)
	fmt.Fprintf(w, "  CVR:      %s\n", c.CVR)
	if c.Form != "" {
		form := c.Form
		if c.FormCode != "" {
			form = fmt.Sprintf("%s (%s)", c.Form, c.FormCode)
		}
		fmt.Fprintf(w, "  Form:     %s\n", form)
	}
	if c.Status != "" {
		fmt.Fprintf(w, "  Status:   %s\n", c.Status)
	}
	if c.Founded != "" {
		fmt.Fprintf(w, "  Founded:  %s\n", c.Founded)
	}
	if len(c.Aliases) > 0 {
		fmt.Fprintf(w, "  Aliases:  %s\n", strings.Join(c.Aliases, ", "))
	}

	fmt.Fprintln(w, "\nAddress")
	if line := addressLine1(c.Address); line != "" {
		fmt.Fprintf(w, "  %s\n", line)
	}
	if line := addressLine2(c.Address); line != "" {
		fmt.Fprintf(w, "  %s\n", line)
	}
	if c.Address.Municipality != "" {
		fmt.Fprintf(w, "  Municipality: %s\n", c.Address.Municipality)
	}

	if c.Industry.Code != "" || c.Industry.Text != "" {
		fmt.Fprintln(w, "\nIndustry")
		if c.Industry.Code != "" {
			fmt.Fprintf(w, "  %s — %s\n", c.Industry.Code, c.Industry.Text)
		} else {
			fmt.Fprintf(w, "  %s\n", c.Industry.Text)
		}
	}

	if c.Email != "" || c.Phone != "" || c.Website != "" {
		fmt.Fprintln(w, "\nContact")
		if c.Email != "" {
			fmt.Fprintf(w, "  Email:    %s\n", c.Email)
		}
		if c.Phone != "" {
			fmt.Fprintf(w, "  Phone:    %s\n", c.Phone)
		}
		if c.Website != "" {
			fmt.Fprintf(w, "  Website:  %s\n", c.Website)
		}
	}

	fmt.Fprintln(w, "\nScale")
	if c.Employees != "" {
		fmt.Fprintf(w, "  Employees: %s\n", c.Employees)
	}
	fmt.Fprintf(w, "  P-units:   %d\n", c.PUnitCount)

	if len(c.Owners) > 0 {
		fmt.Fprintln(w, "\nDeltagere")
		for _, o := range c.Owners {
			line := fmt.Sprintf("  %s — %s", o.Name, o.Role)
			if o.OwnershipPct != "" {
				line += fmt.Sprintf(" (%s%%)", o.OwnershipPct)
			}
			fmt.Fprintln(w, line)
		}
	}
}

