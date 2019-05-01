package main

import (
	"flag"
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	args := os.Args[2:]
	webhooks.ExpectArgs(args, "app", "secret")
	app, secret := args[0], args[1]

	force := flag.Bool("force", false, "overwrite existing")
	flag.Parse()

	forceStr := "false"
	if *force {
		forceStr = "true"
	}

	res, err := webhooks.SendCmd(webhooks.CmdSetSecret, app, secret, forceStr)
	webhooks.PrintResult(res, err)
}
