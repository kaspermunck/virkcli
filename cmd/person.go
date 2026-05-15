package cmd

import (
	"fmt"
	"os"

	"github.com/kaspermunck/virkcli/virk"
	"github.com/spf13/cobra"
)

var (
	personID         int64
	personActiveOnly bool
	personLimit      int
	personRaw        bool
	personJSON       bool
	personEnvelope   bool
)

var personCmd = &cobra.Command{
	Use:   "person <name>",
	Short: "Search for a person by name, or look up a specific person by enhedsNummer",
	Long:  "Searches the VIRK deltager index by name. Use --id to fetch the full detail record for a specific person.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && personID == 0 {
			return fmt.Errorf("provide a name or --id")
		}
		if len(args) == 1 && personID != 0 {
			return fmt.Errorf("pass either a name or --id, not both")
		}

		client, err := virk.NewClientFromEnv()
		if err != nil {
			return err
		}

		if personRaw {
			var raw []byte
			if personID != 0 {
				raw, err = client.PersonByIDRaw(personID)
			} else {
				raw, err = client.SearchPersonsRaw(args[0], personLimit)
			}
			if err != nil {
				return err
			}
			return writePrettyJSON(raw)
		}

		if personID != 0 {
			p, err := client.PersonByID(personID, personActiveOnly)
			if err != nil {
				return err
			}
			if personEnvelope {
				return encodeJSON(virk.Wrap("Person", p))
			}
			if personJSON {
				return encodeJSON(p)
			}
			printPerson(p)
			return nil
		}

		hits, err := client.SearchPersons(args[0], personLimit)
		if err != nil {
			return err
		}
		if personEnvelope {
			return encodeJSON(virk.Wrap("PersonSearch", hits))
		}
		if personJSON {
			return encodeJSON(hits)
		}
		printPersonHits(hits)
		return nil
	},
}

func init() {
	personCmd.Flags().Int64Var(&personID, "id", 0, "look up a person by deltager enhedsNummer")
	personCmd.Flags().BoolVar(&personActiveOnly, "active", false, "only include currently-active relations (requires --id)")
	personCmd.Flags().IntVar(&personLimit, "limit", 10, "max number of name-search results")
	personCmd.Flags().BoolVar(&personRaw, "raw", false, "print the raw Elasticsearch response as JSON")
	personCmd.Flags().BoolVar(&personJSON, "json", false, "print the parsed person(s) as JSON")
	personCmd.Flags().BoolVar(&personEnvelope, "envelope", false, "emit the shared envelope (Kind=Person with --id, PersonSearch otherwise)")
	rootCmd.AddCommand(personCmd)
}

func printPersonHits(hits []virk.PersonHit) {
	if len(hits) == 0 {
		fmt.Fprintln(os.Stdout, "No matches.")
		return
	}
	fmt.Fprintf(os.Stdout, "%-12s %-5s %s\n", "EnhedsNr", "Rels", "Name")
	for _, h := range hits {
		fmt.Fprintf(os.Stdout, "%-12d %-5d %s\n", h.EnhedsNummer, h.RelationCount, h.Name)
	}
}

func printPerson(p *virk.Person) {
	w := os.Stdout
	fmt.Fprintln(w, "Person")
	fmt.Fprintf(w, "  Name:     %s\n", p.Name)
	fmt.Fprintf(w, "  EnhedsNr: %d\n", p.EnhedsNummer)

	if p.AddressHidden {
		fmt.Fprintln(w, "\nAddress")
		fmt.Fprintln(w, "  (hidden)")
	} else if p.Address != nil {
		fmt.Fprintln(w, "\nAddress")
		if line := addressLine1(*p.Address); line != "" {
			fmt.Fprintf(w, "  %s\n", line)
		}
		if line := addressLine2(*p.Address); line != "" {
			fmt.Fprintf(w, "  %s\n", line)
		}
		if p.Address.Municipality != "" {
			fmt.Fprintf(w, "  Municipality: %s\n", p.Address.Municipality)
		}
	}

	if len(p.Relations) == 0 {
		return
	}
	fmt.Fprintln(w, "\nRelations")
	for _, r := range p.Relations {
		line := fmt.Sprintf("  %s — %s (CVR %s)", r.Company, r.Role, r.CVR)
		if !r.Active && r.EndedAt != "" {
			line += fmt.Sprintf(" — ended %s", r.EndedAt)
		} else if !r.Active {
			line += " — ended"
		}
		fmt.Fprintln(w, line)
	}
}
