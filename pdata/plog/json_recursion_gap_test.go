package plog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nestedBodyJSON builds an OTLP/JSON Logs payload whose single log record body is an
// AnyValue nested `depth` arrayValue levels deep. This is the same recursive shape the
// proto path now guards with RecursionLimit=100 in PR #15597.
func nestedBodyJSON(depth int) []byte {
	var b strings.Builder
	b.WriteString(`{"resourceLogs":[{"scopeLogs":[{"logRecords":[{"body":`)
	for i := 0; i < depth; i++ {
		b.WriteString(`{"arrayValue":{"values":[`)
	}
	b.WriteString(`{"stringValue":"x"}`)
	for i := 0; i < depth; i++ {
		b.WriteString(`]}}`)
	}
	b.WriteString(`}]}]}]}`)
	return []byte(b.String())
}

// Test_JSONUnmarshaler_NoRecursionLimit demonstrates that PR #15597's depth guard covers
// only the proto path: the JSON unmarshaler on the same public pdata API accepts nesting
// far beyond RecursionLimit (100), where the proto path rejects with ErrRecursionDepth.
func Test_JSONUnmarshaler_NoRecursionLimit(t *testing.T) {
	// Depth 200 is 2x the proto RecursionLimit. Proto rejects this; JSON accepts it.
	logs, err := (&JSONUnmarshaler{}).UnmarshalLogs(nestedBodyJSON(200))
	require.NoError(t, err, "JSON path has no recursion guard — accepts depth 200 (proto rejects at 100)")
	assert.Equal(t, 1, logs.ResourceLogs().Len())
}
