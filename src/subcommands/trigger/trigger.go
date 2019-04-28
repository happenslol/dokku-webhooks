package main

import (
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	webhooks.ExpectArgs(os.Args, "app", "hook")
	app, hook := os.Args[0], os.Args[1]
	webhooks.CommandTrigger(app, hook)
}
