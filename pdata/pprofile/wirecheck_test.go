// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Does a NEGATIVE index survive the OTLP wire format and reach the panic?
// If it does, the switchDictionary crash is remotely triggerable, not just a
// library-API concern.

func negativeIndexOnTheWire(t *testing.T) Profiles {
	t.Helper()

	p := NewProfiles()
	dic := p.Dictionary()
	dic.StringTable().Append("")
	attr := dic.AttributeTable().AppendEmpty()
	attr.SetKeyStrindex(0)
	attr.Value().SetStr("v")
	dic.LocationTable().AppendEmpty().SetAddress(1)

	prof := p.ResourceProfiles().AppendEmpty().
		ScopeProfiles().AppendEmpty().
		Profiles().AppendEmpty()
	prof.Samples().AppendEmpty().AttributeIndices().Append(-1)

	buf, err := (&ProtoMarshaler{}).MarshalProfiles(p)
	require.NoError(t, err, "marshal must succeed")

	got, err := (&ProtoUnmarshaler{}).UnmarshalProfiles(buf)
	require.NoError(t, err, "OTLP unmarshal accepted the payload")

	return got
}

func TestWire_NegativeIndexSurvivesUnmarshal(t *testing.T) {
	got := negativeIndexOnTheWire(t)

	idx := got.ResourceProfiles().At(0).ScopeProfiles().At(0).
		Profiles().At(0).Samples().At(0).AttributeIndices().At(0)

	require.Equal(t, int32(-1), idx,
		"negative index survived the proto round trip unvalidated")
}

func TestWire_NegativeIndexFromWirePanicsOnMergeTo(t *testing.T) {
	got := negativeIndexOnTheWire(t)

	require.NotPanics(t, func() {
		_ = got.MergeTo(NewProfiles())
	}, "profile decoded from the wire must not panic in MergeTo")
}
