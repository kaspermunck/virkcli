package virk

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const punitBase = "http://distribution.virk.dk/cvr-permanent/produktionsenhed/_search"

// PUnit describes a production unit (P-enhed) under a Danish company.
type PUnit struct {
	PNumber    string   `json:"pNumber"`
	ParentCVR  string   `json:"parentCvr,omitempty"`
	Name       string   `json:"name,omitempty"`
	Status     string   `json:"status,omitempty"`
	Address    Address  `json:"address"`
	Industry   Industry `json:"industry,omitempty"`
	Employees  string   `json:"employees,omitempty"`
	Email      string   `json:"email,omitempty"`
	Phone      string   `json:"phone,omitempty"`
	Website    string   `json:"website,omitempty"`
}

// PUnit returns the production unit metadata for a given P-number.
func (c *Client) PUnit(pnr string) (*PUnit, error) {
	raw, err := c.PUnitRaw(pnr)
	if err != nil {
		return nil, err
	}
	list, err := parsePUnits(raw)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("no production unit found with P-number %s", pnr)
	}
	return &list[0], nil
}

// PUnitsByCVR returns all production units belonging to a CVR.
func (c *Client) PUnitsByCVR(cvr string) ([]PUnit, error) {
	raw, err := c.pUnitsByCVRRaw(cvr)
	if err != nil {
		return nil, err
	}
	return parsePUnits(raw)
}

// PUnitRaw runs a P-number lookup and returns the raw Elasticsearch body.
func (c *Client) PUnitRaw(pnr string) ([]byte, error) {
	pnrInt, err := strconv.Atoi(pnr)
	if err != nil {
		return nil, fmt.Errorf("invalid P-number: %s", pnr)
	}
	query := map[string]any{
		"query": map[string]any{"term": map[string]any{"VrproduktionsEnhed.pNummer": pnrInt}},
		"size":  1,
	}
	return c.postJSON(punitBase, query)
}

func (c *Client) pUnitsByCVRRaw(cvr string) ([]byte, error) {
	cvrInt, err := strconv.Atoi(cvr)
	if err != nil {
		return nil, fmt.Errorf("invalid CVR: %s", cvr)
	}
	query := map[string]any{
		"query": map[string]any{"term": map[string]any{"VrproduktionsEnhed.virksomhedsrelation.cvrNummer": cvrInt}},
		"size":  100,
	}
	return c.postJSON(punitBase, query)
}

func parsePUnits(raw []byte) ([]PUnit, error) {
	var resp struct {
		Hits struct {
			Hits []struct {
				Source struct {
					VrproduktionsEnhed struct {
						PNummer  int `json:"pNummer"`
						Metadata struct {
							NyesteNavn                periodedString      `json:"nyesteNavn"`
							NyesteBeliggenhedsadresse beliggenhedsadresse `json:"nyesteBeliggenhedsadresse"`
							NyesteHovedbranche        hovedbranche        `json:"nyesteHovedbranche"`
							NyesteKontaktoplysninger  []string            `json:"nyesteKontaktoplysninger"`
							NyesteCvrNummerRelation   int                 `json:"nyesteCvrNummerRelation"`
							NyesteAarsbeskaeftigelse  *struct {
								IntervalKodeAntalAnsatte string `json:"intervalKodeAntalAnsatte"`
							} `json:"nyesteAarsbeskaeftigelse"`
							SammensatStatus string `json:"sammensatStatus"`
						} `json:"produktionsEnhedMetadata"`
					} `json:"VrproduktionsEnhed"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse P-unit response: %w", err)
	}
	out := make([]PUnit, 0, len(resp.Hits.Hits))
	for _, h := range resp.Hits.Hits {
		p := h.Source.VrproduktionsEnhed
		m := p.Metadata
		industryText := m.NyesteHovedbranche.BranchetekstLang
		if industryText == "" {
			industryText = m.NyesteHovedbranche.Branchetekst
		}
		punit := PUnit{
			PNumber:   strconv.Itoa(p.PNummer),
			Name:      m.NyesteNavn.Navn,
			Status:    m.SammensatStatus,
			Address:   buildAddress(m.NyesteBeliggenhedsadresse),
			Industry:  Industry{Code: m.NyesteHovedbranche.Branchekode, Text: industryText},
		}
		if m.NyesteCvrNummerRelation != 0 {
			punit.ParentCVR = strconv.Itoa(m.NyesteCvrNummerRelation)
		}
		if m.NyesteAarsbeskaeftigelse != nil {
			punit.Employees = m.NyesteAarsbeskaeftigelse.IntervalKodeAntalAnsatte
		}
		punit.Email, punit.Phone, punit.Website = classifyContacts(m.NyesteKontaktoplysninger)
		out = append(out, punit)
	}
	return out, nil
}
