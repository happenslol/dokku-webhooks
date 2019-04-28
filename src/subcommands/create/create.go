package main

import (
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	args := os.Args[2:]
	webhooks.ExpectArgs(args, "app", "hook", "command")
	app, hook, command := args[0], args[1], args[2]
	webhooks.CommandCreate(app, hook, command)
}
