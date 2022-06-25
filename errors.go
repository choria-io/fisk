package fisk

import (
	"errors"
	"fmt"
)

var (
	// ErrUnknownLongFlag indicates an unknown long form flag was passed
	ErrUnknownLongFlag = errors.New("unknown long flag")

	// ErrUnknownShortFlag indicates an unknown short form flag was passed
	ErrUnknownShortFlag = errors.New("unknown short flag")

	// ErrExpectedFlagArgument indicates a flag requiring an argument did not have one supplied
	ErrExpectedFlagArgument = errors.New("expected argument for flag")

	// ErrCommandNotSpecified indicates a command was expected
	ErrCommandNotSpecified = fmt.Errorf("command not specified")

	// ErrSubCommandRequired indicates that a command was invoked, but it required a sub command
	ErrSubCommandRequired = errors.New("must select a subcommand")

	// ErrRequiredArgument indicates a required argument was not given
	ErrRequiredArgument = errors.New("required argument")

	// ErrRequiredFlag indicates a required flag was not given
	ErrRequiredFlag = errors.New("required flag")

	// ErrExpectedKnownCommand indicates that an unknown command argument was encountered
	ErrExpectedKnownCommand = errors.New("expected command")

	// ErrFlagCannotRepeat indicates a flag cannot be passed multiple times to fill an array
	ErrFlagCannotRepeat = errors.New("cannot be repeated")
)
