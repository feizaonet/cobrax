package cobrax

import (
	"context"
	"testing"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSEBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		baseURL  string
		expected string
	}{
		{"explicit base url wins", ":8090", "https://api.example.com", "https://api.example.com"},
		{"colon addr normalised", ":8090", "", "http://localhost:8090"},
		{"host addr used as-is", "localhost:8080", "", "http://localhost:8080"},
		{"empty addr", "", "", "http://"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sseBaseURL(tt.addr, tt.baseURL))
		})
	}
}

// recordingTransport captures the Serve arguments so tests can assert on them.
type recordingTransport struct {
	called bool
	opts   ServeOptions
}

func (r *recordingTransport) Serve(_ context.Context, _ *mcpserver.MCPServer, opts ServeOptions) error {
	r.called = true
	r.opts = opts
	return nil
}

func TestWithTransport(t *testing.T) {
	factory := func() *cobra.Command {
		return &cobra.Command{Use: "root", Run: func(_ *cobra.Command, _ []string) {}}
	}

	rec := &recordingTransport{}
	srv, err := NewMCPServer(
		MCPOptions{Enabled: true, Addr: ":12345", Name: "test"},
		factory,
		WithTransport(rec),
	)
	require.NoError(t, err)

	err = srv.Start(context.Background())
	require.NoError(t, err)

	assert.True(t, rec.called, "custom transport should be used")
	assert.Equal(t, ":12345", rec.opts.Addr)
}
