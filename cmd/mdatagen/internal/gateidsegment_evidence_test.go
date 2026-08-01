// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Evidence for https://github.com/open-telemetry/opentelemetry-collector/issues/15676.
// mdatagen keeps its own copy of the pre-PR feature-gate ID regexp
// (cmd/mdatagen/internal/metadata.go:24), so IDs with an empty dot-separated segment
// still pass mdatagen validation on the PR branch, where featuregate.validateID
// rejects them.
func TestEvidenceMdatagenAcceptsEmptySegmentGateIDs(t *testing.T) {
	for _, id := range []string{
		"receiver.sample.",
		"receiver.sample.example.",
		"receiver.sample..example",
		"receiver.sample.example..gate",
	} {
		t.Run(id, func(t *testing.T) {
			md := &Metadata{
				Type:   "sample",
				Status: &Status{Class: "receiver"},
				FeatureGates: []FeatureGate{
					{
						ID:           FeatureGateID(id),
						Description:  "Test feature",
						Stage:        FeatureGateStageAlpha,
						FromVersion:  "v0.100.0",
						ReferenceURL: "https://github.com/open-telemetry/opentelemetry-collector/issues/12345",
					},
				},
			}

			require.NoError(t, md.validateFeatureGates())
		})
	}
}
