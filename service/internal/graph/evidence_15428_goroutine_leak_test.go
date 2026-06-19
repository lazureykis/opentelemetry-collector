// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Evidence for review of PR #15428. NOT for upstream merge.
//
// PR #15428 replaces the blocking `host.AsyncErrorChannel <- event.Err()` with
// `go func(){ host.AsyncErrorChannel <- event.Err() }()`. The control loop in
// otelcol/collector.go reads that unbuffered channel only inside its select; the
// first `break LOOP` (shutdown, config-reload-failure, signal, ctx done) exits the
// loop and the channel is never read again. So a fatal error reported during
// shutdown/reload spawns a goroutine that blocks forever: the error is neither
// delivered nor "dropped cleanly" — it leaks one goroutine per fatal error.
//
// This test proves the leak quantitatively, then drains the channel to release the
// parked senders so the package's goleak.VerifyTestMain stays green.

package graph

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/service/extensions"
)

func TestEvidence15428_FatalErrorWithoutReaderLeaksGoroutine(t *testing.T) {
	// Unbuffered channel with no reader == the state after the control loop's
	// `break LOOP`, i.e. during shutdown or a config reload.
	ch := make(chan error)
	host := &Host{
		AsyncErrorChannel: ch,
		ServiceExtensions: &extensions.Extensions{},
	}

	before := runtime.NumGoroutine()

	const n = 50
	for i := 0; i < n; i++ {
		host.NotifyComponentStatusChange(nil, componentstatus.NewFatalErrorEvent(assert.AnError))
	}

	// Every fatal error spawns a sender that parks on the unbuffered send.
	require.Eventually(t, func() bool {
		return runtime.NumGoroutine() >= before+n
	}, time.Second, 5*time.Millisecond,
		"expected n sender goroutines blocked on the channel send")

	// They never leave on their own: nobody will read this channel again.
	time.Sleep(100 * time.Millisecond)
	leaked := runtime.NumGoroutine() - before
	assert.GreaterOrEqual(t, leaked, n,
		"a fatal error reported during shutdown/reload leaks a goroutine blocked forever on the unbuffered send")
	t.Logf("leaked goroutines after %d fatal errors with no reader: %d", n, leaked)

	// Cleanup so this evidence test itself does not trip goleak: draining the
	// channel releases the parked senders, confirming they were only ever blocked
	// on the send (not doing useful work). goleak.VerifyTestMain is the arbiter
	// that they actually exit.
	for i := 0; i < n; i++ {
		<-ch
	}
}
