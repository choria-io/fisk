package fisk

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var (
	envarTransformRegexp = regexp.MustCompile(`[^a-zA-Z0-9_]+`)
)

type ApplicationValidator func(*Application) error

// An Application contains the definitions of flags, arguments and commands
// for an application.
type Application struct {
	cmdMixin
	initialized bool

	Name string
	Help string

	author             string
	version            string
	errorWriter        io.Writer // Destination for errors.
	usageWriter        io.Writer // Destination for usage
	usageTemplate      string
	errorUsageTemplate string
	usageFuncs         template.FuncMap
	validator          ApplicationValidator
	terminate          func(status int) // See Terminate()
	noInterspersed     bool             // can flags be interspersed with args (or must they come first)
	defaultEnvars      bool
	completion         bool
	cheats             map[string]string
	cheatTags          []string

	// Help flag. Exposed for user customisation.
	HelpFlag *FlagClause
	// Help command. Exposed for user customisation. May be nil.
	HelpCommand *CmdClause
	// Version flag. Exposed for user customisation. May be nil.
	VersionFlag *FlagClause
	// Cheat command. Exposed for user customisation. May be nil.
	CheatCommand *CmdClause
}

// Newf creates a new application with printf parsing of the help
func Newf(name string, format string, a ...interface{}) *Application {
	return New(name, fmt.Sprintf(format, a...))
}

// New creates a new Fisk application instance.
func New(name, help string) *Application {
	a := &Application{
		Name:               name,
		Help:               help,
		errorWriter:        os.Stderr, // Left for backwards compatibility purposes.
		usageWriter:        os.Stderr,
		usageTemplate:      ShorterMainUsageTemplate,
		errorUsageTemplate: compactWithoutFlagsOrArgs,
		terminate:          os.Exit,
		cheats:             map[string]string{},
		cheatTags:          []string{name},
	}

	a.flagGroup = newFlagGroup()
	a.argGroup = newArgGroup()
	a.cmdGroup = newCmdGroup(a)
	a.HelpFlag = a.Flag("help", "Show context-sensitive help")
	a.HelpFlag.UnNegatableBool()

	a.Flag("help-long", "Generate long help.").Hidden().PreAction(a.generateLongHelp).UnNegatableBool()
	a.Flag("help-compact", "Generate compact help.").Hidden().PreAction(a.generateCompactHelp).UnNegatableBool()
	a.Flag("help-man", "Generate a man page.").Hidden().PreAction(a.generateManPage).UnNegatableBool()
	a.Flag("completion-bash", "Output possible completions for the given args.").Hidden().UnNegatableBoolVar(&a.completion)
	a.Flag("completion-script-bash", "Generate completion script for bash.").Hidden().PreAction(a.generateBashCompletionScript).UnNegatableBool()
	a.Flag("completion-script-zsh", "Generate completion script for ZSH.").Hidden().PreAction(a.generateZSHCompletionScript).UnNegatableBool()

	return a
}

func (a *Application) generateCompactHelp(c *ParseContext) error {
	a.Writer(os.Stdout)
	if err := a.UsageForContextWithTemplate(c, 2, CompactUsageTemplate); err != nil {
		return err
	}
	a.terminate(0)
	return nil
}

func (a *Application) generateLongHelp(c *ParseContext) error {
	a.Writer(os.Stdout)
	if err := a.UsageForContextWithTemplate(c, 2, LongHelpTemplate); err != nil {
		return err
	}
	a.terminate(0)
	return nil
}

func (a *Application) generateManPage(c *ParseContext) error {
	a.Writer(os.Stdout)
	if err := a.UsageForContextWithTemplate(c, 2, ManPageTemplate); err != nil {
		return err
	}
	a.terminate(0)
	return nil
}

func (a *Application) generateBashCompletionScript(c *ParseContext) error {
	a.Writer(os.Stdout)
	if err := a.UsageForContextWithTemplate(c, 2, BashCompletionTemplate); err != nil {
		return err
	}
	a.terminate(0)
	return nil
}

func (a *Application) generateZSHCompletionScript(c *ParseContext) error {
	a.Writer(os.Stdout)
	if err := a.UsageForContextWithTemplate(c, 2, ZshCompletionTemplate); err != nil {
		return err
	}
	a.terminate(0)
	return nil
}

// DefaultEnvars configures all flags (that do not already have an associated
// envar) to use a default environment variable in the form "<app>_<flag>".
//
// For example, if the application is named "foo" and a flag is named "bar-
// waz" the environment variable: "FOO_BAR_WAZ".
func (a *Application) DefaultEnvars() *Application {
	a.defaultEnvars = true
	return a
}

