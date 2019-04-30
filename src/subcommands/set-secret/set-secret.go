package main

import (
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	args := os.Args[2:]
	webhooks.ExpectArgs(args, "app", "secret")
	app, secret := args[0], args[1]
	res, err := webhooks.SendCmd(webhooks.CmdSetSecret, app, secret)
	webhooks.PrintResult(res, err)
}
