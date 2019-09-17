package commands

import (
	"strings"

	route53 "github.com/rancher/rdns-server/commands/aws"
	"github.com/rancher/rdns-server/commands/coredns"
	"github.com/rancher/rdns-server/commands/global"

	"github.com/c-bata/go-prompt"
	"github.com/urfave/cli"
)

var (
	Commands = map[string]cli.Command{
		"route53": {
			Name:   "route53",
			Usage:  "use aws route53 provider",
			Flags:  route53.Flags(),
			Action: route53.Action,
		},
		"coredns": {
			Name:   "coredns",
			Usage:  "use coredns provider",
			Flags:  coredns.Flags(),
			Action: coredns.Action,
		},
	}
	Flags = global.Flags
)

func Completer(d prompt.Document) []prompt.Suggest {
	if d.TextBeforeCursor() == "" {
		return []prompt.Suggest{}
	}

	args := strings.Split(d.TextBeforeCursor(), " ")
	w := d.GetWordBeforeCursor()

	// if PIPE is in text before the cursor, returns empty suggestions.
	for i := range args {
		if args[i] == "|" {
			return []prompt.Suggest{}
		}
	}

	// if word before the cursor starts with "-", returns CLI flag options.
	if strings.HasPrefix(w, "-") {
		return optionCompleter(args, strings.HasPrefix(w, "--"))
	}

	return argumentsCompleter(excludeOptions(args))
}

func argumentsCompleter(args []string) []prompt.Suggest {
	suggests := make([]prompt.Suggest, 0)
	for name, command := range Commands {
		if command.Name != "prompt" {
			suggests = append(suggests, prompt.Suggest{
				Text:        name,
				Description: command.Usage,
			})
		}
	}

	if len(args) <= 1 {
		return prompt.FilterHasPrefix(suggests, args[0], true)
	}

	switch args[0] {
	case "route53":
		if len(args) == 2 {
			subCommands := make([]prompt.Suggest, 0)
			return prompt.FilterHasPrefix(subCommands, args[1], true)
		}
	case "coredns":
		if len(args) == 2 {
			subCommands := make([]prompt.Suggest, 0)
			return prompt.FilterHasPrefix(subCommands, args[1], true)
		}
	default:
		if len(args) == 2 {
			return prompt.FilterHasPrefix(getSubCommandSuggest(args[0]), args[1], true)
		}
	}
	return []prompt.Suggest{}
}

func getSubCommandSuggest(name string) []prompt.Suggest {
	subCommands := make([]prompt.Suggest, 0)
	for _, com := range Commands[name].Subcommands {
		subCommands = append(subCommands, prompt.Suggest{
			Text:        com.Name,
			Description: com.Usage,
		})
	}
	return subCommands
}
