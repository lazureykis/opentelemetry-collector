package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildProfilesWithAttr() Profiles {
	p := NewProfiles()
	rp := p.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr("host.name", "server-1")
	sp := rp.ScopeProfiles().AppendEmpty()
	sp.Scope().Attributes().PutStr("scope.key", "scope-val")
	return p
}

func TestJSONMarshalProfilesMutatesInput(t *testing.T) {
	p := buildProfilesWithAttr()
	want := NewProfiles()
	p.CopyTo(want)

	_, err := (&JSONMarshaler{}).MarshalProfiles(p)
	require.NoError(t, err)

	got, ok := p.ResourceProfiles().At(0).Resource().Attributes().Get("host.name")
	require.True(t, ok, "host.name attribute disappeared from caller's Profiles after MarshalProfiles")
	assert.Equal(t, "server-1", got.Str())
	assert.Equal(t, want, p, "MarshalProfiles must not mutate its input")
}

func TestProtoMarshalProfilesMutatesInput(t *testing.T) {
	p := buildProfilesWithAttr()
	want := NewProfiles()
	p.CopyTo(want)

	_, err := (&ProtoMarshaler{}).MarshalProfiles(p)
	require.NoError(t, err)

	got, ok := p.ResourceProfiles().At(0).Resource().Attributes().Get("host.name")
	require.True(t, ok, "host.name attribute disappeared from caller's Profiles after MarshalProfiles")
	assert.Equal(t, "server-1", got.Str())
	assert.Equal(t, want, p, "MarshalProfiles must not mutate its input")
}
