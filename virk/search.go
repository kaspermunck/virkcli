package virk

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// SearchOpts controls a company search query.
type SearchOpts struct {
	Query  string
	City   string
	Active bool
	Limit  int
}

// SearchHit is the compact shape returned by search — enough for a picklist.
type SearchHit struct {
	CVR    string `json:"cvr"`
	Name   string `json:"name"`
	Form   string `json:"form,omitempty"`
	Status string `json:"status,omitempty"`
	City   string `json:"city,omitempty"`
	Score  float64 `json:"score,omitempty"`
}

func (c *Client) Search(opts SearchOpts) ([]SearchHit, error) {
	raw, err := c.SearchRaw(opts)
	if err != nil {
		return nil, err
	}
	return parseSearchHits(raw)
}

// SearchRaw runs the search query and returns the raw Elasticsearch body.
func (c *Client) SearchRaw(opts SearchOpts) ([]byte, error) {
	query := buildSearchQuery(opts)
	return c.postJSON(cvrBase, query)
}

func buildSearchQuery(opts SearchOpts) map[string]any {
	must := []any{}
	if opts.Query != "" {
		must = append(must, map[string]any{
			"match": map[string]any{
				"Vrvirksomhed.virksomhedMetadata.nyesteNavn.navn": opts.Query,
			},
		})
	}
	if opts.City != "" {
		must = append(must, map[string]any{
			"match": map[string]any{
				"Vrvirksomhed.virksomhedMetadata.nyesteBeliggenhedsadresse.postdistrikt": opts.City,
			},
		})
	}
	if opts.Active {
		must = append(must, map[string]any{
			"match": map[string]any{
				"Vrvirksomhed.virksomhedMetadata.sammensatStatus": "NORMAL",
			},
		})
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	q := map[string]any{
		"size": limit,
		"_source": map[string]any{
			"includes": []string{
				"Vrvirksomhed.cvrNummer",
				"Vrvirksomhed.virksomhedMetadata.nyesteNavn.navn",
				"Vrvirksomhed.virksomhedMetadata.nyesteVirksomhedsform",
				"Vrvirksomhed.virksomhedMetadata.nyesteBeliggenhedsadresse.postdistrikt",
				"Vrvirksomhed.virksomhedMetadata.sammensatStatus",
			},
		},
	}
	if len(must) == 0 {
		q["query"] = map[string]any{"match_all": map[string]any{}}
	} else {
		q["query"] = map[string]any{"bool": map[string]any{"must": must}}
	}
	return q
}

func parseSearchHits(raw []byte) ([]SearchHit, error) {
	var resp struct {
		Hits struct {
			Hits []struct {
				Score  float64 `json:"_score"`
				Source struct {
					Vrvirksomhed struct {
						CVR      int `json:"cvrNummer"`
						Metadata struct {
							NyesteNavn                periodedString      `json:"nyesteNavn"`
							NyesteVirksomhedsform     virksomhedsform     `json:"nyesteVirksomhedsform"`
							NyesteBeliggenhedsadresse beliggenhedsadresse `json:"nyesteBeliggenhedsadresse"`
							SammensatStatus           string              `json:"sammensatStatus"`
						} `json:"virksomhedMetadata"`
					} `json:"Vrvirksomhed"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}
	out := make([]SearchHit, 0, len(resp.Hits.Hits))
	for _, h := range resp.Hits.Hits {
		v := h.Source.Vrvirksomhed
		m := v.Metadata
		out = append(out, SearchHit{
			CVR:    strconv.Itoa(v.CVR),
			Name:   m.NyesteNavn.Navn,
			Form:   m.NyesteVirksomhedsform.KortBeskrivelse,
			Status: m.SammensatStatus,
			City:   m.NyesteBeliggenhedsadresse.Postdistrikt,
			Score:  h.Score,
		})
	}
	return out, nil
}
