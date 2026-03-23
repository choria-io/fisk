package fisk

// ShorterMainUsageTemplate is similar to KingpinDefaultUsageTemplate except
// in the main app usage output it does not expand full help text for
// every single sub command all the way to the deepest leve, it also
// does not show global flags in the top app.
//
// Additionally, it supports the new HelpLong on sub commands so one can
// either rely on it always printing just the first line of sub command help
// or only putting short help in the main help and long help in long - the
// long will only be shown when it's rendering usage for that command when it's
// the command the user is executing (select).
//
// This yields a friendlier welcome to new users with the details should
// they do help on any sub command
var ShorterMainUsageTemplate = `{{define "FormatCommand" -}}
{{if .FlagSummary}} {{.FlagSummary}}{{end -}}
{{range .Args}} {{if not .Required}}[{{end}}<{{.Name}}>{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end -}}
{{end -}}

{{define "FormatCommands" -}}
{{range .Commands -}}
{{if not .Hidden -}}
  {{.FullCommand}}{{if .Default}}*{{end}}{{template "FormatCommand" .}}
{{.Help|Wrap 4}}
{{end -}}
{{end -}}
{{end -}}

{{ define "FormatCommandsForTopLevel"  -}}
{{range .Commands -}}
{{if not .Hidden -}}
{{if not (eq .FullCommand "help") -}}
  {{.FullCommand}}{{if .Default}}*{{end}}{{template "FormatCommand" .}}
{{.Help|FirstLine|Wrap 4}}
{{end -}}
{{end -}}
{{end -}}
{{end -}}

{{define "FormatUsage" -}}
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end}}
{{if .Help}}
{{.Help|Wrap 0 -}}
{{end -}}
{{end -}}

{{if .Context.SelectedCommand -}}
usage: {{.App.Name}} {{.Context.SelectedCommand}}{{template "FormatUsage" .Context.SelectedCommand}}
{{if .Context.SelectedCommand.HelpLong}}{{.Context.SelectedCommand.HelpLong|Wrap 0}}
{{end}}
{{else -}}
usage: {{.App.Name}}{{template "FormatUsage" .App}}
{{end -}}
{{if .Context.SelectedCommand -}}
{{if .Context.Flags|VisibleFlags -}}
Flags:
{{.Context.Flags|FlagsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if .Context.Args -}}
Args:
{{.Context.Args|ArgsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if len .Context.SelectedCommand.Commands -}}
Subcommands:
{{template "FormatCommands" .Context.SelectedCommand}}
{{end -}}
{{else if .App.Commands -}}
Commands:
{{template "FormatCommandsForTopLevel" .App}}
{{end -}}
`

// CompactMainUsageTemplate formats commands and subcommands in a two column
// layout to make for a cleaners and more readable usage text. In this format,
// sections are rendered as follows. Global flags are also separate from local
// flags in this template. Global flags will be shown at the top level and local
// flags will be shown when showing help for a subcommand.
//
// usage: <command> [<flags>] <command> [<arg> ...]
//
// # Help text
//
// Commands|Subcommands:
//
//	command1    Help text for command 1
//	command2    Help text for command 2
//	command2    Help text for command 3
//
// Flags:
//
//	-h, --help     Show context-sensitive help
var CompactMainUsageTemplate = `{{define "FormatCommand" -}}
{{if .FlagSummary}} {{.FlagSummary}}{{end -}}
{{range .Args}} {{if not .Required}}[{{end}}<{{.Name}}>{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end -}}
{{end -}}

{{define "FormatUsage" -}}
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end}}
{{if .Help}}
{{.Help|Wrap 0 -}}
{{end -}}
{{end -}}

{{if .Context.SelectedCommand -}}
usage: {{.App.Name}} {{.Context.SelectedCommand}}{{template "FormatUsage" .Context.SelectedCommand}}
{{if .Context.SelectedCommand.HelpLong}}{{.Context.SelectedCommand.HelpLong|Wrap 0}}
{{end -}}
{{else -}}
usage: {{.App.Name}}{{template "FormatUsage" .App}}
{{end -}}
{{if .Context.SelectedCommand -}}
{{if .Context.Args -}}
Args:
{{.Context.Args|ArgsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if len .Context.SelectedCommand.Commands -}}
Subcommands:
{{.Context.SelectedCommand.Commands|CommandsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{else if .App.Commands -}}
Commands:
{{.App.Commands|CommandsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if .Context.SelectedCommand -}}
{{if .Context.SelectedCommand.Flags|VisibleFlags -}}
Flags:
{{.Context.SelectedCommand.Flags|FlagsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{ if len .Context.SelectedCommand.Aliases -}}
Command Aliases: {{ .Context.SelectedCommand.Aliases | Join }}
{{end }}
{{end -}}
{{if GlobalFlags .Context|VisibleFlags -}}
{{if .HelpFlagIsSet -}}
Global Flags:
{{ GlobalFlags .Context|FlagsToTwoColumns|FormatTwoColumns}}
{{else -}}
Pass --help to see global flags applicable to this command.
{{end -}}
{{end -}}
{{if and (IsLLMContext) (.App.LLMExtraInfo) -}}
LLM Information:
{{.App.LLMExtraInfo|Wrap 2}}
{{end -}}
`

