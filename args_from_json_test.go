package fisk

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// roundTripModel marshals the introspect model and unmarshals it again, exactly
// as --fisk-introspect does, so ArgsFromJSON is exercised against a model whose
// Values are nil and only the persisted type information survives.
func roundTripModel(t *testing.T, build func(app *Application)) *ApplicationModel {
	t.Helper()

	app := New("app", "an app")
	build(app)

	data, err := json.Marshal(app.introspectModel())
	require.NoError(t, err)

	var m ApplicationModel
	require.NoError(t, json.Unmarshal(data, &m))

	return &m
}

func findCmd(t *testing.T, m *ApplicationModel, name string) *CmdModel {
	t.Helper()

	for _, c := range m.Commands {
		if c.Name == name {
			return c
		}
	}

	t.Fatalf("command %q not found", name)
	return nil
}

func TestArgsFromJSON(t *testing.T) {
	t.Run("flags then positional args in declared order", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			cmd := app.Command("do", "do a thing")
			cmd.Flag("level", "log level").Enum("debug", "info", "warn")
			cmd.Flag("count", "how many").Int()
			cmd.Arg("subject", "the subject").Required().String()
			cmd.Arg("body", "the body").String()
		})

		args, err := findCmd(t, m, "do").ArgsFromJSON([]byte(`{"level":"info","count":3,"subject":"hello","body":"world"}`))
		require.NoError(t, err)
		require.Equal(t, []string{"--level=info", "--count=3", "--", "hello", "world"}, args)
	})

	t.Run("integers are not distorted by float conversion", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			app.Command("do", "do a thing").Flag("count", "how many").Int()
		})

		args, err := findCmd(t, m, "do").ArgsFromJSON([]byte(`{"count":9007199254740993}`))
		require.NoError(t, err)
		require.Equal(t, []string{"--count=9007199254740993"}, args)
	})

	t.Run("negatable boolean", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			app.Command("do", "do a thing").Flag("force", "force it").Bool()
		})
		cmd := findCmd(t, m, "do")

		on, err := cmd.ArgsFromJSON([]byte(`{"force":true}`))
		require.NoError(t, err)
		require.Equal(t, []string{"--force"}, on)

		off, err := cmd.ArgsFromJSON([]byte(`{"force":false}`))
		require.NoError(t, err)
		require.Equal(t, []string{"--no-force"}, off)
	})

	t.Run("un-negatable boolean", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			app.Command("do", "do a thing").Flag("yes", "say yes").UnNegatableBool()
		})
		cmd := findCmd(t, m, "do")

		on, err := cmd.ArgsFromJSON([]byte(`{"yes":true}`))
		require.NoError(t, err)
		require.Equal(t, []string{"--yes"}, on)

		off, err := cmd.ArgsFromJSON([]byte(`{"yes":false}`))
		require.NoError(t, err)
		require.Empty(t, off)
	})

	t.Run("un-negatable boolean defaulting true cannot be set false", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			app.Command("do", "do a thing").Flag("color", "use color").Default("true").UnNegatableBool()
		})

		_, err := findCmd(t, m, "do").ArgsFromJSON([]byte(`{"color":false}`))
		require.ErrorContains(t, err, "cannot be set to false")
	})

	t.Run("cumulative flag repeats", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			app.Command("do", "do a thing").Flag("tag", "a tag").Strings()
		})

		args, err := findCmd(t, m, "do").ArgsFromJSON([]byte(`{"tag":["a","b"]}`))
		require.NoError(t, err)
		require.Equal(t, []string{"--tag=a", "--tag=b"}, args)
	})

	t.Run("string map flag repeats as key=value sorted", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			app.Command("do", "do a thing").Flag("label", "a label").StringMap()
		})

		args, err := findCmd(t, m, "do").ArgsFromJSON([]byte(`{"label":{"b":"2","a":"1"}}`))
		require.NoError(t, err)
		require.Equal(t, []string{"--label=a=1", "--label=b=2"}, args)
	})

	t.Run("cumulative positional argument expands", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			app.Command("do", "do a thing").Arg("files", "the files").Strings()
		})

		args, err := findCmd(t, m, "do").ArgsFromJSON([]byte(`{"files":["f1","f2"]}`))
		require.NoError(t, err)
		require.Equal(t, []string{"--", "f1", "f2"}, args)
	})

	t.Run("unknown property is rejected", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			app.Command("do", "do a thing").Flag("level", "log level").String()
		})

		_, err := findCmd(t, m, "do").ArgsFromJSON([]byte(`{"nope":"x"}`))
		require.ErrorContains(t, err, `unknown property "nope"`)
	})

	t.Run("omitted positional with default fills the gap", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			cmd := app.Command("do", "do a thing")
			cmd.Arg("first", "first arg").Default("da").String()
			cmd.Arg("second", "second arg").String()
		})

		args, err := findCmd(t, m, "do").ArgsFromJSON([]byte(`{"second":"vb"}`))
		require.NoError(t, err)
		require.Equal(t, []string{"--", "da", "vb"}, args)
	})

	t.Run("omitted positional without default is an error when a later one is set", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			cmd := app.Command("do", "do a thing")
			cmd.Arg("first", "first arg").String()
			cmd.Arg("second", "second arg").String()
		})

		_, err := findCmd(t, m, "do").ArgsFromJSON([]byte(`{"second":"vb"}`))
		require.ErrorContains(t, err, `argument "first" must be set`)
	})

	t.Run("empty input yields no arguments", func(t *testing.T) {
		m := roundTripModel(t, func(app *Application) {
			app.Command("do", "do a thing").Flag("level", "log level").String()
		})

		args, err := findCmd(t, m, "do").ArgsFromJSON(nil)
		require.NoError(t, err)
		require.Empty(t, args)
	})
}

// TestArgsFromJSONCumulativeArgMustBeLast uses a hand-built model because the
// fluent builder does not allow declaring a cumulative argument before others.
func TestArgsFromJSONCumulativeArgMustBeLast(t *testing.T) {
	cmd := &CmdModel{
		Name: "do",
		ArgGroupModel: &ArgGroupModel{
			Args: []*ArgModel{
				{Name: "files", Cumulative: true},
				{Name: "trailing"},
			},
		},
	}

	_, err := cmd.ArgsFromJSON([]byte(`{"files":["a"]}`))
	require.ErrorContains(t, err, "cumulative but is not the last argument")
}
