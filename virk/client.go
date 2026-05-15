package virk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const cvrBase = "http://distribution.virk.dk/cvr-permanent/virksomhed/_search"

type Client struct {
	http     *http.Client
	username string
	password string
}

func NewClientFromEnv() (*Client, error) {
	u := os.Getenv("VIRK_USERNAME")
	p := os.Getenv("VIRK_PASSWORD")
	// Fall back to macOS Keychain (service "virkcli", accounts "VIRK_USERNAME"/"VIRK_PASSWORD").
	// Env vars win when set.
	if u == "" {
		if v, err := keychainLookup("virkcli", "VIRK_USERNAME"); err == nil {
			u = v
		}
	}
	if p == "" {
		if v, err := keychainLookup("virkcli", "VIRK_PASSWORD"); err == nil {
			p = v
		}
	}
	if u == "" || p == "" {
		return nil, fmt.Errorf("VIRK_USERNAME and VIRK_PASSWORD must be set (env vars or macOS keychain service 'virkcli')")
	}
	return &Client{http: &http.Client{}, username: u, password: p}, nil
}

// keychainLookup retrieves a generic-password entry from the user's login keychain
// via the `security` CLI. Used as a fallback when the corresponding env var is unset.
func keychainLookup(service, account string) (string, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", service, "-a", account, "-w").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// postJSON POSTs a JSON query to an Elasticsearch endpoint and returns the raw body.
func (c *Client) postJSON(url string, query map[string]any) ([]byte, error) {
	body, _ := json.Marshal(query)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("VIRK API returned HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

type Company struct {
	CVR        string   `json:"cvr"`
	Name       string   `json:"name"`
	Aliases    []string `json:"aliases,omitempty"`
	Form       string   `json:"form,omitempty"`
	FormCode   string   `json:"formCode,omitempty"`
	Status     string   `json:"status,omitempty"`
	Founded    string   `json:"founded,omitempty"`
	Address    Address  `json:"address"`
	Industry   Industry `json:"industry,omitempty"`
	Employees  string   `json:"employees,omitempty"`
	Email      string   `json:"email,omitempty"`
	Phone      string   `json:"phone,omitempty"`
	Website    string   `json:"website,omitempty"`
	PUnitCount int      `json:"pUnitCount"`
	Owners     []Owner  `json:"owners,omitempty"`
}

type Address struct {
	Street       string `json:"street,omitempty"`
	Floor        string `json:"floor,omitempty"`
	Door         string `json:"door,omitempty"`
	Postcode     string `json:"postcode,omitempty"`
	City         string `json:"city,omitempty"`
	Municipality string `json:"municipality,omitempty"`
	Country      string `json:"country,omitempty"`
}

type Industry struct {
	Code string `json:"code,omitempty"`
	Text string `json:"text,omitempty"`
}

type Owner struct {
	Name         string `json:"name"`
	Role         string `json:"role,omitempty"`
	OwnershipPct string `json:"ownershipPct,omitempty"`
}

func (c *Client) Lookup(cvr string) (*Company, error) {
	raw, err := c.LookupRaw(cvr)
	if err != nil {
		return nil, err
	}
	return parseCompany(cvr, raw)
}

// LookupRaw returns the raw Elasticsearch response body for a CVR lookup.
func (c *Client) LookupRaw(cvr string) ([]byte, error) {
	cvrInt, err := strconv.Atoi(cvr)
	if err != nil {
		return nil, fmt.Errorf("invalid CVR number: %s", cvr)
	}

	query := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"Vrvirksomhed.cvrNummer": cvrInt,
			},
		},
		"size": 1,
	}

	return c.postJSON(cvrBase, query)
}

type periodedString struct {
	Navn    string `json:"navn"`
	Vaerdi  string `json:"vaerdi"`
	Periode struct {
		GyldigFra string  `json:"gyldigFra"`
		GyldigTil *string `json:"gyldigTil"`
	} `json:"periode"`
}

type kommune struct {
	KommuneKode int    `json:"kommuneKode"`
	KommuneNavn string `json:"kommuneNavn"`
}

