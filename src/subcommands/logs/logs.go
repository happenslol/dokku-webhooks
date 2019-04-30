package main

import (
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	args := os.Args[2:]
	webhooks.ExpectArgs(args, "app")
	app := args[0]
	res, err := webhooks.SendCmd(webhooks.CmdLogs, app)
	webhooks.PrintResult(res, err)
}
