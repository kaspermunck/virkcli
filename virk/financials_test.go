package virk

import "testing"

func TestParseXBRL(t *testing.T) {
	xbrl := []byte(`<?xml version="1.0"?>
<xbrli:xbrl xmlns:xbrli="http://www.xbrl.org/2003/instance"
            xmlns:fsa="http://xbrl.dcca.dk/fsa"
            xmlns:xbrldi="http://xbrl.org/2006/xbrldi"
            xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">

  <xbrli:context id="ctx_current">
    <xbrli:entity><xbrli:identifier>1234</xbrli:identifier></xbrli:entity>
    <xbrli:period><xbrli:endDate>2024-12-31</xbrli:endDate></xbrli:period>
  </xbrli:context>

  <xbrli:context id="ctx_dim">
    <xbrli:entity><xbrli:identifier>1234</xbrli:identifier></xbrli:entity>
    <xbrli:period><xbrli:endDate>2024-12-31</xbrli:endDate></xbrli:period>
    <xbrli:scenario><xbrldi:explicitMember dimension="d:EquityDim">d:ShareCapital</xbrldi:explicitMember></xbrli:scenario>
  </xbrli:context>

  <xbrli:context id="ctx_old">
    <xbrli:entity><xbrli:identifier>1234</xbrli:identifier></xbrli:entity>
    <xbrli:period><xbrli:endDate>2023-12-31</xbrli:endDate></xbrli:period>
  </xbrli:context>

  <fsa:Revenue contextRef="ctx_current">1000000</fsa:Revenue>
  <fsa:GrossProfitLoss contextRef="ctx_current">500000</fsa:GrossProfitLoss>
  <fsa:ProfitLoss contextRef="ctx_current">200000</fsa:ProfitLoss>
  <fsa:Equity contextRef="ctx_current">3000000</fsa:Equity>
  <fsa:Equity contextRef="ctx_dim">1500000</fsa:Equity>
  <fsa:Assets contextRef="ctx_current">5000000</fsa:Assets>

  <!-- old period, should be ignored when fiscalYearEnd is specified -->
  <fsa:Revenue contextRef="ctx_old">900000</fsa:Revenue>

</xbrli:xbrl>`)

	fin, err := parseXBRL(xbrl, "2024-12-31")
	if err != nil {
		t.Fatalf("parseXBRL: %v", err)
	}
	assertInt64(t, "Revenue", fin.Revenue, 1000000)
	assertInt64(t, "GrossProfit", fin.GrossProfit, 500000)
	assertInt64(t, "Profit", fin.Profit, 200000)
	assertInt64(t, "Equity", fin.Equity, 3000000)
	assertInt64(t, "Assets", fin.Assets, 5000000)
}

func TestParseXBRLNilFacts(t *testing.T) {
	xbrl := []byte(`<?xml version="1.0"?>
<xbrli:xbrl xmlns:xbrli="http://www.xbrl.org/2003/instance"
            xmlns:fsa="http://xbrl.dcca.dk/fsa"
            xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <xbrli:context id="ctx">
    <xbrli:entity><xbrli:identifier>1234</xbrli:identifier></xbrli:entity>
    <xbrli:period><xbrli:endDate>2024-12-31</xbrli:endDate></xbrli:period>
  </xbrli:context>
  <fsa:Revenue contextRef="ctx" xsi:nil="true"/>
  <fsa:Equity contextRef="ctx">100</fsa:Equity>
</xbrli:xbrl>`)

	fin, err := parseXBRL(xbrl, "2024-12-31")
	if err != nil {
		t.Fatalf("parseXBRL: %v", err)
	}
	if fin.Revenue != nil {
		t.Error("Revenue should be nil for xsi:nil fact")
	}
	assertInt64(t, "Equity", fin.Equity, 100)
}

func TestParseXBRLNoFiscalYearFilter(t *testing.T) {
	xbrl := []byte(`<?xml version="1.0"?>
<xbrli:xbrl xmlns:xbrli="http://www.xbrl.org/2003/instance"
            xmlns:fsa="http://xbrl.dcca.dk/fsa">
  <xbrli:context id="ctx">
    <xbrli:entity><xbrli:identifier>1234</xbrli:identifier></xbrli:entity>
    <xbrli:period><xbrli:endDate>2024-12-31</xbrli:endDate></xbrli:period>
  </xbrli:context>
  <fsa:ProfitLoss contextRef="ctx">42</fsa:ProfitLoss>
</xbrli:xbrl>`)

	fin, err := parseXBRL(xbrl, "")
	if err != nil {
		t.Fatalf("parseXBRL: %v", err)
	}
	assertInt64(t, "Profit", fin.Profit, 42)
}

func assertInt64(t *testing.T, label string, got *int64, want int64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: got nil, want %d", label, want)
		return
	}
	if *got != want {
		t.Errorf("%s: got %d, want %d", label, *got, want)
	}
}
