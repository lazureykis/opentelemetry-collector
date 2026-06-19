// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

// TestPartitionBatcher_ShutdownDeadlockWithTimer demonstrates a deadlock
// introduced by holding currentBatchMu across stopWG.Wait() in
// shutdownInternal(). The timer goroutine launched by Start() is counted in
// stopWG (via stopWG.Go) and itself acquires currentBatchMu inside
// flushCurrentBatchOrRemovePartition()/flush(). When shutdownInternal() holds
// currentBatchMu and blocks in stopWG.Wait(), a timer goroutine that takes the
// timer.C branch and tries to acquire currentBatchMu can never finish, so
// stopWG never drains:
//
//	shutdownInternal: holds currentBatchMu  -> waits on stopWG
//	timer goroutine : counted in stopWG     -> blocks on currentBatchMu
//
// A blocking consumeFunc holds the single worker so shutdown's Wait() stays
// open long enough for the timer goroutine to wedge on the lock. The tiny
// FlushTimeout only makes the race deterministic; it exists for any
// FlushTimeout > 0. This test fails (deadlocks) on the PR branch and passes on
// main, where Wait() runs without holding currentBatchMu.
//
// Not for upstream merge — evidence repro for PR #15464.
func TestPartitionBatcher_ShutdownDeadlockWithTimer(t *testing.T) {
	for iter := 0; iter < 50; iter++ {
		release := make(chan struct{})
		consume := func(context.Context, request.Request) error {
			<-release
			return nil
		}

		cfg := BatchConfig{
			FlushTimeout: 100 * time.Nanosecond, // timer effectively always ready, beats the reset-mask
			Sizer:        request.SizerTypeItems,
			MinSize:      1, // flush immediately so the single worker is occupied
		}
		ba := newPartitionBatcher(cfg, request.NewItemsSizer(), nil, newWorkerPool(1), consume, zap.NewNop(), nil)
		require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))

		// Occupies the single worker; it blocks in consume on <-release, keeping
		// stopWG > 0 so shutdown's Wait() blocks.
		ba.Consume(context.Background(), &requesttest.FakeRequest{Items: 1, Bytes: 1}, newFakeDone())

		shutdownDone := make(chan struct{})
		go func() {
			_ = ba.Shutdown(context.Background())
			close(shutdownDone)
		}()

		// Give shutdown time to reach Wait() while holding currentBatchMu, during
		// which the timer goroutine may wedge on the lock.
		time.Sleep(20 * time.Millisecond)
		close(release) // unblock the worker; only a wedged timer goroutine keeps stopWG > 0

		select {
		case <-shutdownDone:
		case <-time.After(5 * time.Second):
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, true)
			t.Fatalf("Shutdown deadlocked on iteration %d\n%s", iter, buf[:n])
		}
	}
}
