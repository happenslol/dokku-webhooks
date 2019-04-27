package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"

	dokku "github.com/dokku/dokku/plugins/common"
)

const (
	webhooksDir = "/var/lib/dokku/data/webhooks"
	cmdSocket   = "/var/lib/dokku/data/webhooks/cmd.sock"
)

// SendCmd sends a message to the command socket and return the response as
// a string which can be printed out as-is
func SendCmd(cmd Cmd) (string, error) {
	if !dokku.DirectoryExists(webhooksDir) || !dokku.FileExists(cmdSocket) {
		// TODO(happens): Tell user how to enable webhooks
		// NOTE(happens): The directory won't exist if webhooks haven't
		// ever been enabled, the cmdSocket won't exist if the server
		// is not currently running
		log.Fatalln("webhooks are not enabled!")
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
