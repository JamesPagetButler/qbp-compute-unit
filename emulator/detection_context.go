// detection_context.go — schema-versioning for SeamEvent.DetectionContext (issue #42).
//
// SeamEvent.DetectionContext is an OPAQUE []byte the substrate emits verbatim
// (doc/design/m1-gearbox.md §5.4a); consumers (the BMA harness, and future
// tenants) parse it. Once the substrate's DetectionContext shape can evolve
// independently of a given consumer (Walk-α onward, as Gearbox iterations
// diverge from BMA-harness iterations), a consumer reading bytes with the wrong
// schema would get silent garbage with no way to signal "I don't recognize this
// format". The two §I4 reviewers on PR #40 converged on option (a): reserve the
// FIRST byte as a schema-version tag so consumers can dispatch parsers correctly.
//
// Convention (ratified via #42):
//
//	byte 0      — schema version, uint8. 0 is reserved as the "unversioned /
//	              legacy" sentinel; real schemas are 1..255.
//	byte 1..N   — schema-specific payload, still opaque to the substrate.
//
// The substrate does NOT interpret the payload — it only stamps/strips the
// leading version byte via the helpers here, preserving the substrate-emits-
// verbatim discipline. A consumer dispatches on byte 0 and may reject versions
// it does not implement. Because nothing populates DetectionContext today, this
// is additive forward-compat scaffolding, not a breaking wire change: the first
// populated format simply starts at V1.
package emulator

const (
	// DetectionContextSchemaLegacy is the reserved sentinel for an unversioned
	// (pre-#42) or empty DetectionContext. Consumers should treat it as "no
	// recognized schema" and refuse to parse a structured payload from it.
	DetectionContextSchemaLegacy uint8 = 0

	// DetectionContextSchemaV1 is the first versioned DetectionContext schema.
	DetectionContextSchemaV1 uint8 = 1
)

// NewDetectionContext returns a DetectionContext whose leading byte is the given
// schema version, followed by the opaque payload verbatim. version must be
// non-zero (0 is the legacy sentinel); passing 0 is a caller error and returns
// nil so a mis-stamped context can never masquerade as a valid one.
func NewDetectionContext(version uint8, payload []byte) []byte {
	if version == DetectionContextSchemaLegacy {
		return nil
	}
	dc := make([]byte, 0, len(payload)+1)
	dc = append(dc, version)
	dc = append(dc, payload...)
	return dc
}

// NewDetectionContextV1 is the convenience constructor for the current schema.
func NewDetectionContextV1(payload []byte) []byte {
	return NewDetectionContext(DetectionContextSchemaV1, payload)
}

// ParseDetectionContext splits a DetectionContext into its schema version and
// opaque payload. An empty (or nil) context returns (DetectionContextSchemaLegacy,
// nil) — the "no recognized schema" sentinel — so an unpopulated DetectionContext
// is never mistaken for a versioned one. The payload is a sub-slice of dc (it
// aliases the same backing array; copy it if the caller needs to retain it past
// dc's lifetime). The substrate performs no payload interpretation; dispatching
// on the returned version is the consumer's responsibility.
func ParseDetectionContext(dc []byte) (version uint8, payload []byte) {
	if len(dc) == 0 {
		return DetectionContextSchemaLegacy, nil
	}
	return dc[0], dc[1:]
}
