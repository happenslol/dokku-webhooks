package main

import (
    "github.com/dokku/dokku/plugins/common"
    "github.com/happenslol/dokku-webhooks/webhooks"
)

func main() {
    args := os.Args

	if len(args) > 2 {
		common.LogFail(fmt.Sprintf("Unexpected argument(s): %v", args))
	}
	if len(args) == 0 {
		common.LogFail("Expected: <app> <hook>")
	}

    app := os.Args(0)
    hook := os.Args(1)

    webhooks.CommandTrigger(app, hook)
}