// KingpinDefaultUsageTemplate is the default usage template as used by kingpin
var KingpinDefaultUsageTemplate = `{{define "FormatCommand" -}}
{{if .FlagSummary}} {{.FlagSummary}}{{end -}}
{{range .Args}}{{if not .Hidden}} {{if not .Required}}[{{end}}{{if .PlaceHolder}}{{.PlaceHolder}}{{else}}<{{.Name}}>{{end}}{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end}}{{end -}}
{{end -}}

{{define "FormatCommands" -}}
{{range .FlattenedCommands -}}
{{if not .Hidden -}}
  {{.FullCommand}}{{if .Default}}*{{end}}{{template "FormatCommand" .}}
{{.Help|Wrap 4}}
{{end -}}
{{end -}}
{{end -}}

{{define "FormatUsage" -}}
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end}}
{{if .Help}}
{{.Help|Wrap 0 -}}
{{end -}}

{{end -}}

{{if .Context.SelectedCommand -}}
usage: {{.App.Name}} {{.Context.SelectedCommand}}{{template "FormatUsage" .Context.SelectedCommand}}
{{else -}}
usage: {{.App.Name}}{{template "FormatUsage" .App}}
{{end -}}
{{if .Context.Flags|VisibleFlags -}}
Flags:
{{.Context.Flags|FlagsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if .Context.Args -}}
Args:
{{.Context.Args|ArgsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if .Context.SelectedCommand -}}
{{if len .Context.SelectedCommand.Commands -}}
Subcommands:
{{template "FormatCommands" .Context.SelectedCommand}}
{{end -}}
{{else if .App.Commands -}}
Commands:
{{template "FormatCommands" .App}}
{{end -}}
`

// SeparateOptionalFlagsUsageTemplate is a usage template where command's optional flags are listed separately
var SeparateOptionalFlagsUsageTemplate = `{{define "FormatCommand" -}}
{{if .FlagSummary}} {{.FlagSummary}}{{end -}}
{{range .Args}}{{if not .Hidden}} {{if not .Required}}[{{end}}{{if .PlaceHolder}}{{.PlaceHolder}}{{else}}<{{.Name}}>{{end}}{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end}}{{end -}}
{{end -}}

{{define "FormatCommands" -}}
{{range .FlattenedCommands -}}
{{if not .Hidden -}}
  {{.FullCommand}}{{if .Default}}*{{end}}{{template "FormatCommand" .}}
{{.Help|Wrap 4}}
{{end -}}
{{end -}}
{{end -}}

{{define "FormatUsage" -}}
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end}}
{{if .Help}}
{{.Help|Wrap 0 -}}
{{end -}}

{{end -}}
{{if .Context.SelectedCommand -}}
usage: {{.App.Name}} {{.Context.SelectedCommand}}{{template "FormatUsage" .Context.SelectedCommand}}
{{else -}}
usage: {{.App.Name}}{{template "FormatUsage" .App}}
{{end -}}

{{if .Context.Flags|RequiredFlags -}}
Required flags:
{{.Context.Flags|RequiredFlags|FlagsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if  .Context.Flags|OptionalFlags -}}
Optional flags:
{{.Context.Flags|OptionalFlags|FlagsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if .Context.Args -}}
Args:
{{.Context.Args|ArgsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if .Context.SelectedCommand -}}
Subcommands:
{{if .Context.SelectedCommand.Commands -}}
{{template "FormatCommands" .Context.SelectedCommand}}
{{end -}}
{{else if .App.Commands -}}
Commands:
{{template "FormatCommands" .App}}
{{end -}}
`

