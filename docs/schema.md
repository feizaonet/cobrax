# Schema Generation

cobrax automatically generates JSON schemas for MCP tools from Cobra commands.

## Tool Properties

- **Name**: Command path with underscores (`kubectl_get_pods`)
- **Description**: From command's Long, Short, and Example fields
- **Input Schema**: Flat — all flags and positional arguments appear as direct top-level properties
- **Output Schema**: Standard format (stdout, stderr, exitCode)

## Input Schema

The input schema is **flat**: every flag and positional argument is a direct
property of the top-level object. There are no nested `flags` or `args`
sub-objects.

```json
{
  "type": "object",
  "properties": {
    "namespace": { "type": "string", "description": "Kubernetes namespace", "default": "default" },
    "replicas": { "type": "integer", "default": 3 },
    "output": { "type": "string" },
    "module": { "type": "string", "description": "The module value (required)" }
  },
  "required": ["module"]
}
```

### Flags

Each flag becomes a top-level property. The type mapping is:

| pflag type | JSON Schema type |
|---|---|
| `bool` | `boolean` |
| `int`, `uint`, `count` (all widths) | `integer` |
| `float32`, `float64` | `number` |
| `string` | `string` |
| `string` + `jsonschema` annotation | the annotation's schema |
| `stringSlice`, `stringArray` | `array` of `string` |
| `intSlice`, `int32Slice`, `int64Slice`, `uintSlice` | `array` of `integer` |
| `float32Slice`, `float64Slice` | `array` of `number` |
| `boolSlice` | `array` of `boolean` |
| `durationSlice`, `ipSlice`, `ipNetSlice` | `array` of `string` |
| `stringToString` | `object` with string values |
| `stringToInt`, `stringToInt64` | `object` with integer values |
| `duration`, `ip`, `ipMask`, `ipNet`, `bytesHex`, `bytesBase64` | `string` with pattern validation |

### Required flags

Flags marked as required (via `cmd.MarkFlagRequired()` /
`cmd.MarkPersistentFlagRequired()`) are added to the schema's `required` array.

### Defaults

Default values are included in the schema from the flag's `DefValue`, except for:

- empty strings (`""`)
- empty arrays (`[]`)

### JSON schema annotation

A string flag annotated with the `jsonschema` key uses that JSON schema instead
of the plain `string` type. If the annotation schema does not carry its own
`description`, the flag's usage string is used.

```go
type SomeJSONObject struct {
    Foo    string
    Bar    int
}

schema, _ := jsonschema.For[SomeJSONObject](nil)
bytes, _ := schema.MarshalJSON()

cmd.Flags().String("a_json_obj", "", "Some JSON Object")
obj := cmd.Flags().Lookup("a_json_obj")
obj.Annotations = map[string][]string{
    "jsonschema": {string(bytes)},
}
```

### Positional arguments

Positional arguments parsed from `cmd.Use` also become top-level properties.
Angle-bracket tokens (`<name>`) are required; square-bracket tokens (`[name]`)
are optional; a trailing `...` marks a variadic argument (an array of strings).

Argument descriptions come from the `cobrax.arg.<index>` annotation when present.

## Output Schema

```json
{
  "type": "object",
  "properties": {
    "stdout": { "type": "string" },
    "stderr": { "type": "string" },
    "exitCode": { "type": "integer" }
  }
}
```

## Export Schemas

```bash
./my-cli mcp tools  # Creates mcp-tools.json
```
