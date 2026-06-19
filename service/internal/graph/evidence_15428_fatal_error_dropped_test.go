// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Evidence for review of PR #15428. NOT for upstream merge.
//
// PR #15428 (HEAD a60a05218) replaces the blocking
// `host.AsyncErrorChannel <- event.Err()` with a non-blocking send:
//
//	select {
//	case host.AsyncErrorChannel <- event.Err():
//	default:
//	}
//
// That removes the deadlock and the earlier goroutine leak. But the channel is
// still constructed unbuffered (otelcol/collector.go: `make(chan error)`), and
// the control loop reads it only inside its select. There are two windows where
// no receiver is parked, so a non-blocking send lands on `default:` and the
// fatal error is silently dropped:
//
//   - Startup: setupConfigurationComponents (which calls every component's
//     Start) runs before the `for { select }` control loop begins.
//   - Config reload / SIGHUP: while reloadConfiguration executes, the loop is
//     not in its select.
//
// A fatal error dropped in either window means the collector keeps running
// instead of terminating.
//
// The first test reproduces the drop against the real
// Host.NotifyComponentStatusChange on a production-equivalent unbuffered
// channel. The second shows the proposed one-line fix — `make(chan error, 1)`
// in otelcol/collector.go — delivers the first fatal error even when no
// receiver is parked, while keeping the non-blocking send (so the deadlock and
// leak stay fixed).

package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/service/extensions"
)

func TestEvidence15428_FatalErrorDroppedWhenLoopNotParked(t *testing.T) {
	// Unbuffered, matching otelcol/collector.go on the PR HEAD. No receiver
	// parked models the startup / reload window.
	ch := make(chan error)
	host := &Host{
		AsyncErrorChannel: ch,
		ServiceExtensions: &extensions.Extensions{},
	}

	host.NotifyComponentStatusChange(nil, componentstatus.NewFatalErrorEvent(assert.AnError))

	// The control loop later returns to its select and tries to read.
	select {
	case err := <-ch:
		t.Fatalf("expected the fatal error to be dropped on the unbuffered channel, but it was delivered: %v", err)
	default:
		t.Log("fatal error reported while the loop was not parked was silently dropped; the collector would not terminate")
	}
}

func TestEvidence15428_BufferedChannelDeliversFatalError(t *testing.T) {
	// Proposed fix: make(chan error, 1) in otelcol/collector.go. The
	// non-blocking send lands in the buffer even with no receiver parked, so
	// the control loop drains it on its next select iteration and terminates.
	ch := make(chan error, 1)
	host := &Host{
		AsyncErrorChannel: ch,
		ServiceExtensions: &extensions.Extensions{},
	}

	host.NotifyComponentStatusChange(nil, componentstatus.NewFatalErrorEvent(assert.AnError))

	select {
	case err := <-ch:
		require.ErrorIs(t, err, assert.AnError)
	default:
		t.Fatal("fatal error was dropped even with a buffered channel")
	}

	// A second fatal error before the loop drains the buffer coalesces via the
	// non-blocking send's default arm — no block, no leak.
	host.NotifyComponentStatusChange(nil, componentstatus.NewFatalErrorEvent(assert.AnError))
}