type beliggenhedsadresse struct {
	Landekode    string  `json:"landekode"`
	Vejnavn      string  `json:"vejnavn"`
	HusnummerFra int     `json:"husnummerFra"`
	BogstavFra   string  `json:"bogstavFra"`
	Etage        string  `json:"etage"`
	Sidedoer     string  `json:"sidedoer"`
	Postnummer   int     `json:"postnummer"`
	Postdistrikt string  `json:"postdistrikt"`
	Kommune      kommune `json:"kommune"`
}

type virksomhedsform struct {
	KortBeskrivelse string `json:"kortBeskrivelse"`
	LangBeskrivelse string `json:"langBeskrivelse"`
}

type hovedbranche struct {
	Branchekode      string `json:"branchekode"`
	Branchetekst     string `json:"branchetekst"`
	BranchetekstLang string `json:"branchetekstLang"`
}

type virksomhedMetadata struct {
	NyesteNavn                periodedString      `json:"nyesteNavn"`
	NyesteBinavne             []json.RawMessage   `json:"nyesteBinavne"`
	NyesteVirksomhedsform     virksomhedsform     `json:"nyesteVirksomhedsform"`
	NyesteBeliggenhedsadresse beliggenhedsadresse `json:"nyesteBeliggenhedsadresse"`
	NyesteHovedbranche        hovedbranche        `json:"nyesteHovedbranche"`
	NyesteKontaktoplysninger  []string            `json:"nyesteKontaktoplysninger"`
	NyesteAarsbeskaeftigelse  *struct {
		IntervalKodeAntalAnsatte string `json:"intervalKodeAntalAnsatte"`
	} `json:"nyesteAarsbeskaeftigelse"`
	AntalPenheder  int    `json:"antalPenheder"`
	SammensatStatus string `json:"sammensatStatus"`
	StiftelsesDato string `json:"stiftelsesDato"`
}

type attributVaerdi struct {
	Vaerdi  string `json:"vaerdi"`
	Periode struct {
		GyldigFra string  `json:"gyldigFra"`
		GyldigTil *string `json:"gyldigTil"`
	} `json:"periode"`
}

type deltagerAttribut struct {
	Type     string           `json:"type"`
	Vaerdier []attributVaerdi `json:"vaerdier"`
}

type medlemsData struct {
	Attributter []deltagerAttribut `json:"attributter"`
}

type deltagerOrganisation struct {
	Hovedtype   string        `json:"hovedtype"`
	MedlemsData []medlemsData `json:"medlemsData"`
}

type deltager struct {
	Navne []periodedString `json:"navne"`
}

type deltagerRelation struct {
	Deltager       deltager               `json:"deltager"`
	Organisationer []deltagerOrganisation `json:"organisationer"`
}

type vrvirksomhed struct {
	Metadata         virksomhedMetadata `json:"virksomhedMetadata"`
	DeltagerRelation []deltagerRelation `json:"deltagerRelation"`
}

