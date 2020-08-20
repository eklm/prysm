// Package types includes important type definitions for
// slashable objects detected by slasher.
package types

import (
	"errors"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// DetectionKind defines an enum type that
// gives us information on the type of slashable offense
// found when analyzing validator min-max spans.
type DetectionKind uint8

const (
	// DoubleVote denotes a slashable offense in which
	// a validator cast two conflicting attestations within
	// the same target epoch.
	DoubleVote DetectionKind = iota
	// SurroundVote denotes a slashable offense in which
	// a validator surrounded or was surrounded by a previous
	// attestation created by the same validator.
	SurroundVote
)

// DetectionResult tells us the kind of slashable
// offense found from detecting on min-max spans +
// the slashable epoch for the offense.
// Also includes the signature bytes for assistance in
// finding the attestation for the slashing proof.
type DetectionResult struct {
	ValidatorIndex uint64
	SlashableEpoch uint64
	Kind           DetectionKind
	SigBytes       [2]byte
}

// Marshal the result into bytes, used for removing duplicates.
func (result *DetectionResult) Marshal() []byte {
	numBytes := bytesutil.ToBytes(result.SlashableEpoch, 8)
	var resultBytes []byte
	resultBytes = append(resultBytes, uint8(result.Kind))
	resultBytes = append(resultBytes, result.SigBytes[:]...)
	resultBytes = append(resultBytes, numBytes...)
	return resultBytes
}

// Span defines the structure used for detecting surround and double votes.
type Span struct {
	MinSpan     uint16
	MaxSpan     uint16
	SigBytes    [2]byte
	HasAttested bool
}

// SpannerEncodedLength the byte length of validator span data structure.
var SpannerEncodedLength = uint64(7)

// UnmarshalSpan returns a span from an encoded, flattened byte array.
func UnmarshalSpan(enc []byte) (Span, error) {
	r := Span{}
	if len(enc) != int(SpannerEncodedLength) {
		return r, errors.New("wrong data length for min max span")
	}
	r.MinSpan = uint16(enc[0]) | uint16(enc[1])<<8
	r.MaxSpan = uint16(enc[2]) | uint16(enc[3])<<8
	sigB := [2]byte{}
	copy(sigB[:], enc[4:6])
	r.SigBytes = sigB
	r.HasAttested = enc[6]&1 == 1
	return r, nil
}

// Marshal converts the span struct into a flattened byte array.
func (span Span) Marshal() []byte {
	var attested byte = 0
	if span.HasAttested {
		attested = 1
	}
	return []byte{
		byte(span.MinSpan),
		byte(span.MinSpan >> 8),
		byte(span.MaxSpan),
		byte(span.MaxSpan >> 8),
		span.SigBytes[0],
		span.SigBytes[1],
		attested,
	}
}
