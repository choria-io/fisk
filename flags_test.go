package fisk

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBool(t *testing.T) {
	app := newTestApp()
	b := app.Flag("b", "").Bool()
	_, err := app.Parse([]string{"--b"})
	assert.NoError(t, err)
	assert.True(t, *b)
}

func TestNoBool(t *testing.T) {
	fg := newFlagGroup()
	f := fg.Flag("b", "").Default("true")
	b := f.Bool()
	fg.init("")
	tokens := tokenize([]string{"--no-b"}, false)
	_, err := fg.parse(tokens)
	assert.NoError(t, err)
	assert.False(t, *b)
}

func TestNegateNonBool(t *testing.T) {
	fg := newFlagGroup()
	f := fg.Flag("b", "")
	f.Int()
	fg.init("")
	tokens := tokenize([]string{"--no-b"}, false)
	_, err := fg.parse(tokens)
	assert.Error(t, err)
}

func TestNegativePrefixLongFlag(t *testing.T) {
	fg := newFlagGroup()
	f := fg.Flag("no-comment", "")
	b := f.Bool()
	fg.init("")
	tokens := tokenize([]string{"--no-comment"}, false)
	_, err := fg.parse(tokens)
	assert.NoError(t, err)
	assert.False(t, *b)
}

func TestInvalidFlagDefaultCanBeOverridden(t *testing.T) {
	app := newTestApp()
	app.Flag("a", "").Default("invalid").Bool()
	_, err := app.Parse([]string{})
	assert.Error(t, err)
}

func TestRequiredFlag(t *testing.T) {
	app := newTestApp()
	app.Version("0.0.0").Writer(io.Discard)
	exits := 0
	app.Terminate(func(int) { exits++ })
	app.Flag("a", "").Required().Bool()
	_, err := app.Parse([]string{"--a"})
	assert.NoError(t, err)
	_, err = app.Parse([]string{})
	assert.Error(t, err)
	_, _ = app.Parse([]string{"--version"})
	assert.Equal(t, 1, exits)
}

func TestShortFlag(t *testing.T) {
	app := newTestApp()
	f := app.Flag("long", "").Short('s').Bool()
	_, err := app.Parse([]string{"-s"})
	assert.NoError(t, err)
	assert.True(t, *f)
}

func TestUnicodeShortFlag(t *testing.T) {
	app := newTestApp()
	f := app.Flag("aaa", "").Short('ä').Bool()
	_, err := app.Parse([]string{"-ä"})
	assert.NoError(t, err)
	assert.True(t, *f)
}

func TestCombinedShortFlags(t *testing.T) {
	app := newTestApp()
	a := app.Flag("short0", "").Short('0').Bool()
	b := app.Flag("short1", "").Short('1').Bool()
	c := app.Flag("short2", "").Short('2').Bool()
	_, err := app.Parse([]string{"-01"})
	assert.NoError(t, err)
	assert.True(t, *a)
	assert.True(t, *b)
	assert.False(t, *c)
}

func TestCombinedUnicodeShortFlags(t *testing.T) {
	app := newTestApp()
	a := app.Flag("short0", "").Short('0').Bool()
	b := app.Flag("short1", "").Short('1').Bool()
	c := app.Flag("short2", "").Short('ä').Bool()
	d := app.Flag("short3", "").Short('2').Bool()
	_, err := app.Parse([]string{"-0ä1"})
	assert.NoError(t, err)
	assert.True(t, *a)
	assert.True(t, *b)
	assert.True(t, *c)
	assert.False(t, *d)
}

func TestCombinedShortFlagArg(t *testing.T) {
	a := newTestApp()
	n := a.Flag("short", "").Short('s').Int()
	_, err := a.Parse([]string{"-s10"})
	assert.NoError(t, err)
	assert.Equal(t, 10, *n)
}

func TestCombinedUnicodeShortFlagArg(t *testing.T) {
	app := newTestApp()
	a := app.Flag("short", "").Short('ä').Int()
	_, err := app.Parse([]string{"-ä10"})
	assert.NoError(t, err)
	assert.Equal(t, 10, *a)
}

func TestCombinedUnicodeShortFlagUnicodeArg(t *testing.T) {
	app := newTestApp()
	a := app.Flag("short", "").Short('ä').String()
	_, err := app.Parse([]string{"-äöö"})
	assert.NoError(t, err)
	assert.Equal(t, "öö", *a)
}

func TestEmptyShortFlagIsAnError(t *testing.T) {
	_, err := newTestApp().Parse([]string{"-"})
	assert.Error(t, err)
}

func TestRequiredWithEnvarMissingErrors(t *testing.T) {
	app := newTestApp()
	app.Flag("t", "").Envar("TEST_ENVAR").Required().Int()
	_, err := app.Parse([]string{})
	assert.Error(t, err)
}

func TestRequiredWithEnvar(t *testing.T) {
	os.Setenv("TEST_ENVAR", "123")
	app := newTestApp()
	flag := app.Flag("t", "").Envar("TEST_ENVAR").Required().Int()
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, 123, *flag)
}

