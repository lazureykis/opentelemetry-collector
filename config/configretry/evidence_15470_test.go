// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Evidence for review of PR #15470.
//
// This test PASSING is the regression. The proof config below is a disabled
// BackOffConfig whose relational invariant is violated: MaxElapsedTime (1s) is
// less than InitialInterval (10s). On current main (post-#15443, commit
// 9782f9e8a, which removed the `if !bs.Enabled { return nil }` early-return),
// Validate() runs the relational checks regardless of Enabled and REJECTS this
// config with "'max_elapsed_time' must not be less than 'initial_interval'".
//
// This PR branch re-inserts the `!Enabled` guard above the relational checks
// (backoff.go ~L63), so Validate() returns nil for the same config — silently
// reverting #15443. This test asserts that wrong nil result to document the
// regression on this branch.
package configretry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvidence15470DisabledRelationalRevert(t *testing.T) {
	cfg := BackOffConfig{
		Enabled:         false,
		InitialInterval: 10 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxElapsedTime:  1 * time.Second,
	}
	require.NoError(t, cfg.Validate())
}
