package fisk

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type pluginDelegator struct {
	command        string
	flags          map[string]*string
	cumuFlags      map[string]*[]string
	boolFlags      map[string]*bool
	unNegBoolFlags map[string]*bool
	args           map[string]*string
	cumuArgs       map[string]*[]string
	proxyGlobals   []string
	globalFlags    *flagGroup
	flagsIsSet     map[string]*bool
	parent         string
	name           string
}

func (a *Application) introspectModel() *ApplicationModel {
	model := a.Model()
	var nf []*FlagModel
	for _, flag := range model.Flags {
		if flag.Name == "help" || strings.HasPrefix(flag.Name, "help-") || strings.HasPrefix(flag.Name, "completion-") || strings.HasPrefix(flag.Name, "fisk-") || flag.Name == "version" {
			continue
		}

		nf = append(nf, flag)
	}
	model.Flags = nf

	var nc []*CmdModel
	for _, cmd := range model.Commands {
		if cmd.Name == "help" || cmd.Name == "cheat" || cmd.Name == "help_long" {
			continue
		}
		nc = append(nc, cmd)
	}
	model.Commands = nc

	return model
}

func (a *Application) introspectAction(_ *ParseContext) error {
	a.Writer(os.Stdout)

	j, err := json.Marshal(a.introspectModel())
	if err != nil {
		return err
	}

	fmt.Println(string(j))

	a.terminate(0)

	return nil
}

func (c *CmdClause) addArgsFromModel(model *ArgGroupModel) {
	if model == nil {
		return
	}

	for _, arg := range model.Args {
		a := c.Arg(arg.Name, arg.Help)
		a.placeholder = arg.PlaceHolder
		a.required = arg.Required
		a.hidden = arg.Hidden
		a.defaultValues = arg.Default
		a.envar = arg.Envar

		switch {
		case arg.Cumulative:
			c.pluginDelegator.cumuArgs[arg.Name] = a.Strings()

		default:
			c.pluginDelegator.args[arg.Name] = a.String()
		}
	}
}

func (c *CmdClause) addFlagsFromModel(model *FlagGroupModel, appFlags *FlagGroupModel) {
	if model == nil {
		return
	}

	for _, flag := range model.Flags {
		_, ok := c.pluginDelegator.flagsIsSet[flag.Name]
		if !ok {
			c.pluginDelegator.flagsIsSet[flag.Name] = func() *bool { v := false; return &v }()
		}

		if f, ok := c.pluginDelegator.globalFlags.long[flag.Name]; ok {
			f.setByUser = c.pluginDelegator.flagsIsSet[flag.Name]
			c.pluginDelegator.proxyGlobals = append(c.pluginDelegator.proxyGlobals, flag.Name)
			continue
		}

		f := c.Flag(flag.Name, flag.Help)
		f.shorthand = flag.Short
		f.defaultValues = flag.Default
		f.envar = flag.Envar
		f.placeholder = flag.PlaceHolder
		f.required = flag.Required
		f.hidden = flag.Hidden

		f.setByUser = c.pluginDelegator.flagsIsSet[flag.Name]

		switch {
		case flag.Boolean && flag.Negatable:
			c.pluginDelegator.boolFlags[flag.Name] = f.Bool()

		case flag.Boolean:
			c.pluginDelegator.unNegBoolFlags[flag.Name] = f.UnNegatableBool()

		case flag.Cumulative:
			c.pluginDelegator.cumuFlags[flag.Name] = f.Strings()

		default:
			c.pluginDelegator.flags[flag.Name] = f.String()
		}
	}
}

