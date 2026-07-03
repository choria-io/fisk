package fisk

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIntrospectModelSchemaRoundTrip pins the fix for the bug where Schema /
// RestrictedSchema degraded to all-string after the introspect JSON round-trip:
// the Value-derived type information must survive because IntrospectModel
// precomputes the schemas while the Values are still live.
func TestIntrospectModelSchemaRoundTrip(t *testing.T) {
	app := New("app", "an app")
	set := app.Command("set", "set a thing")
	set.Flag("level", "log level").Enum("debug", "info", "warn")
	set.Flag("count", "how many").Int()
	set.Arg("ttl", "time to live").Duration()

	// round-trip exactly as --fisk-introspect does: marshal the introspect model, ship, unmarshal.
	data, err := json.Marshal(app.introspectModel())
	require.NoError(t, err)

	var m ApplicationModel
	require.NoError(t, json.Unmarshal(data, &m))

	var cmd *CmdModel
	for _, c := range m.Commands {
		if c.Name == "set" {
			cmd = c
		}
	}
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.RestrictedSchema, "RestrictedSchema must be populated and survive the round-trip")

	props, ok := cmd.RestrictedSchema["properties"].(map[string]any)
	require.True(t, ok)

	level, ok := props["level"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "string", level["type"])
	require.ElementsMatch(t, []any{"debug", "info", "warn"}, level["enum"], "enum options must survive")

	count, ok := props["count"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "integer", count["type"], "integer type must survive (not collapse to string)")

	ttl, ok := props["ttl"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "string", ttl["type"])
	require.Contains(t, ttl, "pattern", "duration pattern must survive")

	// the full Schema is also populated
	require.NotNil(t, cmd.Schema)
}

// TestIntrospectModelPerCommandCheats pins that a cheat set on a specific command
// is emitted on that command's model (and survives the JSON round-trip) only when
// the cheat label matches the command name; a differently-labeled cheat stays in
// the application-level Cheats, which remains the merged set across all commands.
func TestIntrospectModelPerCommandCheats(t *testing.T) {
	app := New("app", "an app")
	app.Cheat("", "# top cheat")

	// label matches the command name -> attributed to the command
	app.Command("matching", "a command").Cheat("matching", "# matching cheat")

	// empty label defaults to the command name -> attributed to the command
	app.Command("named", "a command").Cheat("", "# named cheat")

	// label matches one of the command's aliases -> attributed to the command
	aliased := app.Command("aliased", "a command")
	aliased.Alias("al")
	aliased.Cheat("al", "# aliased cheat")

	// nested command whose label matches its name -> attributed
	sub := app.Command("sub", "a sub command")
	sub.Command("leaf", "a leaf").Cheat("leaf", "# leaf cheat")

	// label differs from the command name -> global only, not on the command
	app.Command("mismatch", "a command").Cheat("other", "# other cheat")

	app.Command("plain", "no cheat here")

	data, err := json.Marshal(app.introspectModel())
	require.NoError(t, err)

	var m ApplicationModel
	require.NoError(t, json.Unmarshal(data, &m))

	byName := map[string]*CmdModel{}
	for _, c := range m.Commands {
		byName[c.Name] = c
	}

	// a matching label is attributed to the command
	require.NotNil(t, byName["matching"])
	require.Equal(t, "# matching cheat", byName["matching"].Cheat)

	// an empty label defaults to the command name and is attributed
	require.NotNil(t, byName["named"])
	require.Equal(t, "# named cheat", byName["named"].Cheat)

	// a label matching an alias is attributed to the command
	require.NotNil(t, byName["aliased"])
	require.Equal(t, "# aliased cheat", byName["aliased"].Cheat)

	// nested commands too
	var leaf *CmdModel
	for _, c := range byName["sub"].Commands {
		if c.Name == "leaf" {
			leaf = c
		}
	}
	require.NotNil(t, leaf)
	require.Equal(t, "# leaf cheat", leaf.Cheat)

	// a mismatched label is NOT attributed to the command
	require.NotNil(t, byName["mismatch"])
	require.Empty(t, byName["mismatch"].Cheat)

	// a command with no cheat omits the field entirely
	require.NotNil(t, byName["plain"])
	require.Empty(t, byName["plain"].Cheat)

	// the application-level set still contains every cheat (merged), including
	// the mismatched one under its own label
	require.Equal(t, "# top cheat", m.Cheats["app"])
	require.Equal(t, "# matching cheat", m.Cheats["matching"])
	require.Equal(t, "# named cheat", m.Cheats["named"])
	require.Equal(t, "# aliased cheat", m.Cheats["al"])
	require.Equal(t, "# leaf cheat", m.Cheats["leaf"])
	require.Equal(t, "# other cheat", m.Cheats["other"])
}

// TestRestrictedSchemaOmitsOneOf pins that the Anthropic-restricted schema does
// not emit oneOf for IP-typed values: Anthropic's strict schema subset rejects
// oneOf, so the ipv4/ipv6 alternative belongs only in the unrestricted Schema.
func TestRestrictedSchemaOmitsOneOf(t *testing.T) {
	app := New("app", "an app")
	set := app.Command("set", "set a thing")
	set.Flag("addr", "an address").IP()

	data, err := json.Marshal(app.introspectModel())
	require.NoError(t, err)

	var m ApplicationModel
	require.NoError(t, json.Unmarshal(data, &m))

	var cmd *CmdModel
	for _, c := range m.Commands {
		if c.Name == "set" {
			cmd = c
		}
	}
	require.NotNil(t, cmd)

	restricted, ok := cmd.RestrictedSchema["properties"].(map[string]any)
	require.True(t, ok)
	addr, ok := restricted["addr"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "string", addr["type"])
	require.NotContains(t, addr, "oneOf", "restricted schema must not contain oneOf")

	// the unrestricted schema keeps the ipv4/ipv6 alternative
	full, ok := cmd.Schema["properties"].(map[string]any)
	require.True(t, ok)
	fullAddr, ok := full["addr"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, fullAddr, "oneOf", "unrestricted schema must keep oneOf")
}
