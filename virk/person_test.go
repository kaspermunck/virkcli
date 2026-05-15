package virk

import "testing"

func TestFunktionAndEnd(t *testing.T) {
	ended := "2023-06-30"
	tests := []struct {
		name       string
		md         []medlemsData
		wantFunk   string
		wantEnded  string
	}{
		{"empty", nil, "", ""},
		{"active role", []medlemsData{{
			Attributter: []deltagerAttribut{{
				Type: "FUNKTION",
				Vaerdier: []attributVaerdi{{Vaerdi: "DIREKTØR"}},
			}},
		}}, "DIREKTØR", ""},
		{"ended role", []medlemsData{{
			Attributter: []deltagerAttribut{{
				Type: "FUNKTION",
				Vaerdier: []attributVaerdi{{
					Vaerdi: "FORMAND",
					Periode: struct {
						GyldigFra string  `json:"gyldigFra"`
						GyldigTil *string `json:"gyldigTil"`
					}{GyldigTil: &ended},
				}},
			}},
		}}, "FORMAND", "2023-06-30"},
		{"ignores non-FUNKTION", []medlemsData{{
			Attributter: []deltagerAttribut{{
				Type:     "EJERANDEL_PROCENT",
				Vaerdier: []attributVaerdi{{Vaerdi: "0.5"}},
			}},
		}}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funk, end := funktionAndEnd(tt.md)
			if funk != tt.wantFunk {
				t.Errorf("funktion = %q, want %q", funk, tt.wantFunk)
			}
			if end != tt.wantEnded {
				t.Errorf("endedAt = %q, want %q", end, tt.wantEnded)
			}
		})
	}
}

func TestParsePersonHits(t *testing.T) {
	raw := []byte(`{
		"hits": {"hits": [
			{"_score": 10.5, "_source": {"Vrdeltagerperson": {
				"enhedsNummer": 4004083032,
				"navne": [{"navn": "Ken Klausen"}],
				"virksomhedSummariskRelation": [{}, {}, {}]
			}}},
			{"_score": 5.0, "_source": {"Vrdeltagerperson": {
				"enhedsNummer": 4009165889,
				"navne": [{"navn": "Anders And"}],
				"virksomhedSummariskRelation": [{}]
			}}}
		]}
	}`)
	hits, err := parsePersonHits(raw)
	if err != nil {
		t.Fatalf("parsePersonHits: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("got %d hits, want 2", len(hits))
	}
	if hits[0].EnhedsNummer != 4004083032 {
		t.Errorf("hits[0].EnhedsNummer = %d", hits[0].EnhedsNummer)
	}
	if hits[0].Name != "Ken Klausen" {
		t.Errorf("hits[0].Name = %q", hits[0].Name)
	}
	if hits[0].RelationCount != 3 {
		t.Errorf("hits[0].RelationCount = %d", hits[0].RelationCount)
	}
	if hits[0].Score != 10.5 {
		t.Errorf("hits[0].Score = %f", hits[0].Score)
	}
}

func TestParsePersonDetail(t *testing.T) {
	raw := []byte(`{
		"hits": {"hits": [{
			"_source": {"Vrdeltagerperson": {
				"enhedsNummer": 4004083032,
				"navne": [{"navn": "Ken Klausen"}],
				"adresseHemmelig": true,
				"virksomhedSummariskRelation": [{
					"virksomhed": {
						"cvrNummer": 36930144,
						"navne": [{"navn": "LWOH ApS"}]
					},
					"organisationer": [{
						"hovedtype": "LEDELSESORGAN",
						"medlemsData": [{"attributter": [{
							"type": "FUNKTION",
							"vaerdier": [{"vaerdi": "DIREKTØR"}]
						}]}]
					}]
				}]
			}}
		}]}
	}`)
	p, err := parsePersonDetail(raw, false)
	if err != nil {
		t.Fatalf("parsePersonDetail: %v", err)
	}
	if p.EnhedsNummer != 4004083032 {
		t.Errorf("EnhedsNummer = %d", p.EnhedsNummer)
	}
	if !p.AddressHidden {
		t.Error("AddressHidden should be true")
	}
	if len(p.Relations) != 1 {
		t.Fatalf("got %d relations, want 1", len(p.Relations))
	}
	r := p.Relations[0]
	if r.CVR != "36930144" {
		t.Errorf("CVR = %q", r.CVR)
	}
	if r.Role != "Direktør" {
		t.Errorf("Role = %q", r.Role)
	}
	if !r.Active {
		t.Error("Active should be true")
	}
}

func TestParsePersonDetailActiveOnly(t *testing.T) {
	ended := "2023-01-01"
	_ = ended
	raw := []byte(`{
		"hits": {"hits": [{
			"_source": {"Vrdeltagerperson": {
				"enhedsNummer": 100,
				"navne": [{"navn": "Test"}],
				"virksomhedSummariskRelation": [{
					"virksomhed": {"cvrNummer": 1, "navne": [{"navn": "A"}]},
					"organisationer": [{
						"hovedtype": "LEDELSESORGAN",
						"medlemsData": [{"attributter": [{
							"type": "FUNKTION",
							"vaerdier": [{"vaerdi": "DIREKTØR", "periode": {"gyldigTil": "2023-01-01"}}]
						}]}]
					}]
				}, {
					"virksomhed": {"cvrNummer": 2, "navne": [{"navn": "B"}]},
					"organisationer": [{
						"hovedtype": "LEDELSESORGAN",
						"medlemsData": [{"attributter": [{
							"type": "FUNKTION",
							"vaerdier": [{"vaerdi": "FORMAND"}]
						}]}]
					}]
				}]
			}}
		}]}
	}`)

	all, err := parsePersonDetail(raw, false)
	if err != nil {
		t.Fatalf("parsePersonDetail(all): %v", err)
	}
	if len(all.Relations) != 2 {
		t.Fatalf("all: got %d relations, want 2", len(all.Relations))
	}

	active, err := parsePersonDetail(raw, true)
	if err != nil {
		t.Fatalf("parsePersonDetail(active): %v", err)
	}
	if len(active.Relations) != 1 {
		t.Fatalf("active: got %d relations, want 1", len(active.Relations))
	}
	if active.Relations[0].Company != "B" {
		t.Errorf("active relation company = %q, want B", active.Relations[0].Company)
	}
}