func (c *CmdClause) pluginAction(pd *pluginDelegator) Action {
	return func(pc *ParseContext) error {
		parts := strings.Split(pc.SelectedCommand.FullCommand(), " ")
		args := parts[1:]

		for _, v := range pd.args {
			if v != nil && *v != "" {
				args = append(args, *v)
			}
		}

		for k, v := range pd.flags {
			if v != nil && *pd.flagsIsSet[k] {
				args = append(args, fmt.Sprintf("--%s=%s", k, *v))
			}
		}

		for k, v := range pd.cumuFlags {
			if v == nil {
				continue
			}

			for _, cv := range *v {
				if v != nil && *pd.flagsIsSet[k] {
					args = append(args, fmt.Sprintf("--%s=%s", k, cv))
				}
			}
		}

		for k, v := range pd.boolFlags {
			if !*pd.flagsIsSet[k] {
				continue
			}

			if *v {
				args = append(args, fmt.Sprintf("--%s", k))
			} else {
				args = append(args, fmt.Sprintf("--no-%s", k))
			}
		}

		for k, v := range pd.unNegBoolFlags {
			if !*pd.flagsIsSet[k] {
				continue
			}

			if *v {
				args = append(args, fmt.Sprintf("--%s", k))
			}
		}

		for _, f := range pd.proxyGlobals {
			if !*pd.flagsIsSet[f] {
				continue
			}

			args = append(args, fmt.Sprintf("--%s=%s", f, pd.globalFlags.long[f].value.String()))
		}

		// must be last
		for _, v := range pd.cumuArgs {
			if v != nil {
				for _, i := range *v {
					if i != "" {
						args = append(args, i)
					}
				}
			}
		}

		if os.Getenv("FISK_DEBUG") != "" {
			fmt.Printf("Fisk Plugin Running: %s %s\n", pd.command, strings.Join(args, " "))
			fmt.Printf("PD: %#v\n", pd)
		}

		cmd := exec.Command(pd.command, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = os.Environ()
		return cmd.Run()
	}
}

func (c *CmdClause) addCommandsFromModel(model *CmdGroupModel) {
	if model == nil {
		return
	}

	for _, cmd := range model.Commands {
		pd := pluginDelegator{
			parent:         c.name,
			name:           cmd.Name,
			flags:          map[string]*string{},
			cumuFlags:      map[string]*[]string{},
			args:           map[string]*string{},
			cumuArgs:       map[string]*[]string{},
			boolFlags:      map[string]*bool{},
			unNegBoolFlags: map[string]*bool{},
			flagsIsSet:     c.pluginDelegator.flagsIsSet,   // shared with global so global flags isSet is also handled
			command:        c.pluginDelegator.command,      // the command to run is always the same
			globalFlags:    c.pluginDelegator.globalFlags,  // global flags are global
			proxyGlobals:   c.pluginDelegator.proxyGlobals, // global flags are global
		}

		cm := c.Command(cmd.Name, cmd.Help)
		cm.pluginDelegator = &pd

		cm.aliases = cmd.Aliases
		cm.helpLong = cmd.HelpLong
		cm.hidden = cmd.Hidden
		cm.isDefault = cmd.Default

		if cmd.CmdGroupModel == nil || len(cmd.CmdGroupModel.Commands) == 0 {
			cm.Action(cm.pluginAction(&pd))
		}

		cm.addArgsFromModel(cmd.ArgGroupModel)
		cm.addFlagsFromModel(cmd.FlagGroupModel, nil)
		cm.addCommandsFromModel(cmd.CmdGroupModel)
	}
}

func (a *Application) registerPluginModel(command string, model *ApplicationModel) (*CmdClause, error) {
	cmd := a.Command(model.Name, model.Help)
	cmd.pluginDelegator = &pluginDelegator{
		parent:         a.Name,
		command:        command,
		flags:          map[string]*string{},
		flagsIsSet:     map[string]*bool{},
		cumuFlags:      map[string]*[]string{},
		args:           map[string]*string{},
		cumuArgs:       map[string]*[]string{},
		boolFlags:      map[string]*bool{},
		unNegBoolFlags: map[string]*bool{},
		globalFlags:    a.flagGroup,
	}

	for k, v := range model.Cheats {
		_, ok := a.cheats[k]
		if ok {
			continue
		}

		a.cheats[k] = v
		a.cheatTags = append(a.cheatTags, k)
	}

	cmd.addArgsFromModel(model.ArgGroupModel)
	cmd.addFlagsFromModel(model.FlagGroupModel, a.Model().FlagGroupModel)
	cmd.addCommandsFromModel(model.CmdGroupModel)

	return cmd, nil
}

// ExternalPluginCommand extends the application using a plugin and a model describing the application, when name or help is not an empty string it will override that from the plugin
func (a *Application) ExternalPluginCommand(command string, model json.RawMessage, name string, help string) (*CmdClause, error) {
	var m ApplicationModel
	err := json.Unmarshal(model, &m)
	if err != nil {
		return nil, err
	}

	if name != "" {
		m.Name = name
	}
	if help != "" {
		m.Help = help
	}

	if m.Name == "" {
		return nil, fmt.Errorf("plugin declared no name")
	}
	if m.Help == "" {
		return nil, fmt.Errorf("plugin declared no help")
	}

	return a.registerPluginModel(command, &m)
}
