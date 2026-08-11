package scalefixture

import "testing"

func TestRoundTrip(t *testing.T) {
	profile, err := NewProfile(25000, "2026-08-11T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	profileBytes, err := MarshalProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnmarshalProfile(profileBytes)
	if err != nil || decoded != profile {
		t.Fatalf("profile round trip: decoded=%+v err=%v", decoded, err)
	}
	body, err := GenerateBody(profile)
	if err != nil {
		t.Fatal(err)
	}
	fields, err := ParseBody(body, profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 25000 || fields[0] != [2]string{"field_00000", "value_00000"} || fields[len(fields)-1] != [2]string{"field_24999", "value_24999"} {
		t.Fatalf("unexpected controlled fields: first=%v last=%v count=%d", fields[0], fields[len(fields)-1], len(fields))
	}
}

func TestRejectsMutation(t *testing.T) {
	profile, err := NewProfile(2, "2026-08-11T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range [][]byte{
		[]byte(`{"field_00000":"value_00000"}`),
		[]byte(`{"field_00000":"wrong","field_00001":"value_00001"}`),
		[]byte(`{"field_00000":"value_00000","field_00000":"value_00000"}`),
	} {
		if _, err := ParseBody(body, profile); err == nil {
			t.Fatalf("mutated body accepted: %s", body)
		}
	}
	for _, observedAt := range []string{"", "2026-08-11", "2026-08-11T00:00:00+00:00", "2026-08-11T00:00:99Z"} {
		if _, err := NewProfile(2, observedAt); err == nil {
			t.Fatalf("non-canonical observation time accepted: %q", observedAt)
		}
	}
}