// CompactUsageTemplate is a usage template with compactly formatted commands.
var CompactUsageTemplate = `{{define "FormatCommand" -}}
{{if .FlagSummary}} {{.FlagSummary}}{{end -}}
{{range .Args}}{{if not .Hidden}} {{if not .Required}}[{{end}}{{if .PlaceHolder}}{{.PlaceHolder}}{{else}}<{{.Name}}>{{end}}{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end}}{{end -}}
{{end -}}

{{define "FormatCommandList" -}}
{{range . -}}
{{if not .Hidden -}}
{{.Depth|Indent}}{{.Name}}{{if .Default}}*{{end}}{{template "FormatCommand" .}}
{{end -}}
{{template "FormatCommandList" .Commands -}}
{{end -}}
{{end -}}

{{define "FormatUsage" -}}
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end}}
{{if .Help}}
{{.Help|Wrap 0 -}}
{{end -}}

{{end -}}

{{if .Context.SelectedCommand -}}
usage: {{.App.Name}} {{.Context.SelectedCommand}}{{template "FormatUsage" .Context.SelectedCommand}}
{{else -}}
usage: {{.App.Name}}{{template "FormatUsage" .App}}
{{end -}}
{{if .Context.Flags|VisibleFlags -}}
Flags:
{{.Context.Flags|FlagsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if .Context.Args -}}
Args:
{{.Context.Args|ArgsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if .Context.SelectedCommand -}}
{{if .Context.SelectedCommand.Commands -}}
Commands:
  {{.Context.SelectedCommand}}
{{template "FormatCommandList" .Context.SelectedCommand.Commands}}
{{end -}}
{{else if .App.Commands -}}
Commands:
{{template "FormatCommandList" .App.Commands}}
{{end -}}
`