type lookupResponse struct {
	Hits struct {
		Hits []struct {
			Source struct {
				Vrvirksomhed vrvirksomhed `json:"Vrvirksomhed"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func parseCompany(cvr string, raw []byte) (*Company, error) {
	var result lookupResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(result.Hits.Hits) == 0 {
		return nil, fmt.Errorf("company not found: %s", cvr)
	}

	src := result.Hits.Hits[0].Source.Vrvirksomhed
	meta := src.Metadata

	industryText := meta.NyesteHovedbranche.BranchetekstLang
	if industryText == "" {
		industryText = meta.NyesteHovedbranche.Branchetekst
	}

	company := &Company{
		CVR:        cvr,
		Name:       meta.NyesteNavn.Navn,
		Aliases:    collectAliases(meta.NyesteBinavne),
		Form:       meta.NyesteVirksomhedsform.LangBeskrivelse,
		FormCode:   meta.NyesteVirksomhedsform.KortBeskrivelse,
		Status:     meta.SammensatStatus,
		Founded:    meta.StiftelsesDato,
		Address:    buildAddress(meta.NyesteBeliggenhedsadresse),
		Industry:   Industry{Code: meta.NyesteHovedbranche.Branchekode, Text: industryText},
		PUnitCount: meta.AntalPenheder,
	}
	if meta.NyesteAarsbeskaeftigelse != nil {
		company.Employees = meta.NyesteAarsbeskaeftigelse.IntervalKodeAntalAnsatte
	}
	company.Email, company.Phone, company.Website = classifyContacts(meta.NyesteKontaktoplysninger)
	company.Owners = buildOwners(src.DeltagerRelation)
	return company, nil
}

// collectAliases extracts current secondary names. VIRK reports nyesteBinavne
// as either ["Foo A/S", ...] or [{navn, periode: {gyldigTil}}, ...] depending on CVR.
func collectAliases(items []json.RawMessage) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		var obj periodedString
		if err := json.Unmarshal(item, &obj); err == nil && obj.Navn != "" {
			if obj.Periode.GyldigTil != nil {
				continue
			}
			out = append(out, obj.Navn)
			continue
		}
		var s string
		if err := json.Unmarshal(item, &s); err == nil && s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildAddress(a beliggenhedsadresse) Address {
	street := a.Vejnavn
	if a.HusnummerFra != 0 {
		street = strings.TrimSpace(fmt.Sprintf("%s %d%s", a.Vejnavn, a.HusnummerFra, a.BogstavFra))
	}
	addr := Address{
		Street:       street,
		Floor:        a.Etage,
		Door:         a.Sidedoer,
		City:         a.Postdistrikt,
		Municipality: a.Kommune.KommuneNavn,
		Country:      a.Landekode,
	}
	if a.Postnummer != 0 {
		addr.Postcode = strconv.Itoa(a.Postnummer)
	}
	return addr
}

var phoneOnly = regexp.MustCompile(`^\+?\d[\d\s-]{4,}$`)

func classifyContacts(contacts []string) (email, phone, website string) {
	for _, c := range contacts {
		c = strings.TrimSpace(c)
		switch {
		case c == "":
			continue
		case strings.Contains(c, "@"):
			if email == "" {
				email = c
			}
		case strings.HasPrefix(c, "http://") || strings.HasPrefix(c, "https://") || strings.HasPrefix(c, "www."):
			if website == "" {
				website = c
			}
		case phoneOnly.MatchString(c):
			if phone == "" {
				phone = c
			}
		}
	}
	return
}

func buildOwners(relations []deltagerRelation) []Owner {
	var owners []Owner
	for _, rel := range relations {
		name := latestName(rel.Deltager.Navne)
		for _, org := range rel.Organisationer {
			funktion, ejerandel := orgAttrs(org.MedlemsData)
			role := ownerRole(org.Hovedtype, funktion)
			owners = append(owners, Owner{
				Name:         name,
				Role:         role,
				OwnershipPct: formatOwnership(ejerandel),
			})
		}
	}
	return owners
}

func latestName(navne []periodedString) string {
	for i := len(navne) - 1; i >= 0; i-- {
		if navne[i].Periode.GyldigTil == nil && navne[i].Navn != "" {
			return navne[i].Navn
		}
	}
	if len(navne) > 0 {
		return navne[len(navne)-1].Navn
	}
	return ""
}

func orgAttrs(md []medlemsData) (funktion, ejerandel string) {
	for _, m := range md {
		for _, a := range m.Attributter {
			if len(a.Vaerdier) == 0 {
				continue
			}
			v := a.Vaerdier[len(a.Vaerdier)-1].Vaerdi
			switch a.Type {
			case "FUNKTION":
				funktion = v
			case "EJERANDEL_PROCENT":
				ejerandel = v
			}
		}
	}
	return
}

// ownerRole maps VIRK's hovedtype + funktion codes to a human-readable role.
// hovedtype categorises the relation; funktion refines it for management/board members.
func ownerRole(hovedtype, funktion string) string {
	if funktion != "" && hovedtype == "LEDELSESORGAN" {
		return titleCase(funktion)
	}
	switch hovedtype {
	case "REGISTER":
		return "Reel ejer"
	case "STIFTERE":
		return "Stifter"
	case "REVISION":
		return "Revisor"
	case "LEDELSESORGAN":
		return "Ledelse"
	}
	if funktion != "" {
		return titleCase(funktion)
	}
	return hovedtype
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)
	return strings.ToUpper(lower[:1]) + lower[1:]
}

// formatOwnership renders VIRK's fractional ownership (e.g. "0.9" → "90") as a percentage.
func formatOwnership(v string) string {
	if v == "" {
		return ""
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return v
	}
	pct := f * 100
	if pct == float64(int64(pct)) {
		return strconv.FormatInt(int64(pct), 10)
	}
	return strconv.FormatFloat(pct, 'f', -1, 64)
}
