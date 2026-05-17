package virk

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// EjerHit is one company in which the queried CVR appears as a deltager
// (owner, stifter, board member, auditor, ...). A single relation may
// produce multiple hits when the deltager holds several distinct roles
// (e.g. Stifter + Reel ejer) in the same target company.
type EjerHit struct {
	CVR          string `json:"cvr"`
	Name         string `json:"name"`
	Status       string `json:"status,omitempty"`
	Role         string `json:"role"`
	OwnershipPct string `json:"ownershipPct,omitempty"`
	EndedAt      string `json:"endedAt,omitempty"`
	Active       bool   `json:"active"`
}

// EjerOpts controls a reverse-ownership query.
type EjerOpts struct {
	CVR        string
	ActiveOnly bool
	Limit      int
}

// Ejer returns every company in which the given CVR is registered as a deltager.
// Roles include ownership (REGISTER / Reel ejer), founding (STIFTERE), management
// (LEDELSESORGAN), and audit (REVISION). Combine with --active-only to filter to
// relations that have not ended.
func (c *Client) Ejer(opts EjerOpts) ([]EjerHit, error) {
	raw, err := c.EjerRaw(opts)
	if err != nil {
		return nil, err
	}
	return parseEjerHits(raw, opts)
}

// EjerRaw returns the raw Elasticsearch body for a reverse-ownership query.
func (c *Client) EjerRaw(opts EjerOpts) ([]byte, error) {
	cvrInt, err := strconv.Atoi(opts.CVR)
	if err != nil {
		return nil, fmt.Errorf("invalid CVR number: %s", opts.CVR)
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	query := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"Vrvirksomhed.deltagerRelation.deltager.forretningsnoegle": cvrInt,
			},
		},
		"size": limit,
		"_source": map[string]any{
			"includes": []string{
				"Vrvirksomhed.cvrNummer",
				"Vrvirksomhed.virksomhedMetadata.nyesteNavn.navn",
				"Vrvirksomhed.virksomhedMetadata.sammensatStatus",
				"Vrvirksomhed.deltagerRelation",
			},
		},
	}
	return c.postJSON(cvrBase, query)
}

func parseEjerHits(raw []byte, opts EjerOpts) ([]EjerHit, error) {
	cvrInt, err := strconv.Atoi(opts.CVR)
	if err != nil {
		return nil, fmt.Errorf("invalid CVR number: %s", opts.CVR)
	}

	var resp struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Vrvirksomhed struct {
						CVR      int `json:"cvrNummer"`
						Metadata struct {
							NyesteNavn      periodedString `json:"nyesteNavn"`
							SammensatStatus string         `json:"sammensatStatus"`
						} `json:"virksomhedMetadata"`
						DeltagerRelation []struct {
							Deltager struct {
								Forretningsnoegle int              `json:"forretningsnoegle"`
								EnhedsType        string           `json:"enhedstype"`
								Navne             []periodedString `json:"navne"`
							} `json:"deltager"`
							Organisationer []deltagerOrganisation `json:"organisationer"`
						} `json:"deltagerRelation"`
					} `json:"Vrvirksomhed"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse ejer response: %w", err)
	}

	var out []EjerHit
	for _, h := range resp.Hits.Hits {
		v := h.Source.Vrvirksomhed
		for _, rel := range v.DeltagerRelation {
			if rel.Deltager.Forretningsnoegle != cvrInt {
				continue
			}
			for _, org := range rel.Organisationer {
				funktion, ejerandel, endedAt := orgAttrsTimed(org.MedlemsData)
				role := ownerRole(org.Hovedtype, funktion)
				active := endedAt == ""
				if opts.ActiveOnly && !active {
					continue
				}
				out = append(out, EjerHit{
					CVR:          strconv.Itoa(v.CVR),
					Name:         v.Metadata.NyesteNavn.Navn,
					Status:       v.Metadata.SammensatStatus,
					Role:         role,
					OwnershipPct: formatOwnership(ejerandel),
					EndedAt:      endedAt,
					Active:       active,
				})
			}
		}
	}
	return out, nil
}

// orgAttrsTimed returns the latest FUNKTION + EJERANDEL_PROCENT plus the
// gyldigTil of the most representative attribute. If the relation carries an
// ownership percentage, that period decides activity; otherwise FUNKTION does.
func orgAttrsTimed(md []medlemsData) (funktion, ejerandel, endedAt string) {
	var funktionEnd, ejerandelEnd string
	hasEjerandel := false
	for _, m := range md {
		for _, a := range m.Attributter {
			if len(a.Vaerdier) == 0 {
				continue
			}
			last := a.Vaerdier[len(a.Vaerdier)-1]
			switch a.Type {
			case "FUNKTION":
				funktion = last.Vaerdi
				if last.Periode.GyldigTil != nil {
					funktionEnd = *last.Periode.GyldigTil
				} else {
					funktionEnd = ""
				}
			case "EJERANDEL_PROCENT":
				ejerandel = last.Vaerdi
				hasEjerandel = true
				if last.Periode.GyldigTil != nil {
					ejerandelEnd = *last.Periode.GyldigTil
				} else {
					ejerandelEnd = ""
				}
			}
		}
	}
	if hasEjerandel {
		endedAt = ejerandelEnd
	} else {
		endedAt = funktionEnd
	}
	return
}