// LLMHelpTemplate is a usage template that renders help in Markdown format
// suitable for consumption by Large Language Models.
var LLMHelpTemplate = `{{define "FormatCommand" -}}
{{if .FlagSummary}} {{.FlagSummary}}{{end -}}
{{range .Args}}{{if not .Hidden}} {{if not .Required}}[{{end}}{{if .PlaceHolder}}{{.PlaceHolder}}{{else}}<{{.Name}}>{{end}}{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end}}{{end -}}
{{end -}}

{{define "FormatUsage" -}}
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end}}
{{end -}}

{{define "LLMFormatFlags" -}}
| Flag | Description | Type | Default | Required | Env Var |
|------|-------------|------|---------|----------|---------|
{{range . -}}
{{if not .Hidden -}}
| {{FormatFlagName .}} | {{.Help|EscapeMDTable}} | ` + "`" + `{{FlagType .|EscapeMDTable}}` + "`" + ` | {{if .Default}}` + "`" + `{{FlagDefault .Default|EscapeMDTable}}` + "`" + `{{end}} | {{if .Required}}Yes{{else}}No{{end}}{{if .IsCumulative}}, repeatable{{end}} | {{if .Envar}}` + "`" + `{{.Envar}}` + "`" + `{{end}} |
{{end -}}
{{end -}}
{{end -}}

{{define "LLMFormatArgs" -}}
| Argument | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
{{range . -}}
{{if not .Hidden -}}
| ` + "`" + `{{.Name}}` + "`" + ` | {{.Help|EscapeMDTable}} | ` + "`" + `{{ArgType .|EscapeMDTable}}` + "`" + ` | {{if .Default}}` + "`" + `{{FlagDefault .Default|EscapeMDTable}}` + "`" + `{{end}} | {{if .Required}}Yes{{else}}No{{end}}{{if .IsCumulative}}, repeatable{{end}} |
{{end -}}
{{end -}}
{{end -}}

{{define "LLMFormatCommands" -}}
| Command | Description |{{if .HasTags}} Tags |{{end}}
|---------|-------------|{{if .HasTags}}------|{{end}}
{{range .Commands -}}
{{if not .Hidden -}}
{{if ne .FullCommand "help" -}}
| ` + "`" + `{{.FullCommand}}` + "`" + `{{if .Default}} (default){{end}} | {{.Help|FirstLine|EscapeMDTable}} |{{if .Tags}} {{.Tags|Join}} |{{end}}
{{end -}}
{{end -}}
{{end -}}
{{end -}}

{{if .Context.SelectedCommand -}}
# {{.App.Name}} {{.Context.SelectedCommand}}
{{else -}}
# {{.App.Name}}
{{end -}}

{{if .Context.SelectedCommand -}}
{{.Context.SelectedCommand.Help}}
{{if .Context.SelectedCommand.HelpLong}}
{{.Context.SelectedCommand.HelpLong}}
{{end -}}
{{else -}}
{{.App.Help}}
{{end -}}
{{if .Context.SelectedCommand -}}
{{if .Context.SelectedCommand.Tags -}}

**Tags:** {{.Context.SelectedCommand.Tags|Join}}
{{end -}}
{{if .Context.SelectedCommand.Aliases -}}

**Aliases:** {{.Context.SelectedCommand.Aliases|Join}}
{{end -}}
{{end -}}

## Usage

` + "```" + `
{{if .Context.SelectedCommand -}}
{{.App.Name}} {{.Context.SelectedCommand}}{{template "FormatUsage" .Context.SelectedCommand}}
{{- else -}}
{{.App.Name}}{{template "FormatUsage" .App}}
{{- end}}
` + "```" + `
{{if .Context.SelectedCommand -}}
{{if .Context.Args -}}

## Arguments

{{template "LLMFormatArgs" .Context.Args}}
{{end -}}
{{if .Context.SelectedCommand.Flags|VisibleFlags -}}

## Flags

{{template "LLMFormatFlags" (.Context.SelectedCommand.Flags|VisibleFlags)}}
{{end -}}
{{if len .Context.SelectedCommand.Commands -}}

## Subcommands

{{template "LLMFormatCommands" .Context.SelectedCommand}}
{{end -}}
{{else if .App.Commands -}}

## Commands

{{template "LLMFormatCommands" .App}}
{{end -}}
{{if .Context.SelectedCommand -}}
{{if GlobalFlags .Context|VisibleFlags -}}

## Global Flags

{{template "LLMFormatFlags" (GlobalFlags .Context|VisibleFlags)}}
{{end -}}
{{end -}}
{{if .App.LLMExtraInfo -}}

## Additional Information

{{.App.LLMExtraInfo}}
{{end -}}
`

// ManPageTemplate renders usage in unix man format
var ManPageTemplate = `{{define "FormatFlags" -}}
{{range .Flags -}}
{{if not .Hidden -}}
.TP
\fB{{if .Short}}-{{.Short|Char}}, {{end}}--{{.Name}}{{if not .IsBoolFlag}}={{.FormatPlaceHolder}}{{end -}}\fR
{{.Help}}
{{end -}}
{{end -}}
{{end -}}

{{define "FormatCommand" -}}
{{if .FlagSummary}} {{.FlagSummary}}{{end -}}
{{range .Args}}{{if not .Hidden}} {{if not .Required}}[{{end}}{{if .PlaceHolder}}{{.PlaceHolder}}{{else}}<{{.Name}}>{{end}}{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end}}{{end -}}
{{end -}}

{{define "FormatCommands" -}}
{{range .FlattenedCommands -}}
{{if not .Hidden -}}
.SS
\fB{{.FullCommand}}{{template "FormatCommand" . -}}\fR
.PP
{{.Help}}
{{template "FormatFlags" . -}}
{{end -}}
{{end -}}
{{end -}}

{{define "FormatUsage" -}}
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end -}}\fR
{{end -}}

.TH {{.App.Name}} 1 {{.App.Version}} "{{.App.Author}}"
.SH "NAME"
{{.App.Name}}
.SH "SYNOPSIS"
.TP
\fB{{.App.Name}}{{template "FormatUsage" .App}}
.SH "DESCRIPTION"
{{.App.Help}}
.SH "OPTIONS"
{{template "FormatFlags" .App -}}
{{if .App.Commands -}}
.SH "COMMANDS"
{{template "FormatCommands" .App -}}
{{end -}}
`

