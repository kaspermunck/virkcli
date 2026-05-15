package virk

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const financialsBase = "http://distribution.virk.dk/offentliggoerelser/_search"

// Financials holds the key figures extracted from a VIRK annual report (XBRL/iXBRL).
// For PDF-only filings, PDFOnly is true and the numeric fields are nil.
type Financials struct {
	CVR           string `json:"cvr"`
	FiscalYearEnd string `json:"fiscalYearEnd,omitempty"`
	PDFOnly       bool   `json:"pdfOnly,omitempty"`
	Revenue       *int64 `json:"revenue,omitempty"`
	GrossProfit   *int64 `json:"grossProfit,omitempty"`
	Profit        *int64 `json:"profit,omitempty"`
	Equity        *int64 `json:"equity,omitempty"`
	Assets        *int64 `json:"assets,omitempty"`
}

// fsaFields maps field names to their XBRL tag local names (FSA namespace preferred, IFRS fallback).
var fsaFields = map[string][]string{
	"revenue":      {"Revenue"},
	"gross_profit": {"GrossProfitLoss"},
	"profit":       {"ProfitLoss"},
	"equity":       {"Equity"},
	"assets":       {"Assets"},
}

const (
	fsaNS    = "http://xbrl.dcca.dk/fsa"
	ifrsNS   = "http://xbrl.ifrs.org/taxonomy/2014-03-05/ifrs-full"
	xbrliNS  = "http://www.xbrl.org/2003/instance"
	xbrldiNS = "http://xbrl.org/2006/xbrldi"
)

func (c *Client) Financials(cvr string) (*Financials, error) {
	return c.financialsForYear(cvr, 0)
}

// FinancialsByYear returns figures from the most recent AARSRAPPORT whose
// fiscal-year-end calendar year equals `year`.
func (c *Client) FinancialsByYear(cvr string, year int) (*Financials, error) {
	return c.financialsForYear(cvr, year)
}

// FinancialsAll returns figures from every AARSRAPPORT filing, newest first.
// PDF-only filings are included with PDFOnly=true and nil numeric fields.
func (c *Client) FinancialsAll(cvr string) ([]*Financials, error) {
	filings, err := c.fetchAnnualReports(cvr)
	if err != nil {
		return nil, err
	}
	var out []*Financials
	for _, f := range filings {
		fin, err := c.financialsFromFiling(cvr, f)
		if err != nil {
			return nil, err
		}
		out = append(out, fin)
	}
	return out, nil
}

func (c *Client) financialsForYear(cvr string, year int) (*Financials, error) {
	filing, err := c.pickFiling(cvr, year)
	if err != nil {
		return nil, err
	}
	return c.financialsFromFiling(cvr, filing)
}

