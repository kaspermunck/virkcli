package virk

import "testing"

func TestLatestName(t *testing.T) {
	tests := []struct {
		name  string
		navne []periodedString
		want  string
	}{
		{"empty", nil, ""},
		{"single active", []periodedString{{Navn: "Foo"}}, "Foo"},
		{"ended then active", []periodedString{
			{Navn: "Old", Periode: struct {
				GyldigFra string  `json:"gyldigFra"`
				GyldigTil *string `json:"gyldigTil"`
			}{GyldigTil: ptr("2020-01-01")}},
			{Navn: "Current"},
		}, "Current"},
		{"all ended picks last", []periodedString{
			{Navn: "A", Periode: struct {
				GyldigFra string  `json:"gyldigFra"`
				GyldigTil *string `json:"gyldigTil"`
			}{GyldigTil: ptr("2019-01-01")}},
			{Navn: "B", Periode: struct {
				GyldigFra string  `json:"gyldigFra"`
				GyldigTil *string `json:"gyldigTil"`
			}{GyldigTil: ptr("2020-01-01")}},
		}, "B"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := latestName(tt.navne)
			if got != tt.want {
				t.Errorf("latestName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTitleCase(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"DIREKTØR", "Direktør"},
		{"bestyrelsesmedlem", "Bestyrelsesmedlem"},
		{"Formand", "Formand"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := titleCase(tt.in); got != tt.want {
				t.Errorf("titleCase(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestOwnerRole(t *testing.T) {
	tests := []struct {
		hovedtype, funktion, want string
	}{
		{"REGISTER", "", "Reel ejer"},
		{"STIFTERE", "", "Stifter"},
		{"REVISION", "", "Revisor"},
		{"LEDELSESORGAN", "", "Ledelse"},
		{"LEDELSESORGAN", "DIREKTØR", "Direktør"},
		{"LEDELSESORGAN", "FORMAND", "Formand"},
		{"OTHER", "NOGET", "Noget"},
		{"OTHER", "", "OTHER"},
	}
	for _, tt := range tests {
		t.Run(tt.hovedtype+"_"+tt.funktion, func(t *testing.T) {
			if got := ownerRole(tt.hovedtype, tt.funktion); got != tt.want {
				t.Errorf("ownerRole(%q, %q) = %q, want %q", tt.hovedtype, tt.funktion, got, tt.want)
			}
		})
	}
}

func TestFormatOwnership(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"0.9", "90"},
		{"1", "100"},
		{"0.5", "50"},
		{"invalid", "invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := formatOwnership(tt.in); got != tt.want {
				t.Errorf("formatOwnership(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestClassifyContacts(t *testing.T) {
	email, phone, website := classifyContacts([]string{
		"info@example.com",
		"+45 12345678",
		"http://example.com",
		"",
		"extra@example.com",
	})
	if email != "info@example.com" {
		t.Errorf("email = %q", email)
	}
	if phone != "+45 12345678" {
		t.Errorf("phone = %q", phone)
	}
	if website != "http://example.com" {
		t.Errorf("website = %q", website)
	}
}

func TestBuildAddress(t *testing.T) {
	a := buildAddress(beliggenhedsadresse{
		Vejnavn:      "Testvej",
		HusnummerFra: 42,
		BogstavFra:   "A",
		Etage:        "3",
		Sidedoer:     "tv",
		Postnummer:   2100,
		Postdistrikt: "København Ø",
		Landekode:    "DK",
		Kommune:      kommune{KommuneNavn: "KØBENHAVN"},
	})
	if a.Street != "Testvej 42A" {
		t.Errorf("Street = %q", a.Street)
	}
	if a.Floor != "3" {
		t.Errorf("Floor = %q", a.Floor)
	}
	if a.Door != "tv" {
		t.Errorf("Door = %q", a.Door)
	}
	if a.Postcode != "2100" {
		t.Errorf("Postcode = %q", a.Postcode)
	}
	if a.City != "København Ø" {
		t.Errorf("City = %q", a.City)
	}
	if a.Municipality != "KØBENHAVN" {
		t.Errorf("Municipality = %q", a.Municipality)
	}
}

func TestParseCompany(t *testing.T) {
	raw := []byte(`{
		"hits": {"hits": [{
			"_source": {
				"Vrvirksomhed": {
					"virksomhedMetadata": {
						"nyesteNavn": {"navn": "Test A/S"},
						"nyesteBinavne": [],
						"nyesteVirksomhedsform": {"kortBeskrivelse": "A/S", "langBeskrivelse": "Aktieselskab"},
						"nyesteBeliggenhedsadresse": {"vejnavn": "Gade", "husnummerFra": 1, "postnummer": 1000, "postdistrikt": "KBH"},
						"nyesteHovedbranche": {"branchekode": "620100", "branchetekst": "IT"},
						"nyesteKontaktoplysninger": ["test@test.dk"],
						"sammensatStatus": "NORMAL",
						"stiftelsesDato": "2020-01-01",
						"antalPenheder": 2
					},
					"deltagerRelation": []
				}
			}
		}]}
	}`)
	c, err := parseCompany("12345678", raw)
	if err != nil {
		t.Fatalf("parseCompany: %v", err)
	}
	if c.Name != "Test A/S" {
		t.Errorf("Name = %q", c.Name)
	}
	if c.FormCode != "A/S" {
		t.Errorf("FormCode = %q", c.FormCode)
	}
	if c.Status != "NORMAL" {
		t.Errorf("Status = %q", c.Status)
	}
	if c.PUnitCount != 2 {
		t.Errorf("PUnitCount = %d", c.PUnitCount)
	}
	if c.Email != "test@test.dk" {
		t.Errorf("Email = %q", c.Email)
	}
}

func ptr(s string) *string { return &s }
