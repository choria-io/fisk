package fisk

import (
	"bufio"
	"bytes"
	"fmt"
	"go/doc"
	"go/doc/comment"
	"io"
	"strings"
	"text/template"
)

var (
	preIndent = "  "
)

func formatTwoColumns(w io.Writer, indent, padding, width int, rows [][2]string) {
	max := int(float32(width) * 0.75 / 2)
	if max < 30 {
		max = 30
	}

	// Find size of first column.
	s := 0
	for _, row := range rows {
		if c := len(row[0]); c > s && c < max {
			s = c
		}
	}

	indentStr := strings.Repeat(" ", indent)
	offsetStr := strings.Repeat(" ", s+padding)

	for _, row := range rows {
		buf := bytes.NewBuffer(nil)
		d := new(doc.Package).Parser().Parse(row[1])
		pr := &comment.Printer{
			TextPrefix:     "",
			TextCodePrefix: preIndent,
			TextWidth:      width - s - padding - indent,
		}
		buf.Write(pr.Text(d))

		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
		fmt.Fprintf(w, "%s%-*s%*s", indentStr, s, row[0], padding, "")
		if len(row[0]) >= max {
			fmt.Fprintf(w, "\n%s%s", indentStr, offsetStr)
		}
		fmt.Fprintf(w, "%s\n", lines[0])
		for _, line := range lines[1:] {
			fmt.Fprintf(w, "%s%s%s\n", indentStr, offsetStr, line)
		}
	}
}

// Usage writes application usage to w. It parses args to determine
// appropriate help context, such as which command to show help for.
func (a *Application) Usage(args []string) {
	context, err := a.parseContext(true, args)
	a.FatalIfError(err, "")

	if err := a.UsageForContextWithTemplate(context, 2, a.usageTemplate); err != nil {
		panic(err)
	}
}

func formatAppUsage(app *ApplicationModel) string {
	s := []string{app.Name}
	if len(app.Flags) > 0 {
		s = append(s, app.FlagSummary())
	}
	if len(app.Args) > 0 {
		s = append(s, app.ArgSummary())
	}
	return strings.Join(s, " ")
}

func formatCmdUsage(app *ApplicationModel, cmd *CmdModel) string {
	s := []string{app.Name, cmd.String()}
	if len(cmd.Flags) > 0 {
		s = append(s, cmd.FlagSummary())
	}
	if len(cmd.Args) > 0 {
		s = append(s, cmd.ArgSummary())
	}
	return strings.Join(s, " ")
}

func formatFlag(haveShort bool, flag *FlagModel) string {
	flagString := ""
	flagName := flag.Name

	if flag.IsNegatable() {
		flagName = "[no-]" + flagName
	}

	if flag.Short != 0 {
		flagString += fmt.Sprintf("-%c, --%s", flag.Short, flagName)
	} else {
		if haveShort {
			flagString += fmt.Sprintf("    --%s", flagName)
		} else {
			flagString += fmt.Sprintf("--%s", flagName)
		}
	}
	if !flag.IsBoolFlag() {
		flagString += fmt.Sprintf("=%s", flag.FormatPlaceHolder())
	}
	if v, ok := flag.Value.(repeatableFlag); ok && v.IsCumulative() {
		flagString += " ..."
	}
	return flagString
}

type templateParseContext struct {
	SelectedCommand *CmdModel
	*FlagGroupModel
	*ArgGroupModel
}

type templateContext struct {
	App           *ApplicationModel
	HelpFlagIsSet bool
	Width         int
	Context       *templateParseContext
}

// UsageForContext displays usage information from a ParseContext (obtained from
// Application.ParseContext() or Action(f) callbacks).
func (a *Application) UsageForContext(context *ParseContext) error {
	return a.UsageForContextWithTemplate(context, 2, a.usageTemplate)
}

