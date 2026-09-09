# Tool Execution

cobrax supports two execution models for running Cobra commands as MCP tools.

## Execution Models

### 1. Subprocess model (`Config` + `Command(config)`)

The legacy API re-invokes the same binary as a separate OS process for each tool
call. This works well when all state can be reconstructed from CLI flags alone.

### 2. In-process model (`MCPServer` + `NewMCPServer(opts, cmdFactory)`)

The server calls the command's `Run`/`RunE` function directly in-process. A
factory produces a fresh `*cobra.Command` tree per invocation, so every call
gets its own Options structs, flag values, and closure-captured variables with
no shared mutable state. This is the preferred model when commands depend on
non-serialisable state (API clients, database handles, business-logic objects).

The `MCPOptions.Subprocess` flag switches the `MCPServer` to run tools as a
subprocess instead, which survives MCP server termination.

## Execution Flow

1. **Middleware** (optional) - Wraps execution with custom logic
2. **Command Execution** - Runs the command in-process or as a subprocess, captures output

## Command Construction

MCP tool calls become CLI invocations. The flat input is split into flags and
positional arguments, then reconstructed:

**Input:**
```json
{
  "name": "kubectl_get_pods",
  "arguments": {
    "namespace": "production",
    "output": "json",
    "web-server": "web-server"
  }
}
```

**Constructed:**
```bash
/path/to/kubectl get pods --namespace production --output json web-server
```

**Flag conversion:**
- Boolean: `true` → `--flag`, `false` → omitted
- String/numeric: `--flag value`
- Arrays: `--flag a --flag b`
- Map (`stringToString`): `--flag key=value`
- Null/empty: omitted

## Output

All executions return:

```json
{
  "stdout": "command output...",
  "stderr": "error messages...",
  "exitCode": 0
}
```

Non-zero exit codes indicate command errors (not execution failures).

## Cancellation

The in-process model detaches the command's execution context from the caller's
cancellation (`context.WithoutCancel`), so a client tearing down its transport
does not abort in-flight work. The subprocess model likewise preserves the
subprocess across client cancellation.

## Transports

Commands can be exposed over several transports:

- `mcp start` — stdio (default for editor integration)
- `mcp stream` — SSE over HTTP
- `mcp rest` — plain JSON REST endpoints (`POST /{toolName}`)
