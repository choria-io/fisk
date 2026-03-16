package fisk

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestFormatTwoColumns(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	formatTwoColumns(buf, 2, 2, 20, [][2]string{
		{"--hello", "Hello world help with something that is cool."},
	})
	expected := `  --hello  Hello
           world
           help with
           something
           that is
           cool.
`
	assert.Equal(t, expected, buf.String())
}

func TestFormatTwoColumnsWide(t *testing.T) {
	samples := [][2]string{
		{strings.Repeat("x", 29), "29 chars"},
		{strings.Repeat("x", 30), "30 chars"}}
	buf := bytes.NewBuffer(nil)
	formatTwoColumns(buf, 0, 0, 80, samples)
	expected := `xxxxxxxxxxxxxxxxxxxxxxxxxxxxx29 chars
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
                             30 chars
`
	assert.Equal(t, expected, buf.String())
}

func TestHiddenCommand(t *testing.T) {
	templates := []struct{ name, template string }{
		{"default", KingpinDefaultUsageTemplate},
		{"Compact", CompactUsageTemplate},
		{"Long", LongHelpTemplate},
		{"Man", ManPageTemplate},
		{"LLM", LLMHelpTemplate},
	}

	var buf bytes.Buffer
	t.Log("1")

	a := New("test", "Test").Writer(&buf).Terminate(nil)
	a.Command("visible", "visible")
	a.Command("hidden", "hidden").Hidden()

	for _, tp := range templates {
		buf.Reset()
		a.UsageTemplate(tp.template)
		a.Parse(nil)
		// a.Parse([]string{"--help"})
		usage := buf.String()
		t.Logf("Usage for %s is:\n%s\n", tp.name, usage)

		assert.NotContains(t, usage, "hidden")
		assert.Contains(t, usage, "visible")
	}
}

func TestUsageFuncs(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").Writer(&buf).Terminate(nil)
	tpl := `{{ add 2 1 }}`
	a.UsageTemplate(tpl)
	a.UsageFuncs(template.FuncMap{
		"add": func(x, y int) int { return x + y },
	})
	a.Parse([]string{"--help"})
	usage := buf.String()
	assert.Equal(t, "3", usage)
}

func TestCmdClause_HelpLong(t *testing.T) {
	var buf bytes.Buffer
	tpl := `{{define "FormatUsage"}}{{.HelpLong}}{{end -}}
{{template "FormatUsage" .Context.SelectedCommand}}`

	a := New("test", "Test").Writer(&buf).Terminate(nil).UsageTemplate(KingpinDefaultUsageTemplate)
	a.UsageTemplate(tpl)
	a.Command("command", "short help text").HelpLong("long help text")

	a.Parse([]string{"command", "--help"})
	usage := buf.String()
	assert.Equal(t, "long help text", usage)
}

func TestArgEnvVar(t *testing.T) {
	var buf bytes.Buffer

	a := New("test", "Test").Writer(&buf).Terminate(nil).UsageTemplate(KingpinDefaultUsageTemplate)
	a.Arg("arg", "Enable arg").Envar("ARG").String()
	a.Flag("flag", "Enable flag").Envar("FLAG").String()

	a.Parse([]string{"command", "--help"})
	usage := buf.String()
	assert.Contains(t, usage, "($ARG)")
	assert.Contains(t, usage, "($FLAG)")
}

func TestLLMHelpTemplate(t *testing.T) {
	var buf bytes.Buffer

	a := New("test", "Test application").UsageWriter(&buf).Terminate(nil)
	a.Version("1.0.0")

	sub := a.Command("subscribe", "Generic subscription client").Alias("sub").Tag("read-only")
	sub.HelpLong("Extended help about subscribe command.")
	sub.Arg("subjects", "Subjects to subscribe to").Strings()
	sub.Flag("queue", "Subscribe to a named queue group").String()
	sub.Flag("raw", "Show the raw data received").Short('r').Bool()
	sub.Flag("count", "Quit after receiving this many messages").Int()
	sub.Flag("server", "NATS server urls").Envar("NATS_URL").PlaceHolder("URL").String()
	sub.Flag("timeout", "Time to wait on responses").Default("5s").Duration()

	a.UsageTemplate(LLMHelpTemplate)

	// Test top-level help
	a.Parse([]string{"--help"})
	usage := buf.String()
	t.Logf("Top-level LLM help:\n%s", usage)
	assert.Contains(t, usage, "# test")
	assert.Contains(t, usage, "Test application")
	assert.Contains(t, usage, "## Commands")
	assert.Contains(t, usage, "`subscribe`")
	assert.NotContains(t, usage, "hidden")

	// Test command-level help
	buf.Reset()
	a.Parse([]string{"subscribe", "--help"})
	usage = buf.String()
	t.Logf("Command LLM help:\n%s", usage)
	assert.Contains(t, usage, "# test subscribe")
	assert.Contains(t, usage, "Generic subscription client")
	assert.Contains(t, usage, "Extended help about subscribe command.")
	assert.Contains(t, usage, "**Tags:** read-only")
	assert.Contains(t, usage, "**Aliases:** sub")
	assert.Contains(t, usage, "## Arguments")
	assert.Contains(t, usage, "`subjects`")
	assert.Contains(t, usage, "## Flags")
	assert.Contains(t, usage, "`--queue`")
	assert.Contains(t, usage, "`--[no-]raw`, `-r`")
	assert.Contains(t, usage, "`int`")
	assert.Contains(t, usage, "`duration`")
	assert.Contains(t, usage, "`5s`")
	assert.Contains(t, usage, "`NATS_URL`")
	assert.Contains(t, usage, "## Global Flags")
}

