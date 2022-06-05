package main

import (
	"fmt"
	"os"

	"github.com/choria-io/fisk"
)

// LsCommand context for "ls" command
type LsCommand struct {
	All bool
}

func (l *LsCommand) run(c *fisk.ParseContext) error {
	fmt.Printf("all=%v\n", l.All)
	return nil
}

func configureLsCommand(app *fisk.Application) {
	c := &LsCommand{}
	ls := app.Command("ls", "List files.").Action(c.run)
	ls.Flag("all", "List all files.").Short('a').BoolVar(&c.All)
}

func main() {
	app := fisk.New("modular", "My modular application.")
	configureLsCommand(app)
	fisk.MustParse(app.Parse(os.Args[1:]))
}
