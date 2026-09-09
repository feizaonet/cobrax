package cobrax

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunInProcessLargeOutput verifies that a command producing more output
// than the OS pipe buffer does not deadlock. Before the fix, the pipe was
// drained only after ExecuteContext returned, so a verbose command would block
// forever on write.
func TestRunInProcessLargeOutput(t *testing.T) {
	factory := func() *cobra.Command {
		root := &cobra.Command{Use: "root"}
		big := &cobra.Command{
			Use: "big",
			Run: func(cmd *cobra.Command, _ []string) {
				// Output far more than the OS pipe buffer (typically 16-64KB).
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("x", 256*1024))
			},
		}
		root.AddCommand(big)
		return root
	}

	srv, err := NewMCPServer(MCPOptions{Name: "root"}, factory)
	require.NoError(t, err)

	tools := srv.Tools()
	require.Len(t, tools, 1)

	req := mcp.CallToolRequest{}
	req.Params.Name = tools[0].Name

	done := make(chan struct{})
	var output ToolOutput
	var runErr error
	go func() {
		defer close(done)
		_, output, runErr = srv.runInProcess(context.Background(), req, ToolInput{})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runInProcess did not return; pipe likely not drained (deadlock)")
	}

	require.NoError(t, runErr)
	assert.GreaterOrEqual(t, len(output.StdOut), 256*1024,
		"captured stdout should include the full large output")
}

// TestRunInProcessConcurrent verifies that concurrent in-process calls are
// serialised safely: each call's stdout is captured intact and is not polluted
// by another call's output. Run with -race to catch data races on the buffers.
func TestRunInProcessConcurrent(t *testing.T) {
	factory := func() *cobra.Command {
		root := &cobra.Command{Use: "root"}
		for i := 0; i < 4; i++ {
			c := &cobra.Command{
				Use: fmt.Sprintf("cmd%d", i),
				Run: func(cmd *cobra.Command, _ []string) {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s-start\n", cmd.Name())
					time.Sleep(5 * time.Millisecond)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s-end\n", cmd.Name())
				},
			}
			root.AddCommand(c)
		}
		return root
	}

	srv, err := NewMCPServer(MCPOptions{Name: "root"}, factory)
	require.NoError(t, err)

	tools := srv.Tools()
	require.Len(t, tools, 4)

	var wg sync.WaitGroup
	for _, tool := range tools {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			req := mcp.CallToolRequest{}
			req.Params.Name = name
			_, output, err := srv.runInProcess(context.Background(), req, ToolInput{})
			require.NoError(t, err)
			assert.Contains(t, output.StdOut, name+"-start")
			assert.Contains(t, output.StdOut, name+"-end")
		}(tool.Name)
	}
	wg.Wait()
}

// TestInProcessHandlerRecoversPanic verifies that a panic inside a command's
// Run is recovered by the handler and surfaced as a Go error (and that the
// serialisation mutex is released on the panic path).
func TestInProcessHandlerRecoversPanic(t *testing.T) {
	factory := func() *cobra.Command {
		root := &cobra.Command{Use: "root"}
		c := &cobra.Command{
			Use: "boom",
			Run: func(_ *cobra.Command, _ []string) {
				panic("boom")
			},
		}
		root.AddCommand(c)
		return root
	}

	srv, err := NewMCPServer(MCPOptions{Name: "root"}, factory)
	require.NoError(t, err)

	tools := srv.Tools()
	require.Len(t, tools, 1)

	handler := srv.makeInProcessHandler(Selector{}, nil)
	req := mcp.CallToolRequest{}
	req.Params.Name = tools[0].Name

	_, _, err = handler(context.Background(), req, ToolInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
}

// TestEinoToolsHasInputSchema verifies that EinoTools surfaces the tool's
// input schema. Before the fix, the zero-value InputSchema was serialised and
// the eino tools had empty ParamsOneOf, so LLMs received no parameter schema.
func TestEinoToolsHasInputSchema(t *testing.T) {
	factory := func() *cobra.Command {
		root := &cobra.Command{Use: "root"}
		c := &cobra.Command{
			Use: "greet",
			Run: func(_ *cobra.Command, _ []string) {},
		}
		c.Flags().String("name", "", "Name to greet")
		root.AddCommand(c)
		return root
	}

	srv, err := NewMCPServer(MCPOptions{Name: "root"}, factory)
	require.NoError(t, err)

	tools, err := srv.EinoTools()
	require.NoError(t, err)
	require.Len(t, tools, 1)

	info, err := tools[0].Info(context.Background())
	require.NoError(t, err)
	require.NotNil(t, info.ParamsOneOf, "ParamsOneOf should not be nil")

	jsonSchema, err := info.ParamsOneOf.ToJSONSchema()
	require.NoError(t, err)
	require.NotNil(t, jsonSchema)
	require.NotNil(t, jsonSchema.Properties, "schema properties should not be nil")
	assert.Greater(t, jsonSchema.Properties.Len(), 0,
		"schema should include the tool's flag properties")
}
