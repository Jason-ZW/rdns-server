package commands

import (
	"fmt"

	"github.com/c-bata/go-prompt"
	"github.com/urfave/cli"
)

func PromptCommand() cli.Command {
	return cli.Command{
		Name:   "prompt",
		Usage:  "enter into auto-prompt mode",
		Action: promptAction,
		Flags:  []cli.Flag{},
	}
}

func promptAction(ctx *cli.Context) error {
	fmt.Println("rdns-server cli auto-completion mode")
	defer fmt.Println("exit...")
	p := prompt.New(
		Executor,
		Completer,
		prompt.OptionTitle("rdns-server prompt: interactive rdns cli"),
		prompt.OptionPrefix("rdns-server >>> "),
		prompt.OptionInputTextColor(prompt.Yellow),
		prompt.OptionMaxSuggestion(20),
	)
	p.Run()
	return nil
}
