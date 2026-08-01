// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Repro for the unchecked-index panics left in pprofile after #15517.
//
// #15517 added bounds checks to FromAttributeIndices. Two sibling paths kept the
// same untrusted-input crash:
//
//   - FromLocationIndices (locations.go:18) has no bounds check at all.
//   - The switchDictionary loops validate only the upper bound
//     (`Len() <= int(v)`), so a NEGATIVE index passes the guard and panics.
//
// Every test below FAILS on main with "index out of range".
// Not for upstream merge — evidence only.

// PART A — FromLocationIndices.

func TestRepro_FromLocationIndices_OutOfRange(t *testing.T) {
	table := NewLocationSlice()
	table.AppendEmpty().SetAddress(1)

	stack := NewStack()
	stack.LocationIndices().Append(5) // table.Len() == 1

	require.NotPanics(t, func() {
		FromLocationIndices(table, stack)
	}, "out-of-range location index must not panic")
}

func TestRepro_FromLocationIndices_Negative(t *testing.T) {
	table := NewLocationSlice()
	table.AppendEmpty().SetAddress(1)

	stack := NewStack()
	stack.LocationIndices().Append(-1)

	require.NotPanics(t, func() {
		FromLocationIndices(table, stack)
	}, "negative location index must not panic")
}

// PART B — switchDictionary negative indices, reached via the public Profiles.MergeTo.

func negativeIndexProfiles(t *testing.T, apply func(Profile)) Profiles {
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
	apply(prof)

	return p
}

func TestRepro_MergeTo_NegativeProfileAttributeIndex(t *testing.T) {
	p := negativeIndexProfiles(t, func(prof Profile) {
		prof.AttributeIndices().Append(-1)
	})

	require.NotPanics(t, func() {
		_ = p.MergeTo(NewProfiles())
	}, "negative profile attribute index must not panic")
}

func TestRepro_MergeTo_NegativeSampleAttributeIndex(t *testing.T) {
	p := negativeIndexProfiles(t, func(prof Profile) {
		prof.Samples().AppendEmpty().AttributeIndices().Append(-1)
	})

	require.NotPanics(t, func() {
		_ = p.MergeTo(NewProfiles())
	}, "negative sample attribute index must not panic")
}

func TestRepro_MergeTo_NegativeStackLocationIndex(t *testing.T) {
	p := NewProfiles()
	dic := p.Dictionary()
	dic.StringTable().Append("")
	dic.LocationTable().AppendEmpty().SetAddress(1)
	dic.StackTable().AppendEmpty().LocationIndices().Append(-1)

	prof := p.ResourceProfiles().AppendEmpty().
		ScopeProfiles().AppendEmpty().
		Profiles().AppendEmpty()
	prof.Samples().AppendEmpty().SetStackIndex(0)

	require.NotPanics(t, func() {
		_ = p.MergeTo(NewProfiles())
	}, "negative stack location index must not panic")
}
