// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Evidence for PR #15433 review — NOT for upstream merge.
//
// PR #15433 adds a recursion-depth guard to the protobuf unmarshal path
// (AnyValue -> ArrayValue/KeyValueList -> AnyValue). The identical unbounded
// recursion exists in the JSON unmarshal path (AnyValue.UnmarshalJSON ->
// ArrayValue/KeyValueList.UnmarshalJSON -> AnyValue.UnmarshalJSON), which the
// OTLP/HTTP receiver reaches for `Content-Type: application/json` requests.
// This PR does not touch that path, so the stack-overflow DoS remains reachable
// over JSON.
//
//   - TestGapProtoIsGuarded / TestGapJSONIsUnbounded: run by default; show the
//     proto path returns ErrRecursionDepth while the JSON path recurses past the
//     same limit with no guard.
//   - TestGapJSONOverflowsStack: set GAP_OVERFLOW=1 to run; shrinks the stack and
//     drives the JSON path into "fatal error: stack overflow". The proto path
//     under the same shrunk stack returns the error instead of crashing.
package internal

import (
	"os"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/proto"
)

func gapVarint(buf []byte, v uint64) []byte {
	for v >= 1<<7 {
		buf = append(buf, uint8(v&0x7f|0x80))
		v >>= 7
	}
	return append(buf, uint8(v))
}

func gapField(tag byte, payload []byte) []byte {
	f := append([]byte{tag}, gapVarint(nil, uint64(len(payload)))...)
	return append(f, payload...)
}

// AnyValue.ArrayValue (field 5) -> ArrayValue.Values (field 1, AnyValue), repeated depth times.
func gapNestedArrayProto(depth int) []byte {
	var p []byte
	for i := 0; i < depth; i++ {
		p = gapField(0x2a, gapField(0x0a, p))
	}
	return p
}

// {"arrayValue":{"values":[ ... ]}} nested depth times.
func gapNestedArrayJSON(depth int) []byte {
	var sb strings.Builder
	for i := 0; i < depth; i++ {
		sb.WriteString(`{"arrayValue":{"values":[`)
	}
	for i := 0; i < depth; i++ {
		sb.WriteString(`]}}`)
	}
	return []byte(sb.String())
}

func TestGapProtoIsGuarded(t *testing.T) {
	dest := NewAnyValue()
	err := dest.UnmarshalProto(gapNestedArrayProto(proto.RecursionLimit + 100))
	require.ErrorIs(t, err, proto.ErrRecursionDepth)
}

func TestGapJSONIsUnbounded(t *testing.T) {
	iter := json.BorrowIterator(gapNestedArrayJSON(proto.RecursionLimit * 3))
	defer json.ReturnIterator(iter)
	dest := NewAnyValue()
	dest.UnmarshalJSON(iter)
	require.NotErrorIs(t, iter.Error(), proto.ErrRecursionDepth)
	require.NoError(t, iter.Error(), "JSON path recurses past the proto RecursionLimit with no depth guard")
}

func TestGapJSONOverflowsStack(t *testing.T) {
	if os.Getenv("GAP_OVERFLOW") != "1" {
		t.Skip("set GAP_OVERFLOW=1 to run; this crashes the test binary with a stack overflow")
	}
	debug.SetMaxStack(8 * 1024 * 1024)
	iter := json.BorrowIterator(gapNestedArrayJSON(2_000_000))
	defer json.ReturnIterator(iter)
	dest := NewAnyValue()
	dest.UnmarshalJSON(iter)
	t.Fatal("expected a stack overflow but UnmarshalJSON returned")
}