// LongHelpTemplate is a usage template for --help-long
var LongHelpTemplate = `{{define "FormatCommand" -}}
{{if .FlagSummary}} {{.FlagSummary}}{{end -}}
{{range .Args}}{{if not .Hidden}} {{if not .Required}}[{{end}}{{if .PlaceHolder}}{{.PlaceHolder}}{{else}}<{{.Name}}>{{end}}{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end}}{{end -}}
{{end -}}

{{define "FormatCommands" -}}
{{range .FlattenedCommands -}}
{{if not .Hidden -}}
  {{.FullCommand}}{{template "FormatCommand" .}}
{{.Help|Wrap 4}}
{{with .Flags|FlagsToTwoColumns}}{{FormatTwoColumnsWithIndent . 4 2}}{{end}}
{{end -}}
{{end -}}
{{end -}}

{{define "FormatUsage" -}}
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end}}
{{if .Help}}
{{.Help|Wrap 0 -}}
{{end -}}

{{end -}}

usage: {{.App.Name}}{{template "FormatUsage" .App}}
{{if .Context.Flags|VisibleFlags -}}
Flags:
{{.Context.Flags|FlagsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if .Context.Args -}}
Args:
{{.Context.Args|ArgsToTwoColumns|FormatTwoColumns}}
{{end -}}
{{if .App.Commands -}}
Commands:
{{template "FormatCommands" .App}}
{{end -}}
`

var BashCompletionTemplate = `
_{{.App.Name}}_bash_autocomplete() {
    local cur prev opts base
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    opts=$( ${COMP_WORDS[0]} --completion-bash "${COMP_WORDS[@]:1:$COMP_CWORD}" )
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}
complete -F _{{.App.Name}}_bash_autocomplete -o default {{.App.Name}}

`