// Terminate specifies the termination handler. Defaults to os.Exit(status).
// If nil is passed, a no-op function will be used.
func (a *Application) Terminate(terminate func(int)) *Application {
	if terminate == nil {
		terminate = func(int) {}
	}
	a.terminate = terminate
	return a
}

// Writer specifies the writer to use for usage and errors. Defaults to os.Stderr.
// DEPRECATED: See ErrorWriter and UsageWriter.
func (a *Application) Writer(w io.Writer) *Application {
	a.errorWriter = w
	a.usageWriter = w
	return a
}

// ErrorWriter sets the io.Writer to use for errors.
func (a *Application) ErrorWriter(w io.Writer) *Application {
	a.errorWriter = w
	return a
}

// UsageWriter sets the io.Writer to use for errors.
func (a *Application) UsageWriter(w io.Writer) *Application {
	a.usageWriter = w
	return a
}

// UsageTemplate specifies the text template to use when displaying usage
// information. The default is UsageTemplate.
func (a *Application) UsageTemplate(template string) *Application {
	a.usageTemplate = template
	return a
}

// ErrorUsageTemplate specifies the text template to use when displaying usage
// information after an ErrSubCommandRequired ErrExpectedKnownCommand. The
// default is compactWithoutFlagsOrArgs.
func (a *Application) ErrorUsageTemplate(template string) *Application {
	a.errorUsageTemplate = template
	return a
}

// UsageFuncs adds extra functions that can be used in the usage template.
func (a *Application) UsageFuncs(funcs template.FuncMap) *Application {
	a.usageFuncs = funcs
	return a
}

// Validate sets a validation function to run when parsing.
func (a *Application) Validate(validator ApplicationValidator) *Application {
	a.validator = validator
	return a
}

// ParseContext parses the given command line and returns the fully populated
// ParseContext.
func (a *Application) ParseContext(args []string) (*ParseContext, error) {
	return a.parseContext(false, args)
}

func (a *Application) parseContext(ignoreDefault bool, args []string) (*ParseContext, error) {
	if err := a.init(); err != nil {
		return nil, err
	}
	context := tokenize(args, ignoreDefault)
	err := parse(context, a)
	return context, err
}

// Parse parses command-line arguments. It returns the selected command and an
// error. The selected command will be a space separated subcommand, if
// subcommands have been configured.
//
// This will populate all flag and argument values, call all callbacks, and so
// on.
func (a *Application) Parse(args []string) (command string, err error) {
	context, parseErr := a.ParseContext(args)
	var selected []string
	var setValuesErr error

	if context == nil {
		// Since we do not throw error immediately, there could be a case
		// where a context returns nil. Protect against that.
		return "", parseErr
	}

	if err = a.setDefaults(context); err != nil {
		return "", err
	}

	selected, setValuesErr = a.setValues(context)

	if err = a.applyPreActions(context, !a.completion); err != nil {
		return "", err
	}

	if a.completion {
		a.generateBashCompletion(context)
		a.terminate(0)
	} else {
		if parseErr != nil {
			return "", parseErr
		}

		a.maybeHelp(context)
		if !context.EOL() {
			return "", fmt.Errorf("%w '%s'", ErrUnexpectedArgument, context.Peek())
		}

		if setValuesErr != nil {
			return "", setValuesErr
		}

		command, err = a.execute(context, selected)
		if err == ErrCommandNotSpecified {
			a.writeUsage(context, nil)
		}
	}

	return command, err
}

func (a *Application) writeUsage(context *ParseContext, err error) {
	if err != nil {
		a.Errorf("%s", err)
	}
	if err := a.UsageForContext(context); err != nil {
		panic(err)
	}
	if err != nil {
		a.terminate(1)
	} else {
		a.terminate(0)
	}
}

func (a *Application) maybeHelp(context *ParseContext) {
	for _, element := range context.Elements {
		if flag, ok := element.Clause.(*FlagClause); ok && flag == a.HelpFlag {
			// Re-parse the command-line ignoring defaults, so that help works correctly.
			context, _ = a.parseContext(true, context.rawArgs)
			a.writeUsage(context, nil)
		}
	}
}

