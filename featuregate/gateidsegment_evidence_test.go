// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Evidence for https://github.com/open-telemetry/opentelemetry-collector/issues/15676.
// The same IDs that mdatagen accepts (see cmd/mdatagen/internal/gateidsegment_evidence_test.go)
// make MustRegister panic on this branch, and mdatagen emits MustRegister at package scope,
// so the panic happens at package init.
func TestEvidenceMustRegisterPanicsOnEmptySegmentGateIDs(t *testing.T) {
	for _, id := range []string{
		"receiver.sample.",
		"receiver.sample.example.",
		"receiver.sample..example",
		"receiver.sample.example..gate",
	} {
		t.Run(id, func(t *testing.T) {
			require.Panics(t, func() {
				NewRegistry().MustRegister(id, StageAlpha)
			})
		})
	}
}
