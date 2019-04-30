package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	dokku "github.com/dokku/dokku/plugins/common"
	webhooks "github.com/happenslol/dokku-webhooks"
	columnize "github.com/ryanuber/columnize"
)

const (
	helpHeader = `Usage: dokku webhooks <app>

List registered webhooks for an app

Additional commands:`

	helpContent = `
    webhooks:show <app>, List registered webhooks for an app
    webhooks:listen, Start the webhook server
    webhooks:auto-listen, Automatically keep the webhook server running
    webhooks:stop, Stop the webhook server
    webhooks:secret <app>, Print the secret for an app
    webhooks:set-secret <app> <secret>, Set the secret for an app
    webhooks:enable <app>, Enable all webhooks for an app
    webhooks:disable <app>, Disable all webhooks for an app
    webhooks:create <app> <name> <command>, Create a webhook
    webhooks:delete <app> <name>, Delete a webhook
    webhooks:trigger <app> <name>, Manually trigger a webhook
    webhooks:logs <app>, Show webhook activation logs for an app
`
)

func main() {
	cmd := os.Args[1]
	// args := os.Args[2:]

	switch cmd {
	case "webhooks", "webhooks:show":
		// webhooks.ExpectArgs(args, "app")
		// app := args[0]
		// webhooks.CommandShow(app)
		// run command
		webhooks.CommandPing()
	case "webhooks:help":
		usage()
	case "help":
		command := dokku.NewShellCmd(fmt.Sprintf("ps -o command= %d", os.Getppid()))
		command.ShowOutput = false
		output, err := command.Output()

		if err == nil && strings.Contains(string(output), "--all") {
			fmt.Println(helpContent)
		} else {
			fmt.Print("\n    webhooks, Plugin for running dokku commands through endpoints\n")
		}
	default:
		dokkuNotImplementExitCode, err := strconv.Atoi(os.Getenv("DOKKU_NOT_IMPLEMENTED_EXIT"))
		if err != nil {
			fmt.Println("failed to retrieve DOKKU_NOT_IMPLEMENTED_EXIT environment variable")
			dokkuNotImplementExitCode = 10
		}
		os.Exit(dokkuNotImplementExitCode)
	}
}

func usage() {
	config := columnize.DefaultConfig()
	config.Delim = ","
	config.Prefix = "    "
	config.Empty = ""
	content := strings.Split(helpContent, "\n")[1:]
	fmt.Println(helpHeader)
	fmt.Println(columnize.Format(content, config))
}
