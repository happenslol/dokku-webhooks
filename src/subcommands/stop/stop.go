package main

import (
    "github.com/dokku/dokku/plugins/common"
    "github.com/happenslol/dokku-webhooks/webhooks"
)

func main() {
    args := os.Args

	if len(args) > 0 {
		common.LogFail(fmt.Sprintf("Unexpected argument(s): %v", args))
	}

    webhooks.CommandStop()
}
