package webhooks

import (
	"fmt"
)

func CommandPing() {
	res, err := SendCmd(Cmd{
		T: CmdPing,
	})

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Printf("response: %s\n", res)
}

// CommandShow implements webhooks and webhooks:show
func CommandShow(app string) {
}

// CommandCreate implements webhooks:create
func CommandCreate(app, hook, command string) {
}

// CommandDelete implements webhooks:delete
func CommandDelete(app, hook string) {}

// CommandDisable implements webhooks:disable
func CommandDisable(app string) {}

// CommandListen implements webhooks:listen
func CommandListen() {}

// CommandLogs implements webhooks:logs
func CommandLogs(app string) {}

// CommandSecret implements webhooks:secret
func CommandSecret(app string) {}

// CommandSetSecret implements webhooks:set-secret
func CommandSetSecret(app, secret string) {}

// CommandStop implements webhooks:stop
func CommandStop() {}

// CommandTrigger implements webhooks:trigger
func CommandTrigger(app, hook string) {}
