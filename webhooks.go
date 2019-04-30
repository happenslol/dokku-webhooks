package webhooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	dokku "github.com/dokku/dokku/plugins/common"
)

// CmdType defines which command will be executed
type CmdType int

// Cmd represents an input sent from the cli
type Cmd struct {
	T    CmdType  `json:"t"`
	Args []string `json:"args,omitempty"`
}

// Response will be sent back from the server when a
// Cmd is received
type Response struct {
	Status  int    `json:"status"`
	Content string `json:"content,omitempty"`
}

const (
	webhooksDir = "/var/lib/dokku/data/webhooks"
	cmdSocket   = "/var/lib/dokku/data/webhooks/cmd.sock"

	// CmdPing pings the webhooks server to check its health.
	CmdPing CmdType = iota
	// CmdShowApp returns a list of all webhooks and their status
	// for a specific app.
	// * app name
	CmdShowApp
	// CmdEnableApp activates webhooks for an app.
	// * app name
	CmdEnableApp
	// CmdDisableApp deactivates webhooks for an app.
	// * app name
	CmdDisableApp
	// CmdCreate creates a webhook.
	// * app name
	// * webhook name
	// * command
	CmdCreate
	// CmdDelete deletes a webhook.
	// * app name
	// * webhook name
	CmdDelete
	// CmdSetSecret sets the secret for an app
	CmdSetSecret
	// CmdGenSecret generates a random secret for an app
	CmdGetSecret
	// CmdTrigger manually triggers a webhook as if its endpoint
	// was called with the correct secret.
	// * app name
	// * webhook name
	CmdTrigger
	// CmdLogs returns a list of activations for a specific webhook.
	// * app name
	CmdLogs
	// CmdQuit shuts down the server process.
	CmdQuit
)

// SendCmd sends a message to the command socket and return the response as
// a string which can be printed out as-is
func SendCmd(cmd Cmd) (string, error) {
	if !dokku.DirectoryExists(webhooksDir) {
		// TODO(happens): Tell user how to enable webhooks
		// NOTE(happens): The directory won't exist if webhooks haven't
		// ever been enabled, the cmdSocket won't exist if the server
		// is not currently running
		dokku.LogFail("webhooks are not enabled!")
	}

	if _, err := os.Stat(cmdSocket); err != nil {
		dokku.LogFail("webhooks server is not running!")
	}

	c, err := net.Dial("unix", cmdSocket)
	if err != nil {
		e := fmt.Sprintf("unable to connect to cmd socket: %v\n", err)
		return "", errors.New(e)
	}
	defer c.Close()

	encoded, err := json.Marshal(cmd)
	if err != nil {
		e := fmt.Sprintf("unable to encode command: %v\n", err)
		return "", errors.New(e)
	}

	if _, err = c.Write(encoded); err != nil {
		e := fmt.Sprintf("unable to write to cmd socket: %v\n", err)
		return "", errors.New(e)
	}

	var res Response
	de := json.NewDecoder(c)

	if err = de.Decode(&res); err != nil {
		e := fmt.Sprintf("unable to decode response: %v\n", err)
		return "", errors.New(e)
	}

	if res.Status != 0 {
		e := fmt.Sprintf("received error response: %s\n", res.Content)
		return "", errors.New(e)
	}

	return res.Content, nil
}

// ExpectArgs checks for the specified args to be present, and display
// and error message and quit if there are too little or too many.
func ExpectArgs(actual []string, expected ...string) {
	if len(actual) > len(expected) {
		dokku.LogFail(fmt.Sprintf("Unexpected argument(s): %v", actual))
	}

	if len(actual) == 0 && len(expected) > 0 {
		args := []string{}
		for _, a := range expected {
			args = append(args, fmt.Sprintf("<%s>", a))
		}

		argsStr := strings.Join(args, " ")
		dokku.LogFail(fmt.Sprintf("Expected: %s", argsStr))
	}
}