func (a *Application) listCheats() {
	if len(a.cheats) == 0 {
		fmt.Fprintln(a.usageWriter, "No cheats defined")
		return
	}

	var list []string
	top := ""
	for k := range a.cheats {
		if k == a.Name {
			top = a.Name
			continue
		}
		list = append(list, k)
	}
	sort.Strings(list)
	if top != "" {
		list = append([]string{top}, list...)
	}

	fmt.Fprintln(a.usageWriter, "Available Cheats:")
	fmt.Fprintln(a.usageWriter)
	for _, k := range list {
		fmt.Fprintf(a.usageWriter, "    %s\n", k)
	}
}

func (a *Application) saveCheats(dir string) error {
	if len(a.cheats) == 0 {
		return fmt.Errorf("no cheats defined")
	}

	err := os.MkdirAll(dir, 0744)
	if err != nil {
		return err
	}

	tags := a.cheatTags
	if len(tags) == 0 {
		tags = []string{a.Name}
	}

	var list []string
	for k := range a.cheats {
		list = append(list, k)
	}
	sort.Strings(list)

	for _, k := range list {
		if a.cheats[k] == "" {
			continue
		}

		dest := filepath.Join(dir, k)
		f, err := os.Create(dest)
		if err != nil {
			return err
		}

		fmt.Fprintf(f, "---\ntags: [%s]\n---\n\n", strings.Join(tags, ", "))
		fmt.Fprintln(f, a.cheats[k])
		f.Close()

		fmt.Fprintf(a.usageWriter, "Saved cheat to %s\n", dest)

	}

	return nil
}

// WithCheats enables support for rendering cheat compatible output,
// tags can be supplied which would be set when saving cheat files
//
// See https://github.com/cheat/cheat for information about this format
func (a *Application) WithCheats(tags ...string) *Application {
	if len(tags) > 0 {
		a.cheatTags = tags
	}

	var (
		cheat string
		list  bool
		dir   string
	)

	a.CheatCommand = a.Commandf("cheat", "Shows cheats for %s", a.Name).Action(func(pc *ParseContext) error {
		switch {
		case dir != "":
			return a.saveCheats(dir)

		case list:
			a.listCheats()

		default:
			if len(a.cheats) == 0 {
				a.listCheats()
				break
			}

			if cheat == "" {
				if len(a.cheats) > 1 {
					a.listCheats()
					break
				} else {
					for k := range a.cheats {
						cheat = k
					}
				}
			}

			cheat, ok := a.cheats[cheat]
			if !ok {
				a.listCheats()
				break
			}

			fmt.Fprintln(a.usageWriter, cheat)
		}

		a.terminate(0)

		return nil
	})
	a.CheatCommand.HelpLong(`These cheats are compatible with the 'cheat' CLI tool
and by saving the output using --save these cheats become accessible within that application.

See https://github.com/cheat/cheat for more details`)
	a.CheatCommand.Arg("label", "The cheat to show").StringVar(&cheat)
	a.CheatCommand.Flag("list", "List available cheats").UnNegatableBoolVar(&list)
	a.CheatCommand.Flag("save", "Saves the cheats to the given directory").PlaceHolder("DIRECTORY").StringVar(&dir)

	return a
}

// CheatFile reads a file from fs and use its contents to call Cheat(). Read errors are fatal.
func (a *Application) CheatFile(fs fs.ReadFileFS, cheat string, file string) *Application {
	body, err := fs.ReadFile(file)
	a.FatalIfError(err, "cannot load cheat: %v", err)

	return a.Cheat(cheat, string(body))
}

// Cheat sets the cheat help text to associate with this application,
// the cheat is the name it will be surfaced as in help, if empty its the
// name of the application.
func (a *Application) Cheat(cheat string, help string) *Application {
	if help == "" {
		return a
	}

	if cheat == "" {
		cheat = a.Name
	}

	a.cheats[cheat] = help

	if a.CheatCommand == nil {
		a.WithCheats()
	}

	return a
}

// Version adds a --version flag for displaying the application version.
func (a *Application) Version(version string) *Application {
	a.version = version
	a.VersionFlag = a.Flag("version", "Show application version.").PreAction(func(*ParseContext) error {
		fmt.Fprintln(a.usageWriter, version)
		a.terminate(0)
		return nil
	})
	a.VersionFlag.UnNegatableBool()
	return a
}

// Author sets the author output by some help templates.
func (a *Application) Author(author string) *Application {
	a.author = author
	return a
}

// Action callback to call when all values are populated and parsing is
// complete, but before any command, flag or argument actions.
//
// All Action() callbacks are called in the order they are encountered on the
// command line.
func (a *Application) Action(action Action) *Application {
	a.addAction(action)
	return a
}