func TestSubcommandFlagRequiredWithEnvar(t *testing.T) {
	os.Setenv("TEST_ENVAR", "123")
	app := newTestApp()
	cmd := app.Command("command", "")
	flag := cmd.Flag("t", "").Envar("TEST_ENVAR").Required().Int()
	_, err := app.Parse([]string{"command"})
	assert.NoError(t, err)
	assert.Equal(t, 123, *flag)
}

func TestRegexp(t *testing.T) {
	app := newTestApp()
	flag := app.Flag("reg", "").Regexp()
	_, err := app.Parse([]string{"--reg", "^abc$"})
	assert.NoError(t, err)
	assert.NotNil(t, *flag)
	assert.Equal(t, "^abc$", (*flag).String())
	assert.Regexp(t, *flag, "abc")
	assert.NotRegexp(t, *flag, "abcd")
}

func TestDuplicateShortFlag(t *testing.T) {
	app := newTestApp()
	app.Flag("a", "").Short('a').String()
	app.Flag("b", "").Short('a').String()
	_, err := app.Parse([]string{})
	assert.Error(t, err)
}

func TestDuplicateLongFlag(t *testing.T) {
	app := newTestApp()
	app.Flag("a", "").String()
	app.Flag("a", "").String()
	_, err := app.Parse([]string{})
	assert.Error(t, err)
}

func TestGetFlagAndOverrideDefault(t *testing.T) {
	app := newTestApp()
	a := app.Flag("a", "").Default("default").String()
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "default", *a)
	app.GetFlag("a").Default("new")
	_, err = app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "new", *a)
}

func TestEnvarOverrideDefault(t *testing.T) {
	os.Setenv("TEST_ENVAR", "123")
	app := newTestApp()
	flag := app.Flag("t", "").Default("default").Envar("TEST_ENVAR").String()
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "123", *flag)
}

func TestFlagMultipleValuesDefault(t *testing.T) {
	app := newTestApp()
	a := app.Flag("a", "").Default("default1", "default2").Strings()
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"default1", "default2"}, *a)
}

func TestFlagMultipleValuesDefaultNonRepeatable(t *testing.T) {
	c := newTestApp()
	c.Flag("foo", "foo").Default("a", "b").String()
	_, err := c.Parse([]string{})
	assert.Error(t, err)
}

func TestFlagMultipleValuesDefaultEnvarUnix(t *testing.T) {
	app := newTestApp()
	a := app.Flag("a", "").Envar("TEST_MULTIPLE_VALUES").Strings()
	os.Setenv("TEST_MULTIPLE_VALUES", "123\n456\n")
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"123", "456"}, *a)
}

func TestFlagMultipleValuesDefaultEnvarWindows(t *testing.T) {
	app := newTestApp()
	a := app.Flag("a", "").Envar("TEST_MULTIPLE_VALUES").Strings()
	os.Setenv("TEST_MULTIPLE_VALUES", "123\r\n456\r\n")
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"123", "456"}, *a)
}

func TestFlagMultipleValuesDefaultEnvarNonRepeatable(t *testing.T) {
	c := newTestApp()
	a := c.Flag("foo", "foo").Envar("TEST_MULTIPLE_VALUES_NON_REPEATABLE").String()
	os.Setenv("TEST_MULTIPLE_VALUES_NON_REPEATABLE", "123\n456")
	_, err := c.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "123\n456", *a)
}

func TestFlagHintAction(t *testing.T) {
	c := newTestApp()

	action := func() []string {
		return []string{"opt1", "opt2"}
	}

	a := c.Flag("foo", "foo").HintAction(action)
	args := a.resolveCompletions()
	assert.Equal(t, []string{"opt1", "opt2"}, args)
}

func TestFlagHintOptions(t *testing.T) {
	c := newTestApp()

	a := c.Flag("foo", "foo").HintOptions("opt1", "opt2")
	args := a.resolveCompletions()
	assert.Equal(t, []string{"opt1", "opt2"}, args)
}

func TestFlagEnumVar(t *testing.T) {
	c := newTestApp()
	var bar string

	a := c.Flag("foo", "foo")
	a.Enum("opt1", "opt2")
	b := c.Flag("bar", "bar")
	b.EnumVar(&bar, "opt3", "opt4")

	args := a.resolveCompletions()
	assert.Equal(t, []string{"opt1", "opt2"}, args)

	args = b.resolveCompletions()
	assert.Equal(t, []string{"opt3", "opt4"}, args)
}

func TestMultiHintOptions(t *testing.T) {
	c := newTestApp()

	a := c.Flag("foo", "foo").HintOptions("opt1").HintOptions("opt2")
	args := a.resolveCompletions()
	assert.Equal(t, []string{"opt1", "opt2"}, args)
}
func TestMultiHintActions(t *testing.T) {
	c := newTestApp()

	a := c.Flag("foo", "foo").
		HintAction(func() []string {
			return []string{"opt1"}
		}).
		HintAction(func() []string {
			return []string{"opt2"}
		})
	args := a.resolveCompletions()
	assert.Equal(t, []string{"opt1", "opt2"}, args)
}

