package main

import (
	"fmt"
	"os"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func main() {
	args := os.Args[2:]
	webhooks.ExpectArgs(args, "app")
	app := args[0]

	res, err := webhooks.SendCmd(webhooks.CmdEnableApp, app)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Printf("response: %s\n", res)
}