// PreAction action called after parsing completes but before validation and execution.
func (a *Application) PreAction(action Action) *Application {
	a.addPreAction(action)
	return a
}

// Commandf adds a new top-level command with printf parsing of help
func (a *Application) Commandf(name string, format string, arg ...interface{}) *CmdClause {
	return a.Command(name, fmt.Sprintf(format, arg...))
}

// Command adds a new top-level command.
func (a *Application) Command(name, help string) *CmdClause {
	return a.addCommand(name, help)
}

// Interspersed control if flags can be interspersed with positional arguments
//
// true (the default) means that they can, false means that all the flags must appear before the first positional arguments.
func (a *Application) Interspersed(interspersed bool) *Application {
	a.noInterspersed = !interspersed
	return a
}

func (a *Application) defaultEnvarPrefix() string {
	if a.defaultEnvars {
		return a.Name
	}
	return ""
}

func (a *Application) init() error {
	if a.initialized {
		return nil
	}
	if a.cmdGroup.have() && a.argGroup.have() {
		return fmt.Errorf("can't mix top-level Arg()s with Command()s")
	}

	// If we have subcommands, add a help command at the top-level.
	if a.cmdGroup.have() {
		var command []string
		a.HelpCommand = a.Command("help", "Show help.").PreAction(func(context *ParseContext) error {
			a.Usage(command)
			a.terminate(0)
			return nil
		})
		a.HelpCommand.Arg("command", "Show help on command.").StringsVar(&command)
		// Make help first command.
		l := len(a.commandOrder)
		a.commandOrder = append(a.commandOrder[l-1:l], a.commandOrder[:l-1]...)
	}

	if err := a.flagGroup.init(a.defaultEnvarPrefix()); err != nil {
		return err
	}
	if err := a.cmdGroup.init(); err != nil {
		return err
	}
	if err := a.argGroup.init(); err != nil {
		return err
	}
	for _, cmd := range a.commands {
		if err := cmd.init(); err != nil {
			return err
		}
	}
	flagGroups := []*flagGroup{a.flagGroup}
	for _, cmd := range a.commandOrder {
		if err := checkDuplicateFlags(cmd, flagGroups); err != nil {
			return err
		}
	}
	a.initialized = true
	return nil
}

// Recursively check commands for duplicate flags.
func checkDuplicateFlags(current *CmdClause, flagGroups []*flagGroup) error {
	// Check for duplicates.
	for _, flags := range flagGroups {
		for _, flag := range current.flagOrder {
			if flag.shorthand != 0 {
				if _, ok := flags.short[string(flag.shorthand)]; ok {
					return fmt.Errorf("duplicate short flag -%c", flag.shorthand)
				}
			}
			if _, ok := flags.long[flag.name]; ok {
				return fmt.Errorf("duplicate long flag --%s", flag.name)
			}
		}
	}
	flagGroups = append(flagGroups, current.flagGroup)
	// Check subcommands.
	for _, subcmd := range current.commandOrder {
		if err := checkDuplicateFlags(subcmd, flagGroups); err != nil {
			return err
		}
	}
	return nil
}

func (a *Application) execute(context *ParseContext, selected []string) (string, error) {
	var err error

	if err = a.validateRequired(context); err != nil {
		return "", err
	}

	if err = a.applyValidators(context); err != nil {
		return "", err
	}

	if err = a.applyActions(context); err != nil {
		return "", err
	}

	command := strings.Join(selected, " ")
	if command == "" && a.cmdGroup.have() {
		return "", ErrCommandNotSpecified
	}
	return command, err
}