func TestCombinationHintActionsOptions(t *testing.T) {
	c := newTestApp()

	a := c.Flag("foo", "foo").HintAction(func() []string {
		return []string{"opt1"}
	}).HintOptions("opt2")
	args := a.resolveCompletions()
	assert.Equal(t, []string{"opt1", "opt2"}, args)
}

func TestCombinationEnumActions(t *testing.T) {
	c := newTestApp()
	var foo string

	a := c.Flag("foo", "foo").
		HintAction(func() []string {
			return []string{"opt1", "opt2"}
		})
	a.Enum("opt3", "opt4")

	b := c.Flag("bar", "bar").
		HintAction(func() []string {
			return []string{"opt5", "opt6"}
		})
	b.EnumVar(&foo, "opt3", "opt4")

	// Provided HintActions should override automatically generated Enum options.
	args := a.resolveCompletions()
	assert.Equal(t, []string{"opt1", "opt2"}, args)

	args = b.resolveCompletions()
	assert.Equal(t, []string{"opt5", "opt6"}, args)
}

func TestCombinationEnumOptions(t *testing.T) {
	c := newTestApp()
	var foo string

	a := c.Flag("foo", "foo").HintOptions("opt1", "opt2")
	a.Enum("opt3", "opt4")

	b := c.Flag("bar", "bar").HintOptions("opt5", "opt6")
	b.EnumVar(&foo, "opt3", "opt4")

	// Provided HintOptions should override automatically generated Enum options.
	args := a.resolveCompletions()
	assert.Equal(t, []string{"opt1", "opt2"}, args)

	args = b.resolveCompletions()
	assert.Equal(t, []string{"opt5", "opt6"}, args)

}

func TestFlagIsSetByUser(t *testing.T) {
	app := newTestApp()
	var isSet bool
	b := app.Flag("b", "").IsSetByUser(&isSet).Bool()
	_, err := app.Parse([]string{"--b"})
	assert.NoError(t, err)
	assert.True(t, *b)
	assert.True(t, isSet)

	isSet = false
	_, err = app.Parse([]string{"--no-b"})
	assert.NoError(t, err)
	assert.False(t, *b)
	assert.True(t, isSet)

	isSet2 := true
	isSet = false
	_ = app.Flag("b2", "").IsSetByUser(&isSet2).Bool()
	_, err = app.Parse([]string{"--b", "--unknown"})
	assert.Error(t, err)
	assert.True(t, isSet)
	assert.False(t, isSet2)
}

func TestValidator(t *testing.T) {
	regexpValidator := func(r string) OptionValidator {
		return func(v string) error {
			ok, err := regexp.MatchString(r, v)
			if err != nil {
				return err
			}

			if !ok {
				return fmt.Errorf("does not validate using %q", r)
			}

			return nil
		}
	}

	app := newTestApp()

	arg := app.Arg("arg", "A arg").Default("a").Validator(regexpValidator("^[abc]$")).String()
	flag := app.Flag("flag", "A flag").Validator(regexpValidator("^[xyz]$")).String()

	_, err := app.Parse([]string{"--flag", "x"})
	assert.NoError(t, err)
	assert.Equal(t, *flag, "x")
	assert.Equal(t, *arg, "a")

	*arg = ""
	*flag = ""
	_, err = app.Parse([]string{"b", "--flag", "x"})
	assert.NoError(t, err)
	assert.Equal(t, *flag, "x")
	assert.Equal(t, *arg, "b")

	*arg = ""
	*flag = ""
	_, err = app.Parse([]string{"z", "--flag", "x"})
	assert.Error(t, err, `does not validate using "^[abc]$"`)
	assert.Equal(t, *flag, "")
	assert.Equal(t, *arg, "")
}

func TestNegatableBool(t *testing.T) {
	app := newTestApp()
	boolFlag := app.Flag("neg", "").Bool()
	unNegBoolFlag := app.Flag("unneg", "").UnNegatableBool()

	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.False(t, *unNegBoolFlag)
	assert.False(t, *boolFlag)

	_, err = app.Parse([]string{"--neg"})
	assert.NoError(t, err)
	assert.True(t, *boolFlag)
	assert.False(t, *unNegBoolFlag)

	_, err = app.Parse([]string{"--no-neg"})
	assert.NoError(t, err)
	assert.False(t, *boolFlag)
	assert.False(t, *unNegBoolFlag)

	_, err = app.Parse([]string{"--unneg"})
	assert.NoError(t, err)
	assert.False(t, *boolFlag)
	assert.True(t, *unNegBoolFlag)

	_, err = app.Parse([]string{"--no-unneg"})
	assert.Error(t, err, "unknown long flag '--no-unneg'")
}

func TestHelpForBoolFlags(t *testing.T) {
	app := newTestApp()
	app.Flag("yes", "").Default("true").Bool()
	app.Flag("no", "").Default("false").Bool()
	app.Flag("nonneg", "").UnNegatableBool()

	w := bytes.NewBuffer(nil)
	app.Writer(w).Usage(nil)
	assert.Contains(t, w.String(), "--[no-]yes")
	assert.Contains(t, w.String(), "--[no-]no")
	assert.Contains(t, w.String(), "--nonneg")
}
