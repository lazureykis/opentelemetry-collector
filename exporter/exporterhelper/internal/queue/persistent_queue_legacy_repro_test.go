// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

// Simulates upgrading a collector whose persistent queue on disk is still in the
// pre-#13330 legacy format (individual ri/wi/di keys, no consolidated qmv0 key),
// with two unsent items persisted. The migration path must recover them.
//
// PASS on main (loadLegacyMetadata migrates the indices). FAIL on #15406 head
// (the legacy branch is gone -> queue starts empty -> the persisted items are
// silently abandoned = data loss on upgrade).
func TestPersistentQueue_LegacyFormatUpgradeDataLoss(t *testing.T) {
	ctx := context.Background()
	ext := storagetest.NewMockStorageExtension(nil)
	client, err := ext.GetClient(ctx, component.KindExporter, component.NewID(exportertest.NopType), pipeline.SignalTraces.String())
	require.NoError(t, err)

	// Two unsent items written by an older collector: indices 0 and 1.
	riBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(riBuf, 0)
	wiBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(wiBuf, 2)
	require.NoError(t, client.Set(ctx, "ri", riBuf))
	require.NoError(t, client.Set(ctx, "wi", wiBuf))
	// Item payloads under decimal-index keys, encoded as int64Encoding does.
	require.NoError(t, client.Set(ctx, "0", []byte("11")))
	require.NoError(t, client.Set(ctx, "1", []byte("22")))

	pq := createTestPersistentQueueWithClient(client)

	require.Equal(t, int64(2), pq.Size(), "two unsent items persisted in the legacy format were lost on upgrade")

	_, got1, _, ok := pq.Read(ctx)
	require.True(t, ok)
	require.Equal(t, intRequest(11), got1)
	_, got2, _, ok := pq.Read(ctx)
	require.True(t, ok)
	require.Equal(t, intRequest(22), got2)
}