var ZshCompletionTemplate = `#compdef {{.App.Name}}

# Zsh completion for {{.App.Name}}
# Dynamically generated from --completion-zsh-menu output
#
# Installation:
#   Option 1: Copy to your fpath
#     {{.App.Name}} --completion-script-zsh > /usr/local/share/zsh/site-functions/_{{.App.Name}}
#
#   Option 2: Source in .zshrc
#     source <({{.App.Name}} --completion-script-zsh)
#
#   Option 3: Save to a file and add its directory to fpath in .zshrc (before compinit)
#     fpath=(/path/to/completions $fpath)
#     autoload -Uz compinit && compinit

# Enable menu selection with highlighting
# Preserve the order emitted by the command (most common first)
zmodload zsh/complist
zstyle ':completion:*:{{.App.Name}}:*' menu select=1 interactive
zstyle ':completion:*:{{.App.Name}}:*' sort false
zstyle ':completion:*:{{.App.Name}}:*' group-name ''
zstyle ':completion:*:{{.App.Name}}:*:descriptions' format '%B%F{cyan}-- %d --%f%b'

# Matching: case-insensitive, then substring, then partial-word at hyphens
zstyle ':completion:*:{{.App.Name}}:*' matcher-list \
    'm:{a-zA-Z}={A-Za-z}' \
    'm:{a-zA-Z}={A-Za-z} l:|=* r:|=*' \
    'm:{a-zA-Z}={A-Za-z} r:|[-]=** l:|=*'

_{{.App.Name}}() {
    local curcontext="$curcontext"
    local cur prev flagname
    local -i i j skip_next pos_count arg_idx cmd_depth
    local lt field1 field2 field3 field4 field5 field6
    local fname fshort fplace fneg fdesc fenums

    local -a cmd_path
    local -a cmds cdesc
    local -a fnames fshorts fplaceholders fnegatables fdescs fenumvals
    local -a anames arequireds adescs
    local -a flag_list cmd_list enum_list

    # Build the command path: extract non-flag words between the program
    # name (word 1) and the cursor position.  Flags and their values are
    # skipped so that we resolve the right subcommand context.
    i=2 skip_next=0
    while (( i < CURRENT )); do
        if (( skip_next )); then
            skip_next=0
            (( i++ ))
            continue
        fi
        case "${words[i]}" in
            --*=*) ;;           # --flag=value is self-contained
            --*)                # long flag, maybe followed by a value
                if (( i + 1 < CURRENT )) && [[ "${words[i+1]}" != -* ]]; then
                    skip_next=1
                fi ;;
            -?)                 # single short flag like -s
                if (( i + 1 < CURRENT )) && [[ "${words[i+1]}" != -* ]]; then
                    skip_next=1
                fi ;;
            *)  cmd_path+=("${words[i]}") ;;
        esac
        (( i++ ))
    done

    # Fetch machine-readable completion data for the resolved context
    local comp_output
    comp_output="$({{.App.Name}} "${cmd_path[@]}" --completion-zsh-menu 2>/dev/null)"
    [[ -z "$comp_output" ]] && return 1

    # Parse the tab-separated output into arrays
    cmd_depth=0
    while IFS=$'\t' read -r lt field1 field2 field3 field4 field5 field6; do
        case "$lt" in
            D) cmd_depth=$field1 ;;
            C) cmds+=("$field1");   cdesc+=("$field2") ;;
            F) fnames+=("$field1"); fshorts+=("$field2")
               fplaceholders+=("$field3"); fnegatables+=("$field4")
               fdescs+=("$field5"); fenumvals+=("$field6") ;;
            A) anames+=("$field1"); arequireds+=("$field2")
               adescs+=("$field3") ;;
        esac
    done <<< "$comp_output"

    cur="${words[CURRENT]}"

    # --- Check if we are completing a flag value ---
    # If the previous word is a flag that takes a value, complete the value.
    if (( CURRENT > 2 )); then
        prev="${words[CURRENT-1]}"
        if [[ "$prev" == --* || "$prev" == -? ]]; then
            flagname=""
            if [[ "$prev" == --* ]]; then
                flagname="${prev#--}"
            else
                # Find the long name for this short flag
                for (( j = 1; j <= ${#fnames}; j++ )); do
                    if [[ "-${fshorts[j]}" == "$prev" ]]; then
                        flagname="${fnames[j]}"
                        break
                    fi
                done
            fi
            if [[ -n "$flagname" ]]; then
                for (( j = 1; j <= ${#fnames}; j++ )); do
                    if [[ "${fnames[j]}" == "$flagname" && "${fplaceholders[j]}" != "_" ]]; then
                        # If this flag has enum values, offer them as completions
                        if [[ -n "${fenumvals[j]}" ]]; then
                            enum_list=()
                            for fenums in ${(s:,:)fenumvals[j]}; do
                                enum_list+=("$fenums")
                            done
                            _describe -t enum-values "${fplaceholders[j]}" enum_list -o nosort && return 0
                        fi
                        case "${fplaceholders[j]}" in
                            FILE) _files && return 0 ;;
                            DIR)  _directories && return 0 ;;
                            *)    _message -r "${fplaceholders[j]}" && return 0 ;;
                        esac
                    fi
                done
            fi
        fi
    fi

    # --- Build the flag completion list (reused below) ---
    for (( j = 1; j <= ${#fnames}; j++ )); do
        fname="${fnames[j]}"
        fshort="${fshorts[j]}"
        fneg="${fnegatables[j]}"
        fdesc="${fdescs[j]}"

        [[ "$fname" == "help" || "$fname" == "version" ]] && continue
        [[ "$fname" == completion-* ]] && continue

        flag_list+=("--${fname}:${fdesc}")

        if [[ "$fneg" == "true" ]]; then
            flag_list+=("--no-${fname}:Disable ${fname}")
        fi

        if [[ "$fshort" != "_" ]]; then
            flag_list+=("-${fshort}:${fdesc}")
        fi
    done

    # --- Flag completion (current word starts with -) ---
    if [[ "$cur" == -* ]]; then
        _describe -t flags 'flags' flag_list -o nosort && return 0
        return 1
    fi

    # --- Command / subcommand completion ---
    if (( ${#cmds} > 0 )); then
        for (( j = 1; j <= ${#cmds}; j++ )); do
            [[ "${cmds[j]}" == "help" ]] && continue
            cmd_list+=("${cmds[j]}:${cdesc[j]}")
        done

        _describe -t commands 'commands' cmd_list -o nosort && return 0
    fi

    # --- Argument completion ---
    # Use command depth from the D line to know how many non-flag words
    # are commands vs positional args.  cmd_path contains ALL non-flag
    # words (commands + args), but only cmd_depth of them are commands.
    pos_count=$(( ${#cmd_path} - cmd_depth ))

    arg_idx=$(( pos_count + 1 ))

    if (( arg_idx >= 1 && arg_idx <= ${#anames} )); then
        _message -r "${anames[arg_idx]}: ${adescs[arg_idx]}"
        return 0
    fi

    return 1
}

if [[ "$(basename -- ${(%):-%x})" != "_{{.App.Name}}" ]]; then
    compdef _{{.App.Name}} {{.App.Name}}
fi
`

