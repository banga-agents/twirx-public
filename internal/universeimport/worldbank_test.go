package universeimport

import (
	"os"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

func TestCompileWorldBankFixture(t *testing.T) {
	representation := worldBankFixture(t)
	config := worldBankTestConfig(representation)

	records, err := CompileWorldBank(representation, config)
	if err != nil {
		t.Fatalf("compile World Bank fixture: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	record := records[0]
	if record.NativeKey != "world-bank:CHL/SP.POP.TOTL/2024" {
		t.Fatalf("native key = %q", record.NativeKey)
	}
	if len(record.Packets) != 5 || len(record.Mappings) != 5 || len(record.Frame.Slots) != 5 {
		t.Fatalf("artifacts packets=%d mappings=%d slots=%d, want 5/5/5", len(record.Packets), len(record.Mappings), len(record.Frame.Slots))
	}
	if err := record.Frame.Validate(); err != nil {
		t.Fatalf("frame did not validate: %v", err)
	}
	decoded, err := dataplane.UnmarshalFrame(record.FrameCBOR)
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	if decoded.NativeKey != record.NativeKey || dataplane.DigestBytes(record.FrameCBOR) != record.FrameDigest {
		t.Fatal("frame identity or detached digest changed")
	}

	packets := packetByNativePredicate(record.Packets)
	value := packets["value"].Packet
	if value.Object.NativeLexical != "19764771" || value.Object.Typed == nil || value.Object.Typed.Lexical != "19764771.0" {
		t.Fatalf("source/typed value = %q/%v", value.Object.NativeLexical, value.Object.Typed)
	}
	if value.Source.Locator != "/1/0/value" || value.Source.RepresentationDigest != config.RepresentationDigest {
		t.Fatalf("value proof binding = %q/%x", value.Source.Locator, value.Source.RepresentationDigest)
	}
	if got := value.Context.Dimensions[0].Value.Lexical; got != "world-bank:source/2" {
		t.Fatalf("source database = %q", got)
	}
	unit := packets["unit"].Packet
	if unit.Object.NativeStatus != "not_provided" || unit.Object.NativeLexical != "" || unit.Object.Typed != nil {
		t.Fatalf("missing native unit was not kept explicit: %+v", unit.Object)
	}
	for _, mapping := range record.Mappings {
		if mapping.Claim.Status != "candidate" || mapping.Claim.Relation != "candidate" || mapping.Claim.ReviewDecisionDigest.Present {
			t.Fatalf("mapping crossed review boundary: %+v", mapping.Claim)
		}
	}
}

func TestCompileWorldBankFailsClosed(t *testing.T) {
	representation := worldBankFixture(t)
	tests := []struct {
		name   string
		body   []byte
		mutate func(*Config)
	}{
		{name: "evidence not stored", body: representation, mutate: func(c *Config) { c.EvidenceStored = false }},
		{name: "digest mismatch", body: append([]byte(nil), representation...), mutate: func(c *Config) { c.RepresentationDigest = fixedDigest("wrong") }},
		{name: "noncanonical origin", body: representation, mutate: func(c *Config) { c.OriginID = "world-bank-indicators" }},
		{name: "real evidence without policy", body: representation, mutate: func(c *Config) { c.EvidenceClass = "current_observation" }},
		{name: "fixture implying policy", body: representation, mutate: func(c *Config) {
			c.PolicyDecisionDigest = dataplane.OptionalDigest{Present: true, Value: fixedDigest("decision")}
		}},
		{name: "duplicate key", body: []byte(`[{"page":1,"page":1,"pages":1,"per_page":1,"total":0,"sourceid":"2","lastupdated":"2026-07-13"},[]]`)},
		{name: "unknown record field", body: []byte(`[{"page":1,"pages":1,"per_page":1,"total":1,"sourceid":"2","lastupdated":"2026-07-13"},[{"indicator":{"id":"X","value":"x"},"country":{"id":"CL","value":"Chile"},"countryiso3code":"CHL","date":"2024","value":1,"unit":"","obs_status":"","decimal":0,"unexpected":true}]]`)},
		{name: "exponent value", body: []byte(`[{"page":1,"pages":1,"per_page":1,"total":1,"sourceid":"2","lastupdated":"2026-07-13"},[{"indicator":{"id":"X","value":"x"},"country":{"id":"CL","value":"Chile"},"countryiso3code":"CHL","date":"2024","value":1e2,"unit":"","obs_status":"","decimal":0}]]`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := test.body
			config := worldBankTestConfig(body)
			if test.mutate != nil {
				test.mutate(&config)
			}
			if _, err := CompileWorldBank(body, config); err == nil {
				t.Fatal("accepted invalid or unauthorized input")
			}
		})
	}
}

func TestCompileWorldBankRealEvidenceRequiresNonzeroPolicyDigest(t *testing.T) {
	representation := worldBankFixture(t)
	config := worldBankTestConfig(representation)
	config.EvidenceClass = "current_observation"
	config.PolicyDecisionDigest = dataplane.OptionalDigest{Present: true}
	if _, err := CompileWorldBank(representation, config); err == nil {
		t.Fatal("accepted zero policy-decision digest")
	}
}

func FuzzCompileWorldBank(f *testing.F) {
	seed, err := os.ReadFile("../../origins/fixtures/world-bank-chl-population-2024.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(`[]`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, representation []byte) {
		config := worldBankTestConfig(representation)
		records, err := CompileWorldBank(representation, config)
		if err != nil {
			return
		}
		for _, record := range records {
			if err := record.Frame.Validate(); err != nil {
				t.Fatalf("accepted input produced invalid frame: %v", err)
			}
		}
	})
}

func worldBankFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../origins/fixtures/world-bank-chl-population-2024.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func worldBankTestConfig(representation []byte) Config {
	return Config{
		OriginID:             WorldBankOriginID,
		ObservedAt:           "2026-08-11T00:00:00Z",
		RepresentationDigest: dataplane.DigestBytes(representation),
		ObservationDigest:    fixedDigest("observation"),
		ModuleSetDigest:      fixedDigest("module-set"),
		EvidenceClass:        "test_fixture",
		EvidenceRef:          "origins/fixtures/world-bank-chl-population-2024.json",
		EvidenceStored:       true,
	}
}

func TestSourceNumberToDecimal(t *testing.T) {
	for _, value := range []string{"1e3", "1E3", "1.", "1.2.3"} {
		if _, err := sourceNumberToDecimal(value, 0); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	if got, err := sourceNumberToDecimal("-12", 0); err != nil || got != "-12.0" {
		t.Fatalf("integer conversion = %q, %v", got, err)
	}
	if got, err := sourceNumberToDecimal("12.50", 2); err != nil || !strings.EqualFold(got, "12.50") {
		t.Fatalf("decimal preservation = %q, %v", got, err)
	}
}
