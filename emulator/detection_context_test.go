package emulator

import "testing"

// Round-trip: a V1 context stamps byte 0 = 1 and ParseDetectionContext recovers
// the exact payload.
func TestDetectionContext_V1RoundTrip(t *testing.T) {
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	dc := NewDetectionContextV1(payload)

	if len(dc) != len(payload)+1 {
		t.Fatalf("expected leading version byte: len(dc)=%d, want %d", len(dc), len(payload)+1)
	}
	if dc[0] != DetectionContextSchemaV1 {
		t.Fatalf("leading byte = %d, want V1=%d", dc[0], DetectionContextSchemaV1)
	}

	ver, got := ParseDetectionContext(dc)
	if ver != DetectionContextSchemaV1 {
		t.Fatalf("parsed version = %d, want V1=%d", ver, DetectionContextSchemaV1)
	}
	if string(got) != string(payload) {
		t.Fatalf("parsed payload = %v, want %v", got, payload)
	}
}

// An empty/nil DetectionContext is the legacy sentinel, never a valid versioned
// context — so an unpopulated field can't be mistaken for schema data.
func TestDetectionContext_EmptyIsLegacy(t *testing.T) {
	for _, dc := range [][]byte{nil, {}} {
		ver, payload := ParseDetectionContext(dc)
		if ver != DetectionContextSchemaLegacy {
			t.Fatalf("empty ctx version = %d, want legacy=%d", ver, DetectionContextSchemaLegacy)
		}
		if payload != nil {
			t.Fatalf("empty ctx payload = %v, want nil", payload)
		}
	}
}

// A NON-empty context whose leading byte is 0 still reports the legacy sentinel
// (never a versioned schema), and its trailing bytes are recovered verbatim as
// payload — so a stray/pre-#42 buffer that happens to carry data is dispatched
// as "legacy, do not parse" rather than silently misread as schema-0 data.
func TestDetectionContext_LeadingZeroNonEmptyIsLegacy(t *testing.T) {
	dc := []byte{DetectionContextSchemaLegacy, 0xAA, 0xBB}
	ver, payload := ParseDetectionContext(dc)
	if ver != DetectionContextSchemaLegacy {
		t.Fatalf("leading-zero ctx version = %d, want legacy=%d", ver, DetectionContextSchemaLegacy)
	}
	if string(payload) != string([]byte{0xAA, 0xBB}) {
		t.Fatalf("parsed payload = %v, want [170 187]", payload)
	}
}

// A payload is allowed to be empty: the version byte alone is a valid V1 context
// (version present, zero-length payload) — distinct from the legacy sentinel.
func TestDetectionContext_V1EmptyPayload(t *testing.T) {
	dc := NewDetectionContextV1(nil)
	if len(dc) != 1 || dc[0] != DetectionContextSchemaV1 {
		t.Fatalf("V1 empty-payload ctx = %v, want [%d]", dc, DetectionContextSchemaV1)
	}
	ver, payload := ParseDetectionContext(dc)
	if ver != DetectionContextSchemaV1 || len(payload) != 0 {
		t.Fatalf("parsed (%d, %v), want (V1, empty)", ver, payload)
	}
}

// Stamping with the reserved legacy sentinel (0) is a caller error → nil, so a
// mis-stamped context can never masquerade as valid.
func TestDetectionContext_RejectsLegacyStamp(t *testing.T) {
	if dc := NewDetectionContext(DetectionContextSchemaLegacy, []byte{1, 2, 3}); dc != nil {
		t.Fatalf("stamping version 0 returned %v, want nil", dc)
	}
}

// A consumer dispatches on byte 0; a future/unknown version is still cleanly
// recoverable (version + payload) so the consumer can refuse it explicitly
// rather than read garbage.
func TestDetectionContext_UnknownVersionDispatchable(t *testing.T) {
	future := NewDetectionContext(42, []byte{0x01})
	ver, payload := ParseDetectionContext(future)
	if ver != 42 {
		t.Fatalf("version = %d, want 42", ver)
	}
	if len(payload) != 1 || payload[0] != 0x01 {
		t.Fatalf("payload = %v, want [1]", payload)
	}
}
