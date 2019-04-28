package main

import (
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	webhooks.ExpectArgs(os.Args)
	webhooks.CommandStop()
}
