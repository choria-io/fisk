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