// FinancialsRaw returns the raw filing metadata JSON and XBRL document bytes for
// a CVR's AARSRAPPORT. `year == 0` picks the most recent filing; otherwise picks
// the most recent filing whose fiscal-year-end falls in that calendar year.
func (c *Client) FinancialsRaw(cvr string, year int) (filingJSON, xbrlDoc []byte, err error) {
	filing, err := c.pickFiling(cvr, year)
	if err != nil {
		return nil, nil, err
	}
	filingJSON, err = json.MarshalIndent(filing, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if docURL := xbrlURL(filing); docURL != "" {
		xbrlDoc, err = c.download(docURL)
		if err != nil {
			return filingJSON, nil, err
		}
	}
	return filingJSON, xbrlDoc, nil
}

// FinancialsRawAll returns a JSON array of all AARSRAPPORT filing metadata for a CVR, newest first.
func (c *Client) FinancialsRawAll(cvr string) ([]byte, error) {
	filings, err := c.fetchAnnualReports(cvr)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(filings, "", "  ")
}

func (c *Client) pickFiling(cvr string, year int) (map[string]any, error) {
	filings, err := c.fetchAnnualReports(cvr)
	if err != nil {
		return nil, err
	}
	if len(filings) == 0 {
		return nil, fmt.Errorf("no annual report found for CVR %s", cvr)
	}
	if year == 0 {
		return filings[0], nil
	}
	yearStr := strconv.Itoa(year)
	for _, f := range filings {
		if strings.HasPrefix(filingFiscalYearEnd(f), yearStr) {
			return f, nil
		}
	}
	return nil, fmt.Errorf("no AARSRAPPORT found for CVR %s with fiscal year ending in %d", cvr, year)
}

func filingFiscalYearEnd(filing map[string]any) string {
	regnskab, ok := filing["regnskab"].(map[string]any)
	if !ok {
		return ""
	}
	period, ok := regnskab["regnskabsperiode"].(map[string]any)
	if !ok {
		return ""
	}
	end, _ := period["slutDato"].(string)
	return end
}

func (c *Client) financialsFromFiling(cvr string, filing map[string]any) (*Financials, error) {
	fiscalYearEnd := filingFiscalYearEnd(filing)

	docURL := xbrlURL(filing)
	if docURL == "" {
		return &Financials{
			CVR:           cvr,
			FiscalYearEnd: fiscalYearEnd,
			PDFOnly:       true,
		}, nil
	}

	content, err := c.download(docURL)
	if err != nil {
		return nil, err
	}

	fin, err := parseXBRL(content, fiscalYearEnd)
	if err != nil {
		return nil, err
	}
	fin.CVR = cvr
	fin.FiscalYearEnd = fiscalYearEnd

	return fin, nil
}

// fetchAnnualReports returns every AARSRAPPORT filing for a CVR, sorted newest-first.
// size=100 is generous — no real Danish company has published more than that.
func (c *Client) fetchAnnualReports(cvr string) ([]map[string]any, error) {
	cvrInt, err := strconv.Atoi(cvr)
	if err != nil {
		return nil, fmt.Errorf("invalid CVR number: %s", cvr)
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{"term": map[string]any{"cvrNummer": cvrInt}},
					map[string]any{"match": map[string]any{"dokumenter.dokumentType": "AARSRAPPORT"}},
				},
			},
		},
		"sort": []any{map[string]any{"offentliggoerelsesTidspunkt": map[string]any{"order": "desc"}}},
		"size": 100,
	}

	raw, err := c.postJSON(financialsBase, query)
	if err != nil {
		return nil, err
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]any `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		out = append(out, h.Source)
	}
	return out, nil
}

func (c *Client) download(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func xbrlURL(filing map[string]any) string {
	docs, ok := filing["dokumenter"].([]any)
	if !ok {
		return ""
	}
	for _, d := range docs {
		doc, ok := d.(map[string]any)
		if !ok {
			continue
		}
		mime, _ := doc["dokumentMimeType"].(string)
		if strings.Contains(mime, "xml") || strings.Contains(mime, "html") {
			if url, ok := doc["dokumentUrl"].(string); ok && url != "" {
				return url
			}
		}
	}
	return ""
}

// xbrlContext describes an xbrli:context element: the period it reports on,
// and whether it carries scenario dimensions (explicitMember / typedMember).
// Dimensioned contexts describe sub-breakdowns (e.g. equity per equity-component)
// rather than the consolidated figure, so we skip them when extracting top-line numbers.
type xbrlContext struct {
	periodEnd    string
	hasDimension bool
}

type xbrlFact struct {
	namespace  string
	localName  string
	contextRef string
	nilFlag    bool
	value      string
}

// parseXBRL extracts key financial figures from an XBRL instance document.
// Works for both plain XBRL (<xbrli:xbrl>) and iXBRL/XHTML (both are valid XML).
// fiscalYearEnd (YYYY-MM-DD) is used to pick the current-period context; pass ""
// to accept any period (first match wins).
func parseXBRL(content []byte, fiscalYearEnd string) (*Financials, error) {
	dec := xml.NewDecoder(bytes.NewReader(content))
	dec.Strict = false

	contexts := map[string]xbrlContext{}
	var facts []xbrlFact

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("XBRL parse error: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		if start.Name.Space == xbrliNS && start.Name.Local == "context" {
			id := findAttr(start, "", "id")
			ctx, err := readContext(dec)
			if err != nil {
				return nil, err
			}
			if id != "" {
				contexts[id] = ctx
			}
			continue
		}

		if start.Name.Space == fsaNS || start.Name.Space == ifrsNS {
			ctxRef := findAttr(start, "", "contextRef")
			nilAttr := findAttr(start, "http://www.w3.org/2001/XMLSchema-instance", "nil")
			var text string
			if err := dec.DecodeElement(&text, &start); err != nil {
				return nil, fmt.Errorf("read fact %s: %w", start.Name.Local, err)
			}
			facts = append(facts, xbrlFact{
				namespace:  start.Name.Space,
				localName:  start.Name.Local,
				contextRef: ctxRef,
				nilFlag:    nilAttr == "true",
				value:      strings.TrimSpace(text),
			})
		}
	}

	values := map[string]*int64{}
	for field, locals := range fsaFields {
		for _, local := range locals {
			for _, ns := range []string{fsaNS, ifrsNS} {
				if v := pickValue(facts, contexts, ns, local, fiscalYearEnd); v != nil {
					values[field] = v
					break
				}
			}
			if values[field] != nil {
				break
			}
		}
	}

	return &Financials{
		Revenue:     values["revenue"],
		GrossProfit: values["gross_profit"],
		Profit:      values["profit"],
		Equity:      values["equity"],
		Assets:      values["assets"],
	}, nil
}

func pickValue(facts []xbrlFact, contexts map[string]xbrlContext, ns, local, fiscalYearEnd string) *int64 {
	for _, f := range facts {
		if f.nilFlag || f.namespace != ns || f.localName != local {
			continue
		}
		ctx, ok := contexts[f.contextRef]
		if !ok || ctx.hasDimension {
			continue
		}
		if fiscalYearEnd != "" && ctx.periodEnd != fiscalYearEnd {
			continue
		}
		v := strings.ReplaceAll(f.value, ",", ".")
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			continue
		}
		iv := int64(n)
		return &iv
	}
	return nil
}

func readContext(dec *xml.Decoder) (xbrlContext, error) {
	var ctx xbrlContext
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return ctx, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if t.Name.Space == xbrldiNS && (t.Name.Local == "explicitMember" || t.Name.Local == "typedMember") {
				ctx.hasDimension = true
			}
			if t.Name.Space == xbrliNS && (t.Name.Local == "endDate" || t.Name.Local == "instant") {
				var text string
				if err := dec.DecodeElement(&text, &t); err != nil {
					return ctx, err
				}
				ctx.periodEnd = strings.TrimSpace(text)
				depth--
			}
		case xml.EndElement:
			depth--
		}
	}
	return ctx, nil
}

func findAttr(e xml.StartElement, space, local string) string {
	for _, a := range e.Attr {
		if a.Name.Local == local && (space == "" || a.Name.Space == space) {
			return a.Value
		}
	}
	return ""
}
