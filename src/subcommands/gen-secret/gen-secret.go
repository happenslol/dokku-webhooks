package main

import (
	"flag"
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	args := os.Args[2:]
	webhooks.ExpectArgs(args, "app")
	app := args[0]

	force := flag.Bool("force", false, "overwrite existing")
	flag.Parse()

	forceStr := "false"
	if *force {
		forceStr = "true"
	}

	res, err := webhooks.SendCmd(webhooks.CmdGenSecret, app, forceStr)
	webhooks.PrintResult(res, err)
}
