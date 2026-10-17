// Package cli is the error contract of the subcommands: user-facing
// failures and exit codes.
package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
)

// Fail is a user-facing failure; Code prints it as "Ошибка: ...".
func Fail(format string, a ...any) error { return fmt.Errorf(format, a...) }

// Exit ends a command with a status and no message (usage already shown).
type Exit int

func (c Exit) Error() string { return "exit " + strconv.Itoa(int(c)) }

// Code turns a command's result into the exit status, printing the error.
func Code(err error, stderr io.Writer) int {
	var code Exit
	switch {
	case err == nil:
		return 0
	case errors.As(err, &code):
		return int(code)
	}
	fmt.Fprintf(stderr, "Ошибка: %v\n", err)
	return 1
}
