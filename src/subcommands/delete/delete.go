package main

import (
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	args := os.Args[2:]
	webhooks.ExpectArgs(args, "app", "hook")
	app, hook := args[0], args[1]
	res, err := webhooks.SendCmd(webhooks.CmdDelete, app, hook)
	webhooks.PrintResult(res, err)
}
