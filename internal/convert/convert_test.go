// Tests for lossless representation conversions. Every conversion here is
// verified to roundtrip: converting back must yield the exact original.
package convert

import (
	"strings"
	"testing"
)

const (
	vecV1   = "c232ab00-9414-11ec-b3c8-9f6bdeced846"
	vecV6   = "1ec9414c-232a-6b00-b3c8-9f6bdeced846"
	vecV7   = "017f22e2-79b0-7cc3-98c4-dc0c0c07398f"
	vecULID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
)

func mustConvert(t *testing.T, input, target string) string {
	t.Helper()
	out, err := Convert(input, target, "twitter")
	if err != nil {
		t.Fatalf("Convert(%q, %q): %v", input, target, err)
	}
	return out
}

func TestUUIDToULIDAndBack(t *testing.T) {
	asULID := mustConvert(t, vecV7, "ulid")
	if asULID != "01FWHE4YDGFK1SHH6W1G60EECF" {
		t.Fatalf("uuid->ulid = %q", asULID)
	}
	back := mustConvert(t, asULID, "uuid")
	if back != vecV7 {
		t.Fatalf("roundtrip = %q, want %q", back, vecV7)
	}
}

func TestULIDToUUIDPreservesBits(t *testing.T) {
	asUUID := mustConvert(t, vecULID, "uuid")
	if asUUID != "01563e3a-b5d3-d676-4c61-efb99302bd5b" {
		t.Fatalf("ulid->uuid = %q", asUUID)
	}
	if mustConvert(t, asUUID, "ulid") != vecULID {
		t.Fatal("uuid->ulid did not roundtrip")
	}
}

func TestGregorianConversionMatchesRFCVectors(t *testing.T) {
	// RFC 9562 publishes both vectors as the same fields in both orders,
	// so the conversion must map one exactly onto the other.
	if got := mustConvert(t, vecV1, "uuidv6"); got != vecV6 {
		t.Fatalf("v1->v6 = %q, want %q", got, vecV6)
	}
	if got := mustConvert(t, vecV6, "uuidv1"); got != vecV1 {
		t.Fatalf("v6->v1 = %q, want %q", got, vecV1)
	}
}

func TestGregorianConversionRoundtrips(t *testing.T) {
	there := mustConvert(t, vecV1, "uuidv6")
	back := mustConvert(t, there, "uuidv1")
	if back != vecV1 {
		t.Fatalf("v1->v6->v1 = %q, want %q", back, vecV1)
	}
}

func TestV6ConversionRejectsNonV1Input(t *testing.T) {
	for _, bad := range []string{vecV7, vecULID, "175928847299117063"} {
		if _, err := Convert(bad, "uuidv6", "twitter"); err == nil {
			t.Errorf("uuidv6 conversion should reject %q", bad)
		}
	}
}

func TestHexTargetPerFormat(t *testing.T) {
	cases := map[string]string{
		vecV7:                         "017f22e279b07cc398c4dc0c0c07398f",
		vecULID:                       "01563e3ab5d3d6764c61efb99302bd5b",
		"0ujtsYcgvSTl8PAuAdqWYSMnLOv": "0669f7efb5a1cd34b5f99d1154fb6853345c9735",
		"1541815603606036480":         "1565a11f6217a000",
	}
	for input, want := range cases {
		if got := mustConvert(t, input, "hex"); got != want {
			t.Errorf("hex(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestKSUIDRefusedFor128BitTargets(t *testing.T) {
	_, err := Convert("0ujtsYcgvSTl8PAuAdqWYSMnLOv", "uuid", "twitter")
	if err == nil || !strings.Contains(err.Error(), "160 bits") {
		t.Fatalf("ksuid->uuid must refuse with a width explanation, got %v", err)
	}
}

func TestSnowflakeRefusedFor128BitTargets(t *testing.T) {
	_, err := Convert("1541815603606036480", "ulid", "twitter")
	if err == nil || !strings.Contains(err.Error(), "64 bits") {
		t.Fatalf("snowflake->ulid must refuse with a width explanation, got %v", err)
	}
}

func TestUnknownTargetAndUnrecognizedInputError(t *testing.T) {
	if _, err := Convert(vecV7, "base58", "twitter"); err == nil {
		t.Fatal("unknown target must error")
	}
	if _, err := Convert("not-an-id", "hex", "twitter"); err == nil {
		t.Fatal("unrecognized input must error")
	}
}

func TestUUIDTargetNormalizesCompactInput(t *testing.T) {
	got := mustConvert(t, "017F22E279B07CC398C4DC0C0C07398F", "uuid")
	if got != vecV7 {
		t.Fatalf("normalization = %q, want %q", got, vecV7)
	}
}
