package fisk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// formatFlagArg renders a flag and a single value as one command-line token. It
// is shared with the plugin executor so both render flags identically.
func formatFlagArg(name, value string) string {
	return fmt.Sprintf("--%s=%s", name, value)
}

// formatBoolFlagArg renders a negatable boolean flag as either --name or
// --no-name. It is shared with the plugin executor.
func formatBoolFlagArg(name string, value bool) string {
	if value {
		return "--" + name
	}
	return "--no-" + name
}

// formatUnNegatableBoolFlagArg renders an un-negatable boolean flag. ok is false
// when nothing should be emitted, which is the only way to express a false value
// for a flag that has no negation. It is shared with the plugin executor.
func formatUnNegatableBoolFlagArg(name string, value bool) (arg string, ok bool) {
	if value {
		return "--" + name, true
	}
	return "", false
}

// ArgsFromJSON generates the command-line arguments for this command from a JSON
// object whose keys and value types match the command's Schema(). Each property
// of that schema is one of the command's flags or positional arguments, and this
// converts the supplied value for each into the equivalent command-line tokens.
//
// It returns the argument tail only: the flags first, then "--", then the
// positional arguments in their declared order. The program name and the command
// path are not included. Numbers are decoded with UseNumber so integer values are
// not distorted by floating-point conversion, and properties that are not part of
// the schema are rejected.
func (c *CmdModel) ArgsFromJSON(arguments json.RawMessage) ([]string, error) {
	input := map[string]any{}
	if len(arguments) > 0 {
		dec := json.NewDecoder(bytes.NewReader(arguments))
		dec.UseNumber()
		err := dec.Decode(&input)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON arguments: %w", err)
		}
	}

	flags := map[string]*FlagModel{}
	if c.FlagGroupModel != nil {
		for _, f := range c.Flags {
			flags[f.Name] = f
		}
	}

	args := map[string]*ArgModel{}
	if c.ArgGroupModel != nil {
		for _, a := range c.Args {
			args[a.Name] = a
		}
	}

	for key := range input {
		_, isFlag := flags[key]
		_, isArg := args[key]
		if !isFlag && !isArg {
			return nil, fmt.Errorf("unknown property %q", key)
		}
	}

	var out []string

	// Flags are emitted in declared order so the result is deterministic.
	if c.FlagGroupModel != nil {
		for _, f := range c.Flags {
			val, ok := input[f.Name]
			if !ok {
				continue
			}

			rendered, err := flagArgsFromValue(f, val)
			if err != nil {
				return nil, fmt.Errorf("flag %q: %w", f.Name, err)
			}

			out = append(out, rendered...)
		}
	}

	positionals, err := c.positionalArgsFromInput(input)
	if err != nil {
		return nil, err
	}
	if len(positionals) > 0 {
		out = append(out, "--")
		out = append(out, positionals...)
	}

	return out, nil
}

// flagArgsFromValue renders a single flag from its JSON value. Booleans use the
// negatable or un-negatable form depending on the flag; arrays repeat the flag
// once per element; objects (key=value maps) repeat the flag once per entry,
// sorted by key for determinism; everything else is a single --name=value token.
func flagArgsFromValue(f *FlagModel, val any) ([]string, error) {
	if f.Boolean {
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("expected boolean, got %s", jsonKindOf(val))
		}

		if f.Negatable {
			return []string{formatBoolFlagArg(f.Name, b)}, nil
		}

		arg, ok := formatUnNegatableBoolFlagArg(f.Name, b)
		if ok {
			return []string{arg}, nil
		}

		// A false value for an un-negatable flag cannot be expressed. If the flag
		// defaults to true, omitting it would leave it true, the opposite of what
		// was asked, so this is an error rather than a silent no-op.
		if defaultIsTrue(f.Default) {
			return nil, fmt.Errorf("cannot be set to false: the flag has no negation and defaults to true")
		}

		return nil, nil
	}

	switch v := val.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, err := scalarString(item)
			if err != nil {
				return nil, err
			}
			out = append(out, formatFlagArg(f.Name, s))
		}
		return out, nil

	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		out := make([]string, 0, len(v))
		for _, k := range keys {
			s, err := scalarString(v[k])
			if err != nil {
				return nil, err
			}
			out = append(out, formatFlagArg(f.Name, fmt.Sprintf("%s=%s", k, s)))
		}
		return out, nil

	default:
		s, err := scalarString(val)
		if err != nil {
			return nil, err
		}
		return []string{formatFlagArg(f.Name, s)}, nil
	}
}

// positionalArgsFromInput renders the positional arguments in declared order.
// Command lines cannot skip a positional, so when an argument is omitted but a
// later one is supplied the gap is filled from the omitted argument's default;
// an argument with no default in that position is an error. A cumulative
// argument must be the last one declared.
func (c *CmdModel) positionalArgsFromInput(input map[string]any) ([]string, error) {
	if c.ArgGroupModel == nil || len(c.Args) == 0 {
		return nil, nil
	}

	for i, a := range c.Args {
		if a.Cumulative && i != len(c.Args)-1 {
			return nil, fmt.Errorf("argument %q is cumulative but is not the last argument", a.Name)
		}
	}

	last := -1
	for i, a := range c.Args {
		_, ok := input[a.Name]
		if ok {
			last = i
		}
	}
	if last < 0 {
		return nil, nil
	}

	var out []string
	for i := 0; i <= last; i++ {
		a := c.Args[i]

		val, ok := input[a.Name]
		if !ok {
			if len(a.Default) == 0 {
				return nil, fmt.Errorf("argument %q must be set because a later argument is set", a.Name)
			}
			out = append(out, a.Default...)
			continue
		}

		rendered, err := argValuesFromValue(a, val)
		if err != nil {
			return nil, fmt.Errorf("argument %q: %w", a.Name, err)
		}
		out = append(out, rendered...)
	}

	return out, nil
}

// argValuesFromValue renders a single positional argument from its JSON value.
// Arrays expand to one token per element; everything else is a single token.
// Unlike the plugin executor, an explicit empty string is preserved here because
// the presence of the key distinguishes it from an unset argument.
func argValuesFromValue(a *ArgModel, val any) ([]string, error) {
	arr, ok := val.([]any)
	if !ok {
		s, err := scalarString(val)
		if err != nil {
			return nil, err
		}
		return []string{s}, nil
	}

	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, err := scalarString(item)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// scalarString renders a scalar JSON value as the string fisk would parse it
// from. json.Number is emitted verbatim so integers keep their exact form.
func scalarString(v any) (string, error) {
	switch x := v.(type) {
	case string:
		return x, nil
	case json.Number:
		return x.String(), nil
	case bool:
		return strconv.FormatBool(x), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("unsupported value type %s", jsonKindOf(v))
	}
}

// jsonKindOf names the JSON kind of a decoded value for error messages.
func jsonKindOf(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case json.Number:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// defaultIsTrue reports whether any of a flag's default values parses as true.
func defaultIsTrue(def []string) bool {
	for _, d := range def {
		b, err := strconv.ParseBool(d)
		if err == nil && b {
			return true
		}
	}
	return false
}
