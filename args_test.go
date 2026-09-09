package cobrax

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────────────────────────────────────
// parseArgSpecs tests
// ──────────────────────────────────────────────────────────────────────────────

func TestParseArgSpecs_NoArgs(t *testing.T) {
	cmd := &cobra.Command{Use: "serve"}
	specs := parseArgSpecs(cmd)
	assert.Nil(t, specs, "no arg tokens → nil specs")
}

func TestParseArgSpecs_FlagsOnly(t *testing.T) {
	cmd := &cobra.Command{Use: "list [flags]"}
	specs := parseArgSpecs(cmd)
	assert.Nil(t, specs, "[flags] sentinel should be ignored")
}

func TestParseArgSpecs_OptionalSingleArg(t *testing.T) {
	cmd := &cobra.Command{Use: "get [resource]"}
	specs := parseArgSpecs(cmd)
	require.Len(t, specs, 1)
	s := specs[0]
	assert.Equal(t, "resource", s.Name)
	assert.False(t, s.Required)
	assert.False(t, s.Variadic)
	assert.Equal(t, 0, s.Index)
}

func TestParseArgSpecs_RequiredSingleArg(t *testing.T) {
	cmd := &cobra.Command{Use: "delete <name>"}
	specs := parseArgSpecs(cmd)
	require.Len(t, specs, 1)
	s := specs[0]
	assert.Equal(t, "name", s.Name)
	assert.True(t, s.Required)
	assert.False(t, s.Variadic)
}

func TestParseArgSpecs_VariadicOptional(t *testing.T) {
	cmd := &cobra.Command{Use: "make [targets...]"}
	specs := parseArgSpecs(cmd)
	require.Len(t, specs, 1)
	s := specs[0]
	assert.Equal(t, "targets", s.Name)
	assert.False(t, s.Required)
	assert.True(t, s.Variadic)
}

func TestParseArgSpecs_VariadicRequired(t *testing.T) {
	cmd := &cobra.Command{Use: "run <files...>"}
	specs := parseArgSpecs(cmd)
	require.Len(t, specs, 1)
	s := specs[0]
	assert.Equal(t, "files", s.Name)
	assert.True(t, s.Required)
	assert.True(t, s.Variadic)
}

func TestParseArgSpecs_MultipleArgs(t *testing.T) {
	// Simulates: /oncall <module> [title]
	cmd := &cobra.Command{Use: "oncall <module> [title]"}
	specs := parseArgSpecs(cmd)
	require.Len(t, specs, 2)

	assert.Equal(t, "module", specs[0].Name)
	assert.True(t, specs[0].Required)
	assert.Equal(t, 0, specs[0].Index)

	assert.Equal(t, "title", specs[1].Name)
	assert.False(t, specs[1].Required)
	assert.Equal(t, 1, specs[1].Index)
}

func TestParseArgSpecs_AnnotationDescription(t *testing.T) {
	cmd := &cobra.Command{
		Use: "oncall <module> [title]",
		Annotations: map[string]string{
			AnnotationArgPrefix + "0": "The on-call module (e.g. bke, kafka)",
			AnnotationArgPrefix + "1": "Short incident title",
		},
	}
	specs := parseArgSpecs(cmd)
	require.Len(t, specs, 2)
	assert.Equal(t, "The on-call module (e.g. bke, kafka)", specs[0].Description)
	assert.Equal(t, "Short incident title", specs[1].Description)
}

func TestParseArgSpecs_FlagsAndArgsMixed(t *testing.T) {
	cmd := &cobra.Command{Use: "push <image> [tag] [flags]"}
	specs := parseArgSpecs(cmd)
	// [flags] is stripped before regex matching
	require.Len(t, specs, 2)
	assert.Equal(t, "image", specs[0].Name)
	assert.Equal(t, "tag", specs[1].Name)
}
