package main

import (
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	args := os.Args[2:]
	webhooks.ExpectArgs(args, "app", "secret")
	app, secret := args[0], args[1]
	webhooks.CommandSetSecret(app, secret)
}