func (a *Application) setDefaults(context *ParseContext) error {
	flagElements := map[string]*ParseElement{}
	for _, element := range context.Elements {
		if flag, ok := element.Clause.(*FlagClause); ok {
			if flag.name == "help" {
				return nil
			}
			flagElements[flag.name] = element
		}
	}

	argElements := map[string]*ParseElement{}
	for _, element := range context.Elements {
		if arg, ok := element.Clause.(*ArgClause); ok {
			argElements[arg.name] = element
		}
	}

	// Check required flags and set defaults.
	for _, flag := range context.flags.long {
		if flagElements[flag.name] == nil {
			if err := flag.setDefault(); err != nil {
				return err
			}
		}
	}

	for _, arg := range context.arguments.args {
		if argElements[arg.name] == nil {
			if err := arg.setDefault(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *Application) validateRequired(context *ParseContext) error {
	flagElements := map[string]*ParseElement{}
	for _, element := range context.Elements {
		if flag, ok := element.Clause.(*FlagClause); ok {
			flagElements[flag.name] = element
		}
	}

	argElements := map[string]*ParseElement{}
	for _, element := range context.Elements {
		if arg, ok := element.Clause.(*ArgClause); ok {
			argElements[arg.name] = element
		}
	}

	// Check required flags and set defaults.
	for _, flag := range context.flags.long {
		if flagElements[flag.name] == nil {
			// Check required flags were provided.
			if flag.needsValue() {
				return fmt.Errorf("%w --%s not provided", ErrRequiredFlag, flag.name)
			}
		}
	}

	for _, arg := range context.arguments.args {
		if argElements[arg.name] == nil {
			if arg.needsValue() {
				return fmt.Errorf("%w '%s' not provided", ErrRequiredArgument, arg.name)
			}
		}
	}
	return nil
}

func (a *Application) setValues(context *ParseContext) (selected []string, err error) {
	// Set all arg and flag values.
	var (
		lastCmd *CmdClause
		flagSet = map[string]struct{}{}
	)
	for _, element := range context.Elements {
		switch clause := element.Clause.(type) {
		case *FlagClause:
			if _, ok := flagSet[clause.name]; ok {
				if v, ok := clause.value.(repeatableFlag); !ok || !v.IsCumulative() {
					return nil, fmt.Errorf("flag '%s' %w", clause.name, ErrFlagCannotRepeat)
				}
			}
			if err = clause.value.Set(*element.Value); err != nil {
				return
			}
			flagSet[clause.name] = struct{}{}

		case *ArgClause:
			if err = clause.value.Set(*element.Value); err != nil {
				return
			}

		case *CmdClause:
			selected = append(selected, clause.name)
			lastCmd = clause
		}
	}

	if lastCmd != nil && len(lastCmd.commands) > 0 {
		return nil, fmt.Errorf("%w of '%s'", ErrSubCommandRequired, lastCmd.FullCommand())
	}

	return
}

func (a *Application) applyValidators(context *ParseContext) (err error) {
	// Call command validation functions.
	for _, element := range context.Elements {
		if cmd, ok := element.Clause.(*CmdClause); ok && cmd.validator != nil {
			if err = cmd.validator(cmd); err != nil {
				return err
			}
		}
	}

	if a.validator != nil {
		err = a.validator(a)
	}
	return err
}

func (a *Application) applyPreActions(context *ParseContext, dispatch bool) error {
	if err := a.actionMixin.applyPreActions(context); err != nil {
		return err
	}
	// Dispatch to actions.
	if dispatch {
		for _, element := range context.Elements {
			if applier, ok := element.Clause.(actionApplier); ok {
				if err := applier.applyPreActions(context); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (a *Application) applyActions(context *ParseContext) error {
	if err := a.actionMixin.applyActions(context); err != nil {
		return err
	}
	// Dispatch to actions.
	for _, element := range context.Elements {
		if applier, ok := element.Clause.(actionApplier); ok {
			if err := applier.applyActions(context); err != nil {
				return err
			}
		}
	}
	return nil
}

// Errorf prints an error message to w in the format "<appname>: error: <message>".
func (a *Application) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(a.errorWriter, a.Name+": error: "+format+"\n", args...)
}

// Fatalf writes a formatted error to w then terminates with exit status 1.
func (a *Application) Fatalf(format string, args ...interface{}) {
	a.Errorf(format, args...)
	a.terminate(1)
}

// FatalUsage prints an error message followed by usage information, then
// exits with a non-zero status.
func (a *Application) FatalUsage(format string, args ...interface{}) {
	a.Errorf(format, args...)
	// Force usage to go to error output.
	a.usageWriter = a.errorWriter
	a.Usage([]string{})
	a.terminate(1)
}

// FatalUsageContext writes a printf formatted error message to w, then usage
// information for the given ParseContext, before exiting.
func (a *Application) FatalUsageContext(context *ParseContext, format string, args ...interface{}) {
	a.Errorf(format, args...)
	if err := a.UsageForContext(context); err != nil {
		panic(err)
	}
	a.terminate(1)
}

// FatalIfError prints an error and exits if err is not nil. The error is printed
// with the given formatted string, if any.
func (a *Application) FatalIfError(err error, format string, args ...interface{}) {
	if err == nil {
		return
	}

	prefix := ""
	if format != "" {
		prefix = fmt.Sprintf(format, args...) + ": "
	}
	a.Errorf(prefix+"%s", err)
	a.terminate(1)
}

// MustParseWithUsage parses args using Parse() and shows usage on certain errors
//
// When a parent command with no action is called a compact usage will be shown
// listing the subcommands available without any flags or arguments allowing a
// user to quickly evaluate the next command to use, this is to assist in discovering
// the layout and design of a CLI tool.
//
// When a various argument of flag errors are encountered an error is shown followed by the
// full help for that command showing available arguments and flags.
//
// All other errors just shows the error.
func (a *Application) MustParseWithUsage(args []string) (command string) {
	cmd, err := a.Parse(args)
	if err == nil {
		return cmd
	}

	ut := a.usageTemplate

	switch {
	case errorIs(err, ErrSubCommandRequired):
		fmt.Fprintf(a.errorWriter, "error: a subcommand from the list below is required, use --help for full help including flags and arguments\n\n")
		ut = a.errorUsageTemplate

	case errorIs(err, ErrExpectedKnownCommand):
		fmt.Fprintf(a.errorWriter, "error: %v, use --help for full help including flags and arguments\n\n", err)
		ut = a.errorUsageTemplate

	case errorIs(err, ErrRequiredArgument, ErrRequiredFlag, ErrUnknownLongFlag, ErrUnknownShortFlag, ErrExpectedFlagArgument, ErrFlagCannotRepeat, ErrUnexpectedArgument):
		fmt.Fprintf(a.errorWriter, "error: %v\n\n", err)

	default:
		a.Fatalf("%v", err)
	}

	pc, _ := a.parseContext(true, args)
	a.UsageForContextWithTemplate(pc, 2, ut)
	a.terminate(1)

	return ""
}

func (a *Application) completionOptions(context *ParseContext) []string {
	args := context.rawArgs

	var (
		currArg string
		prevArg string
		target  cmdMixin
	)

	numArgs := len(args)
	if numArgs > 1 {
		args = args[1:]
		currArg = args[len(args)-1]
	}
	if numArgs > 2 {
		prevArg = args[len(args)-2]
	}

	target = a.cmdMixin
	if context.SelectedCommand != nil {
		// A subcommand was in use. We will use it as the target
		target = context.SelectedCommand.cmdMixin
	}

	if (currArg != "" && strings.HasPrefix(currArg, "--")) || strings.HasPrefix(prevArg, "--") {
		if context.argsOnly {
			return nil
		}

		// Perform completion for A flag. The last/current argument started with "-"
		var (
			flagName  string // The name of a flag if given (could be half complete)
			flagValue string // The value assigned to a flag (if given) (could be half complete)
		)

		if strings.HasPrefix(prevArg, "--") && !strings.HasPrefix(currArg, "--") {
			// Matches: 	./myApp --flag value
			// Wont Match: 	./myApp --flag --
			flagName = prevArg[2:] // Strip the "--"
			flagValue = currArg
		} else if strings.HasPrefix(currArg, "--") {
			// Matches: 	./myApp --flag --
			// Matches:		./myApp --flag somevalue --
			// Matches: 	./myApp --
			flagName = currArg[2:] // Strip the "--"
		}

		options, flagMatched, valueMatched := target.FlagCompletion(flagName, flagValue)
		if valueMatched {
			// Value Matched. Show cmdCompletions
			return target.CmdCompletion(context)
		}

		// Add top level flags if we're not at the top level and no match was found.
		if context.SelectedCommand != nil && !flagMatched {
			topOptions, topFlagMatched, topValueMatched := a.FlagCompletion(flagName, flagValue)
			if topValueMatched {
				// Value Matched. Back to cmdCompletions
				return target.CmdCompletion(context)
			}

			if topFlagMatched {
				// Top level had a flag which matched the input. Return its options.
				options = topOptions
			} else {
				// Add top level flags
				options = append(options, topOptions...)
			}
		}
		return options
	}

	// Perform completion for sub commands and arguments.
	return target.CmdCompletion(context)
}

func (a *Application) generateBashCompletion(context *ParseContext) {
	options := a.completionOptions(context)
	fmt.Printf("%s", strings.Join(options, "\n"))
}

func envarTransform(name string) string {
	return strings.ToUpper(envarTransformRegexp.ReplaceAllString(name, "_"))
}

func errorIs(err error, targets ...error) bool {
	for _, t := range targets {
		if errors.Is(err, t) {
			return true
		}
	}

	return false
}
