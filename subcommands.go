package webhooks

import (
	"log"
)

// CommandCreate implements webhooks:create
func CommandCreate(app, hook, command string) {
	log.Printf("running create")
}

// CommandDelete implements webhooks:delete
func CommandDelete(app, hook string) {}

// CommandDisable implements webhooks:disable
func CommandDisable(app string) {}

// CommandEnable implements webhooks:enable
func CommandEnable(app string) {}

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