func TestLLMHelpTemplateTags(t *testing.T) {
	var buf bytes.Buffer

	a := New("test", "Test application").UsageWriter(&buf).Terminate(nil)
	a.Command("read", "Read data").Tag("read-only")
	a.Command("write", "Write data").Tag("makes-changes")
	a.Command("delete", "Delete data").Tag("destructive")

	a.UsageTemplate(LLMHelpTemplate)

	a.Parse([]string{"--help"})
	usage := buf.String()
	t.Logf("Tags LLM help:\n%s", usage)
	assert.Contains(t, usage, "Tags")
	assert.Contains(t, usage, "read-only")
	assert.Contains(t, usage, "makes-changes")
	assert.Contains(t, usage, "destructive")
}

func TestLLMHelpTemplateNoTags(t *testing.T) {
	var buf bytes.Buffer

	a := New("test", "Test application").UsageWriter(&buf).Terminate(nil)
	a.Command("read", "Read data")
	a.Command("write", "Write data")

	a.UsageTemplate(LLMHelpTemplate)

	a.Parse([]string{"--help"})
	usage := buf.String()
	t.Logf("No Tags LLM help:\n%s", usage)
	assert.NotContains(t, usage, "Tags")
}

func TestLLMExtraInfoInLLMTemplate(t *testing.T) {
	var buf bytes.Buffer

	a := New("test", "Test application").UsageWriter(&buf).Terminate(nil)
	a.LLMExtraInformation("This tool supports tags: read-only, makes-changes, destructive.\nUse LLMFORMAT=1 for markdown help.")
	a.Command("read", "Read data")

	a.UsageTemplate(LLMHelpTemplate)

	a.Parse([]string{"--help"})
	usage := buf.String()
	t.Logf("LLM extra info:\n%s", usage)
	assert.Contains(t, usage, "## Additional Information")
	assert.Contains(t, usage, "This tool supports tags")
	assert.Contains(t, usage, "LLMFORMAT=1")
}

func TestLLMExtraInfoInCompactTemplate(t *testing.T) {
	var buf bytes.Buffer

	a := New("test", "Test application").UsageWriter(&buf).Terminate(nil)
	a.LLMExtraInformation("Use LLMFORMAT=1 for LLM-friendly help.")
	a.Command("read", "Read data")

	// Without CLAUDECODE=1, extra info should not appear
	t.Setenv("CLAUDECODE", "")
	a.Parse([]string{"--help"})
	usage := buf.String()
	t.Logf("Compact without CLAUDECODE:\n%s", usage)
	assert.NotContains(t, usage, "LLM Information")

	// With CLAUDECODE=1, extra info should appear
	buf.Reset()
	t.Setenv("CLAUDECODE", "1")
	a.Parse([]string{"--help"})
	usage = buf.String()
	t.Logf("Compact with CLAUDECODE:\n%s", usage)
	assert.Contains(t, usage, "LLM Information")
	assert.Contains(t, usage, "LLMFORMAT=1")
}

func TestLLMExtraInfoDefaultHint(t *testing.T) {
	var buf bytes.Buffer

	a := New("test", "Test application").UsageWriter(&buf).Terminate(nil)
	a.Command("read", "Read data")

	// With CLAUDECODE=1, even without explicit LLMExtraInformation(), the default hint shows
	t.Setenv("CLAUDECODE", "1")
	a.Parse([]string{"--help"})
	usage := buf.String()
	assert.Contains(t, usage, "LLM Information")
	assert.Contains(t, usage, "--help-llm")
	assert.Contains(t, usage, "LLMFORMAT=1")

	// Without CLAUDECODE, no hint
	buf.Reset()
	t.Setenv("CLAUDECODE", "")
	a.Parse([]string{"--help"})
	usage = buf.String()
	assert.NotContains(t, usage, "LLM Information")
}

func TestShortMainUSage(t *testing.T) {
	var buf bytes.Buffer

	a := New("test", "Test Command").UsageWriter(&buf).Terminate(nil)
	sub := a.Command("sub", "Sub command").HelpLong("sub long help")
	sub.Command("subsub1", "Subsub2 command").HelpLong("subsub1 long help")
	sub.Command("subsub2", "Subsub2 command")

	a.UsageTemplate(ShorterMainUsageTemplate)

	a.Parse([]string{"--help"})
	assert.NotContains(t, buf.String(), "long help")
	assert.NotContains(t, buf.String(), "Flags:")

	buf.Reset()
	a.Parse([]string{"sub", "--help"})
	assert.Contains(t, buf.String(), "sub long help")
	assert.NotContains(t, buf.String(), "subsub1 long help")
	assert.NotContains(t, buf.String(), "subsub2 long help")

	buf.Reset()
	a.Parse([]string{"sub", "subsub1", "--help"})
	assert.NotContains(t, buf.String(), "sub long help")
	assert.Contains(t, buf.String(), "subsub1 long help")
	assert.NotContains(t, buf.String(), "subsub2 long help")
}
