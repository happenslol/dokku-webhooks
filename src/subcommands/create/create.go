package main

import (
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	webhooks.ExpectArgs(os.Args, "app", "hook", "command")
	app, hook, command := os.Args[0], os.Args[1], os.Args[2]
	webhooks.CommandCreate(app, hook, command)
}
