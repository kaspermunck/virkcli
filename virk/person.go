package virk

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const personBase = "http://distribution.virk.dk/cvr-permanent/deltager/_search"

// PersonHit is the compact shape used for a name search.
type PersonHit struct {
	EnhedsNummer int64   `json:"enhedsNummer"`
	Name         string  `json:"name"`
	RelationCount int    `json:"relationCount"`
	Score        float64 `json:"score,omitempty"`
}

// Person is the detailed shape for a single person lookup by enhedsNummer.
type Person struct {
	EnhedsNummer int64            `json:"enhedsNummer"`
	Name         string           `json:"name"`
	AddressHidden bool            `json:"addressHidden,omitempty"`
	Address      *Address         `json:"address,omitempty"`
	Relations    []PersonRelation `json:"relations,omitempty"`
}

// PersonRelation is a role held at a specific company by a person.
type PersonRelation struct {
	CVR      string `json:"cvr,omitempty"`
	Company  string `json:"company"`
	Type     string `json:"type,omitempty"` // hovedtype (REGISTER, LEDELSESORGAN, ...)
	Role     string `json:"role,omitempty"` // humanised (Direktør, Bestyrelsesmedlem, ...)
	EndedAt  string `json:"endedAt,omitempty"`
	Active   bool   `json:"active"`
}

// SearchPersons runs a fuzzy name match against the deltager index.
func (c *Client) SearchPersons(name string, limit int) ([]PersonHit, error) {
	raw, err := c.searchPersonsRaw(name, limit)
	if err != nil {
		return nil, err
	}
	return parsePersonHits(raw)
}

// SearchPersonsRaw returns the raw Elasticsearch response for a name search.
func (c *Client) SearchPersonsRaw(name string, limit int) ([]byte, error) {
	return c.searchPersonsRaw(name, limit)
}

func (c *Client) searchPersonsRaw(name string, limit int) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("name must not be empty")
	}
	if limit <= 0 {
		limit = 10
	}
	query := map[string]any{
		"query": map[string]any{"match": map[string]any{"Vrdeltagerperson.navne.navn": name}},
		"size":  limit,
	}
	return c.postJSON(personBase, query)
}

// PersonByID fetches detailed info for a person by their deltager enhedsNummer.
func (c *Client) PersonByID(id int64, activeOnly bool) (*Person, error) {
	raw, err := c.personByIDRaw(id)
	if err != nil {
		return nil, err
	}
	return parsePersonDetail(raw, activeOnly)
}

// PersonByIDRaw returns the raw Elasticsearch response for a deltager by enhedsNummer.
func (c *Client) PersonByIDRaw(id int64) ([]byte, error) {
	return c.personByIDRaw(id)
}

func (c *Client) personByIDRaw(id int64) ([]byte, error) {
	query := map[string]any{
		"query": map[string]any{"term": map[string]any{"Vrdeltagerperson.enhedsNummer": id}},
		"size":  1,
	}
	return c.postJSON(personBase, query)
}

type personSource struct {
	EnhedsNummer   int64            `json:"enhedsNummer"`
	Navne          []periodedString `json:"navne"`
	AdresseHemmelig bool            `json:"adresseHemmelig"`
	Beliggenhedsadresse []struct {
		beliggenhedsadresse
		Periode struct {
			GyldigFra string  `json:"gyldigFra"`
			GyldigTil *string `json:"gyldigTil"`
		} `json:"periode"`
	} `json:"beliggenhedsadresse"`
	VirksomhedSummariskRelation []struct {
		Virksomhed struct {
			CVRNummer int              `json:"cvrNummer"`
			Navne     []periodedString `json:"navne"`
		} `json:"virksomhed"`
		Organisationer []deltagerOrganisation `json:"organisationer"`
	} `json:"virksomhedSummariskRelation"`
}

func parsePersonHits(raw []byte) ([]PersonHit, error) {
	var resp struct {
		Hits struct {
			Hits []struct {
				Score  float64 `json:"_score"`
				Source struct {
					Vrdeltagerperson personSource `json:"Vrdeltagerperson"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse person search response: %w", err)
	}
	out := make([]PersonHit, 0, len(resp.Hits.Hits))
	for _, h := range resp.Hits.Hits {
		p := h.Source.Vrdeltagerperson
		out = append(out, PersonHit{
			EnhedsNummer:  p.EnhedsNummer,
			Name:          latestName(p.Navne),
			RelationCount: len(p.VirksomhedSummariskRelation),
			Score:         h.Score,
		})
	}
	return out, nil
}

func parsePersonDetail(raw []byte, activeOnly bool) (*Person, error) {
	var resp struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Vrdeltagerperson personSource `json:"Vrdeltagerperson"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse person detail response: %w", err)
	}
	if len(resp.Hits.Hits) == 0 {
		return nil, fmt.Errorf("no person found")
	}
	p := resp.Hits.Hits[0].Source.Vrdeltagerperson
	person := &Person{
		EnhedsNummer:  p.EnhedsNummer,
		Name:          latestName(p.Navne),
		AddressHidden: p.AdresseHemmelig,
	}
	if !p.AdresseHemmelig {
		for i := len(p.Beliggenhedsadresse) - 1; i >= 0; i-- {
			a := p.Beliggenhedsadresse[i]
			if a.Periode.GyldigTil == nil {
				addr := buildAddress(a.beliggenhedsadresse)
				person.Address = &addr
				break
			}
		}
	}
	for _, rel := range p.VirksomhedSummariskRelation {
		companyName := latestName(rel.Virksomhed.Navne)
		for _, org := range rel.Organisationer {
			funktion, endedAt := funktionAndEnd(org.MedlemsData)
			active := endedAt == ""
			if activeOnly && !active {
				continue
			}
			role := ownerRole(org.Hovedtype, funktion)
			person.Relations = append(person.Relations, PersonRelation{
				CVR:     strconv.Itoa(rel.Virksomhed.CVRNummer),
				Company: companyName,
				Type:    org.Hovedtype,
				Role:    role,
				EndedAt: endedAt,
				Active:  active,
			})
		}
	}
	return person, nil
}

// funktionAndEnd reads the latest FUNKTION value and its gyldigTil date (if any).
func funktionAndEnd(md []medlemsData) (funktion, endedAt string) {
	for _, m := range md {
		for _, a := range m.Attributter {
			if a.Type != "FUNKTION" || len(a.Vaerdier) == 0 {
				continue
			}
			last := a.Vaerdier[len(a.Vaerdier)-1]
			funktion = last.Vaerdi
			if last.Periode.GyldigTil != nil {
				endedAt = *last.Periode.GyldigTil
			}
		}
	}
	return
}
