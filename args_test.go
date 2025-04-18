package fisk

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgRemainder(t *testing.T) {
	app := New("test", "")
	v := app.Arg("test", "").Strings()
	args := []string{"hello", "world"}
	_, err := app.Parse(args)
	assert.NoError(t, err)
	assert.Equal(t, args, *v)
}

func TestArgRemainderErrorsWhenNotLast(t *testing.T) {
	a := newArgGroup()
	a.Arg("test", "").Strings()
	a.Arg("test2", "").String()
	assert.Error(t, a.init())
}

func TestArgMultipleRequired(t *testing.T) {
	terminated := false
	app := New("test", "")
	app.Version("0.0.0").Writer(io.Discard)
	app.Arg("a", "").Required().String()
	app.Arg("b", "").Required().String()
	app.Terminate(func(int) { terminated = true })

	_, err := app.Parse([]string{})
	assert.Error(t, err)
	_, err = app.Parse([]string{"A"})
	assert.Error(t, err)
	_, err = app.Parse([]string{"A", "B"})
	assert.NoError(t, err)
	_, _ = app.Parse([]string{"--version"})
	assert.True(t, terminated)
}

func TestInvalidArgsDefaultCanBeOverridden(t *testing.T) {
	app := New("test", "")
	app.Arg("a", "").Default("invalid").Bool()
	_, err := app.Parse([]string{})
	assert.Error(t, err)
}

func TestArgMultipleValuesDefault(t *testing.T) {
	app := New("test", "")
	a := app.Arg("a", "").Default("default1", "default2").Strings()
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"default1", "default2"}, *a)
}

func TestRequiredArgWithEnvarMissingErrors(t *testing.T) {
	app := newTestApp()
	app.Arg("t", "").Envar("TEST_ARG_ENVAR").Required().Int()
	_, err := app.Parse([]string{})
	assert.Error(t, err)
}

func TestArgRequiredWithEnvar(t *testing.T) {
	os.Setenv("TEST_ARG_ENVAR", "123")
	app := newTestApp()
	flag := app.Arg("t", "").Envar("TEST_ARG_ENVAR").Required().Int()
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, 123, *flag)
}

func TestSubcommandArgRequiredWithEnvar(t *testing.T) {
	os.Setenv("TEST_ARG_ENVAR", "123")
	app := newTestApp()
	cmd := app.Command("command", "")
	flag := cmd.Arg("t", "").Envar("TEST_ARG_ENVAR").Required().Int()
	_, err := app.Parse([]string{"command"})
	assert.NoError(t, err)
	assert.Equal(t, 123, *flag)
}

func TestArgIsSetByUser(t *testing.T) {
	app := newTestApp()
	var isSet bool
	var b bool
	app.Arg("b", "").IsSetByUser(&isSet).Required().BoolVar(&b)
	_, err := app.Parse([]string{"true"})
	assert.NoError(t, err)
	assert.True(t, b)
	assert.True(t, isSet)

	isSet = false
	b = false
	_, err = app.Parse([]string{})
	assert.Error(t, err)
	assert.False(t, b)
	assert.False(t, isSet)

	app = newTestApp()
	app.Arg("b", "").BoolVar(&b)
	isSet = false
	_, err = app.Parse([]string{"false"})
	assert.NoError(t, err)
	assert.False(t, b)
	assert.False(t, isSet)

	app = newTestApp()
	app.Arg("b", "").Default("false").BoolVar(&b)
	isSet = false
	_, err = app.Parse([]string{})
	assert.NoError(t, err)
	assert.False(t, b)
	assert.False(t, isSet)
}