// ZshMenuCompletionDataTemplate outputs tab-separated completion data for the zsh completion script.
//
// Format:
//
//	D\tdepth                                         - command depth (number of command words consumed)
//	C\tname\tdescription                               - command/subcommand
//	F\tname\tshort\tplaceholder\tnegatable\tdesc\tenum - flag (short="_" if none, placeholder="_" if bool, enum=comma-separated or empty)
//	A\tname\trequired|optional\tdescription            - positional argument
var ZshMenuCompletionDataTemplate = `
{{- if .Context.SelectedCommand -}}
D	{{ .Context.SelectedCommand.Depth }}
{{ range .Context.SelectedCommand.Commands -}}
{{- if not .Hidden }}C	{{ .Name }}	{{ ZshEscape (FirstLine .Help) }}
{{ end -}}
{{- end -}}
{{- range VisibleFlags .Context.SelectedCommand.Flags -}}
F	{{ .Name }}	{{ if .Short }}{{ Char .Short }}{{ else }}_{{ end }}	{{ if .Boolean }}_{{ else }}{{ .FormatPlaceHolder }}{{ end }}	{{ .Negatable }}	{{ ZshEscape (FirstLine .Help) }}	{{ JoinCompletions .Completions }}
{{ end -}}
{{- range VisibleFlags (GlobalFlags .Context) -}}
F	{{ .Name }}	{{ if .Short }}{{ Char .Short }}{{ else }}_{{ end }}	{{ if .Boolean }}_{{ else }}{{ .FormatPlaceHolder }}{{ end }}	{{ .Negatable }}	{{ ZshEscape (FirstLine .Help) }}	{{ JoinCompletions .Completions }}
{{ end -}}
{{- range .Context.SelectedCommand.Args -}}
{{- if not .Hidden }}A	{{ .Name }}	{{ if .Required }}required{{ else }}optional{{ end }}	{{ ZshEscape (FirstLine .Help) }}
{{ end -}}
{{- end -}}
{{- else -}}
D	0
{{ range .App.Commands -}}
{{- if not .Hidden }}C	{{ .Name }}	{{ ZshEscape (FirstLine .Help) }}
{{ end -}}
{{- end -}}
{{- range VisibleFlags .App.Flags -}}
F	{{ .Name }}	{{ if .Short }}{{ Char .Short }}{{ else }}_{{ end }}	{{ if .Boolean }}_{{ else }}{{ .FormatPlaceHolder }}{{ end }}	{{ .Negatable }}	{{ ZshEscape (FirstLine .Help) }}	{{ JoinCompletions .Completions }}
{{ end -}}
{{- end -}}`