// UsageForContextWithTemplate is the base usage function. You generally don't need to use this.
func (a *Application) UsageForContextWithTemplate(context *ParseContext, indent int, tmpl string) error {
	width := guessWidth(a.usageWriter)
	funcs := template.FuncMap{
		"Join": func(items []string) string { return strings.Join(items, ", ") },
		"Indent": func(level int) string {
			return strings.Repeat(" ", level*indent)
		},
		"Wrap": func(indent int, s string) string {
			buf := bytes.NewBuffer(nil)
			indentText := strings.Repeat(" ", indent)

			d := new(doc.Package).Parser().Parse(s)
			pr := &comment.Printer{
				TextPrefix:     indentText,
				TextCodePrefix: "  " + indentText,
				TextWidth:      width - indent,
			}
			buf.Write(pr.Text(d))

			return buf.String()
		},
		"FormatFlag": formatFlag,
		"VisibleFlags": func(flags []*FlagModel) []*FlagModel {
			var vis []*FlagModel
			for _, flag := range flags {
				if !flag.Hidden {
					vis = append(vis, flag)
				}
			}

			return vis
		},
		"FlagsToTwoColumns": func(f []*FlagModel) [][2]string {
			rows := [][2]string{}
			haveShort := false
			for _, flag := range f {
				if flag.Short != 0 {
					haveShort = true
					break
				}
			}
			for _, flag := range f {
				if !flag.Hidden {
					rows = append(rows, [2]string{formatFlag(haveShort, flag), flag.HelpWithEnvar()})
				}
			}
			return rows
		},
		"GlobalFlags": func(c *templateParseContext) []*FlagModel {
			if c.SelectedCommand == nil {
				return c.Flags
			}

			var sflags []string
			for _, f := range c.SelectedCommand.Flags {
				sflags = append(sflags, f.Name)
			}

			var globals []*FlagModel
			for _, f := range c.Flags {
				var known bool
				for _, sf := range sflags {
					if f.Name == sf {
						known = true
						break
					}
				}

				if !known {
					globals = append(globals, f)
				}
			}

			return globals
		},
		"RequiredFlags": func(f []*FlagModel) []*FlagModel {
			requiredFlags := []*FlagModel{}
			for _, flag := range f {
				if flag.Required {
					requiredFlags = append(requiredFlags, flag)
				}
			}
			return requiredFlags
		},
		"OptionalFlags": func(f []*FlagModel) []*FlagModel {
			var optionalFlags []*FlagModel

			for _, flag := range f {
				if !flag.Required {
					optionalFlags = append(optionalFlags, flag)
				}
			}
			return optionalFlags
		},
		"CommandsToTwoColumns": func(c []*CmdModel) [][2]string {
			rows := [][2]string{}
			for _, cmd := range c {
				if !cmd.Hidden && cmd.FullCommand != "help" {
					shortHelp := strings.Split(cmd.Help, "\n")[0]
					rows = append(rows, [2]string{cmd.FullCommand, shortHelp})
				}
			}
			return rows
		},
		"ArgsToTwoColumns": func(a []*ArgModel) [][2]string {
			rows := [][2]string{}
			for _, arg := range a {
				if !arg.Hidden {
					var s string
					if arg.PlaceHolder != "" {
						s = arg.PlaceHolder
					} else {
						s = "<" + arg.Name + ">"
					}
					if !arg.Required {
						s = "[" + s + "]"
					}
					rows = append(rows, [2]string{s, arg.HelpWithEnvar()})
				}
			}
			return rows
		},
		"FormatTwoColumns": func(rows [][2]string) string {
			buf := bytes.NewBuffer(nil)
			formatTwoColumns(buf, indent, indent, width, rows)
			return buf.String()
		},
		"FormatTwoColumnsWithIndent": func(rows [][2]string, indent, padding int) string {
			buf := bytes.NewBuffer(nil)
			formatTwoColumns(buf, indent, padding, width, rows)
			return buf.String()
		},
		"FormatAppUsage":     formatAppUsage,
		"FormatCommandUsage": formatCmdUsage,
		"IsCumulative": func(value Value) bool {
			r, ok := value.(remainderArg)
			return ok && r.IsCumulative()
		},
		"Char": func(c rune) string {
			return string(c)
		},
		"FirstLine": func(v string) string {
			if v == "" {
				return v
			}

			scanner := bufio.NewScanner(strings.NewReader(v))
			scanner.Scan()
			return scanner.Text()
		},
	}
	for k, v := range a.usageFuncs {
		funcs[k] = v
	}

	t, err := template.New("usage").Funcs(funcs).Parse(tmpl)
	if err != nil {
		return err
	}
	var selectedCommand *CmdModel
	if context.SelectedCommand != nil {
		selectedCommand = context.SelectedCommand.Model()
	}
	ctx := templateContext{
		App:           a.Model(),
		Width:         width,
		HelpFlagIsSet: a.helpFlagIsSet,
		Context: &templateParseContext{
			SelectedCommand: selectedCommand,
			FlagGroupModel:  context.flags.Model(),
			ArgGroupModel:   context.arguments.Model(),
		},
	}
	return t.Execute(a.usageWriter, ctx)
}
