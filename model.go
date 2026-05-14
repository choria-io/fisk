package fisk

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Data model for Fisk command-line structure.

var (
	ignoreInCount = map[string]bool{
		"help":                   true,
		"help-long":              true,
		"help-man":               true,
		"completion-bash":        true,
		"completion-script-bash": true,
		"completion-script-zsh":  true,
		"help-llm":               true,
		"help-compact":           true,
		"fisk-introspect":        true,
	}
)

type FlagGroupModel struct {
	Flags []*FlagModel `json:"flags,omitempty"`
}

func (f *FlagGroupModel) FlagSummary() string {
	out := []string{}
	count := 0

	for _, flag := range f.Flags {
		if !ignoreInCount[flag.Name] {
			count++
		}

		if flag.Required {
			if flag.IsBoolFlag() {
				if flag.IsNegatable() {
					out = append(out, fmt.Sprintf("--[no-]%s", flag.Name))
				} else {
					out = append(out, fmt.Sprintf("--%s=%s", flag.Name, flag.FormatPlaceHolder()))
				}
			}
		}
	}

	if count != len(out) {
		out = append(out, "[<flags>]")
	}

	return strings.Join(out, " ")
}

type FlagModel struct {
	Name        string   `json:"name"`
	Help        string   `json:"help"`
	Short       rune     `json:"short,omitempty"`
	Default     []string `json:"default,omitempty"`
	Envar       string   `json:"envar,omitempty"`
	PlaceHolder string   `json:"place_holder,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Hidden      bool     `json:"hidden,omitempty"`

	// used by plugin model
	Boolean     bool     `json:"boolean"`
	Negatable   bool     `json:"negatable,omitempty"`
	Cumulative  bool     `json:"cumulative"`
	Completions []string `json:"completions,omitempty"`

	Value Value `json:"-"`
}

// valueTypeHint returns a short human-readable name for a Value (e.g. "int", "duration", "enum(a|b)").
// Returns "string" when v is nil.
func valueTypeHint(v Value) string {
	if v == nil {
		return "string"
	}

	switch x := v.(type) {
	case *boolValue, *unNegatableBoolValue:
		return "bool"
	case *counterValue:
		return "counter"
	case *stringValue:
		return "string"
	case *intValue:
		return "int"
	case *int8Value:
		return "int8"
	case *int16Value:
		return "int16"
	case *int32Value:
		return "int32"
	case *int64Value:
		return "int64"
	case *uintValue:
		return "uint"
	case *uint8Value:
		return "uint8"
	case *uint16Value:
		return "uint16"
	case *uint32Value:
		return "uint32"
	case *uint64Value:
		return "uint64"
	case *float32Value:
		return "float32"
	case *float64Value:
		return "float64"
	case *durationValue:
		return "duration"
	case *bytesValue:
		return "bytes"
	case *ipValue, *resolvedIPValue:
		return "ip"
	case *tcpAddrValue:
		return "tcp_address"
	case *urlValue, *urlListValue:
		return "url"
	case *fileStatValue:
		return "path"
	case *fileValue:
		return "file"
	case *regexpValue:
		return "regexp"
	case *stringMapValue:
		return "key=value"
	case *enumValue:
		return "enum(" + strings.Join(x.options, "|") + ")"
	case *enumsValue:
		return "enum(" + strings.Join(x.options, "|") + ")"
	case *hexBytesValue:
		return "hex"
	default:
		if isBoolFlag(v) {
			return "bool"
		}
		return "string"
	}
}

// valueSchema returns a JSON schema fragment describing a Value. The description is "<help> (type-hint)".
// plainBool only matters when v is nil (plugin-model path) and forces a boolean type. cumulative wraps
// scalar types in an array.
func valueSchema(help string, v Value, cumulative bool, plainBool bool, helpHint bool) map[string]any {
	hint := valueTypeHint(v)
	if v == nil && plainBool {
		hint = "bool"
	}
	desc := help
	if helpHint {
		if desc != "" {
			desc = desc + " (" + hint + ")"
		} else {
			desc = "(" + hint + ")"
		}
	}

	schema := map[string]any{
		"description": desc,
	}

	if v == nil {
		if plainBool {
			schema["type"] = "boolean"
		} else {
			schema["type"] = "string"
		}
	} else {
		switch x := v.(type) {
		case *boolValue, *unNegatableBoolValue:
			schema["type"] = "boolean"
		case *counterValue, *intValue:
			schema["type"] = "integer"
		case *stringValue:
			schema["type"] = "string"
		case *int8Value:
			schema["type"] = "integer"
			schema["minimum"] = math.MinInt8
			schema["maximum"] = math.MaxInt8
		case *int16Value:
			schema["type"] = "integer"
			schema["minimum"] = math.MinInt16
			schema["maximum"] = math.MaxInt16
		case *int32Value:
			schema["type"] = "integer"
			schema["minimum"] = math.MinInt32
			schema["maximum"] = math.MaxInt32
		case *int64Value:
			schema["type"] = "integer"
		case *uintValue:
			schema["type"] = "integer"
			schema["minimum"] = 0
		case *uint8Value:
			schema["type"] = "integer"
			schema["minimum"] = 0
			schema["maximum"] = math.MaxUint8
		case *uint16Value:
			schema["type"] = "integer"
			schema["minimum"] = 0
			schema["maximum"] = math.MaxUint16
		case *uint32Value:
			schema["type"] = "integer"
			schema["minimum"] = 0
			schema["maximum"] = math.MaxUint32
		case *uint64Value:
			schema["type"] = "integer"
			schema["minimum"] = 0
		case *float32Value:
			schema["type"] = "number"
			schema["maximum"] = math.MaxFloat32
		case *float64Value:
			schema["type"] = "number"
		case *durationValue:
			schema["type"] = "string"
			schema["pattern"] = `^(0|[-+]?(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+)$`
		case *bytesValue:
			schema["type"] = "string"
			schema["pattern"] = `^(0|[-+]?(\d+(\.\d+)?[a-zA-Z]+)+)$`
		case *ipValue:
			schema["type"] = "string"
			schema["oneOf"] = []any{
				map[string]any{"format": "ipv4"},
				map[string]any{"format": "ipv6"},
			}
		case *resolvedIPValue:
			// hostname or IP, no usable pattern
			schema["type"] = "string"
		case *tcpAddrValue:
			// host:port where port may be a service name, no usable pattern
			schema["type"] = "string"
		case *urlValue:
			schema["type"] = "string"
			schema["format"] = "uri-reference"
		case *urlListValue:
			schema["type"] = "array"
			schema["items"] = map[string]any{
				"type":   "string",
				"format": "uri-reference",
			}
		case *fileStatValue:
			schema["type"] = "string"
		case *fileValue:
			schema["type"] = "string"
		case *regexpValue:
			schema["type"] = "string"
		case *stringMapValue:
			schema["type"] = "object"
			schema["additionalProperties"] = map[string]any{
				"type": "string",
			}
		case *enumValue:
			schema["type"] = "string"
			schema["enum"] = x.options
		case *enumsValue:
			schema["type"] = "array"
			schema["items"] = map[string]any{
				"type": "string",
				"enum": x.options,
			}
		case *hexBytesValue:
			schema["type"] = "string"
			schema["pattern"] = `^([0-9a-fA-F]{2})*$`
		default:
			if isBoolFlag(v) {
				schema["type"] = "boolean"
			} else {
				schema["type"] = "string"
			}
		}
	}

	if cumulative {
		if t, ok := schema["type"].(string); ok && t != "array" && t != "object" {
			schema["type"] = "array"
			schema["items"] = map[string]any{"type": t}
		}
	}

	return schema
}

// Schema returns a JSON schema fragment for the flag, the schema does not reflect flag name nor if it's required as that is for the parent property, callers should set those if needed.//
//
// when typeHint is true the description will have a type hint added to it
func (f *FlagModel) Schema(typeHint bool) map[string]any {
	return valueSchema(f.Help, f.Value, f.Cumulative, f.Boolean, typeHint)
}

func (f *FlagModel) String() string {
	if f.Value == nil {
		return ""
	}
	return f.Value.String()
}

func (f *FlagModel) IsCumulative() bool {
	if f.Value == nil {
		return false
	}

	v, ok := f.Value.(repeatableFlag)
	if !ok {
		return false
	}

	return v.IsCumulative()
}

func (f *FlagModel) IsBoolFlag() bool {
	return isBoolFlag(f.Value)
}

func (f *FlagModel) IsNegatable() bool {
	bf, ok := f.Value.(BoolFlag)
	return ok && bf.BoolFlagIsNegatable()
}

func (f *FlagModel) FormatPlaceHolder() string {
	if f.PlaceHolder != "" {
		return f.PlaceHolder
	}
	if len(f.Default) > 0 {
		ellipsis := ""
		if len(f.Default) > 1 {
			ellipsis = "..."
		}
		if _, ok := f.Value.(*stringValue); ok {
			return strconv.Quote(f.Default[0]) + ellipsis
		}
		return f.Default[0] + ellipsis
	}
	return strings.ToUpper(f.Name)
}

func (f *FlagModel) HelpWithEnvar() string {
	if f.Envar == "" {
		return f.Help
	}
	return fmt.Sprintf("%s ($%s)", f.Help, f.Envar)
}

type ArgGroupModel struct {
	Args []*ArgModel `json:"args,omitempty"`
}

func (a *ArgGroupModel) ArgSummary() string {
	depth := 0
	out := []string{}
	for _, arg := range a.Args {
		var h string
		if arg.PlaceHolder != "" {
			h = arg.PlaceHolder
		} else {
			h = "<" + arg.Name + ">"
		}
		if !arg.Required {
			h = "[" + h
			depth++
		}
		out = append(out, h)
	}
	out[len(out)-1] = out[len(out)-1] + strings.Repeat("]", depth)
	return strings.Join(out, " ")
}

func (a *ArgModel) HelpWithEnvar() string {
	if a.Envar == "" {
		return a.Help
	}
	return fmt.Sprintf("%s ($%s)", a.Help, a.Envar)
}

type ArgModel struct {
	Name        string   `json:"name"`
	Help        string   `json:"help"`
	Default     []string `json:"default,omitempty"`
	Envar       string   `json:"envar,omitempty"`
	PlaceHolder string   `json:"place_holder,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Hidden      bool     `json:"hidden,omitempty"`
	Value       Value    `json:"-"`

	// used by plugin model
	Cumulative bool `json:"cumulative"`
}

func (a *ArgModel) IsCumulative() bool {
	if a.Value == nil {
		return false
	}

	v, ok := a.Value.(remainderArg)
	if !ok {
		return false
	}

	return v.IsCumulative()
}

func (a *ArgModel) String() string {
	if a.Value == nil {
		return ""
	}

	return a.Value.String()
}

// Schema returns a JSON schema fragment for the argument, the schema does not reflect argument name nor if its required as that is for the parent property, callers should set those if needed.
//
// when typeHint is true the description will have a type hint added to it
func (a *ArgModel) Schema(typeHint bool) map[string]any {
	return valueSchema(a.Help, a.Value, a.Cumulative, false, typeHint)
}

type CmdGroupModel struct {
	Commands []*CmdModel `json:"commands,omitempty"`
}

// HasTags returns true if any visible command in the group has tags
func (c *CmdGroupModel) HasTags() bool {
	for _, cmd := range c.Commands {
		if !cmd.Hidden && len(cmd.Tags) > 0 {
			return true
		}
	}
	return false
}

func (c *CmdGroupModel) FlattenedCommands() (out []*CmdModel) {
	for _, cmd := range c.Commands {
		if len(cmd.Commands) == 0 {
			out = append(out, cmd)
		}
		out = append(out, cmd.FlattenedCommands()...)
	}
	return
}

type CmdModel struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases,omitempty"`
	Help        string   `json:"help"`
	HelpLong    string   `json:"help_long,omitempty"`
	FullCommand string   `json:"-"`
	Depth       int      `json:"-"`
	Hidden      bool     `json:"hidden,omitempty"`
	Default     bool     `json:"default,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	*FlagGroupModel
	*ArgGroupModel
	*CmdGroupModel
}

func (c *CmdModel) String() string {
	return c.FullCommand
}

type ApplicationModel struct {
	Name         string            `json:"name"`
	Help         string            `json:"help"`
	Cheat        string            `json:"cheat,omitempty"`
	Version      string            `json:"version,omitempty"`
	Author       string            `json:"author,omitempty"`
	Cheats       map[string]string `json:"cheats,omitempty"`
	CheatTags    []string          `json:"cheat_tags,omitempty"`
	LLMExtraInfo string            `json:"llm_extra_info,omitempty"`

	*ArgGroupModel
	*CmdGroupModel
	*FlagGroupModel
}

func (a *Application) Model() *ApplicationModel {
	return &ApplicationModel{
		Name:           a.Name,
		Help:           a.Help,
		Version:        a.version,
		Author:         a.author,
		Cheats:         a.cheats,
		CheatTags:      a.cheatTags,
		LLMExtraInfo:   a.llmExtraInfo,
		FlagGroupModel: a.flagGroup.Model(),
		ArgGroupModel:  a.argGroup.Model(),
		CmdGroupModel:  a.cmdGroup.Model(),
	}
}

func (a *argGroup) Model() *ArgGroupModel {
	m := &ArgGroupModel{}
	for _, arg := range a.args {
		m.Args = append(m.Args, arg.Model())
	}
	return m
}

func (a *ArgClause) Model() *ArgModel {
	m := &ArgModel{
		Name:        a.name,
		Help:        a.help,
		Default:     a.defaultValues,
		Envar:       a.envar,
		PlaceHolder: a.placeholder,
		Required:    a.required,
		Hidden:      a.hidden,
		Value:       a.value,
	}

	m.Cumulative = m.IsCumulative()

	return m
}

func (f *flagGroup) Model() *FlagGroupModel {
	m := &FlagGroupModel{}
	for _, fl := range f.flagOrder {
		m.Flags = append(m.Flags, fl.Model())
	}
	return m
}

func (f *FlagClause) Model() *FlagModel {
	m := &FlagModel{
		Name:        f.name,
		Help:        f.help,
		Short:       f.shorthand,
		Default:     f.defaultValues,
		Envar:       f.envar,
		PlaceHolder: f.placeholder,
		Required:    f.required,
		Hidden:      f.hidden,
		Value:       f.value,
	}

	m.Boolean = m.IsBoolFlag()
	m.Negatable = m.IsNegatable()
	m.Cumulative = m.IsCumulative()
	m.Completions = f.resolveCompletions()

	return m
}

func (c *cmdGroup) Model() *CmdGroupModel {
	m := &CmdGroupModel{}
	for _, cm := range c.commandOrder {
		m.Commands = append(m.Commands, cm.Model())
	}
	return m
}

func (c *CmdClause) Model() *CmdModel {
	depth := 0
	for i := c; i != nil; i = i.parent {
		depth++
	}
	return &CmdModel{
		Name:           c.name,
		Aliases:        c.aliases,
		Help:           c.help,
		HelpLong:       c.helpLong,
		Depth:          depth,
		Hidden:         c.hidden,
		Default:        c.isDefault,
		Tags:           c.tags,
		FullCommand:    c.FullCommand(),
		FlagGroupModel: c.flagGroup.Model(),
		ArgGroupModel:  c.argGroup.Model(),
		CmdGroupModel:  c.cmdGroup.Model(),
	}
}
