package common

// CmdType defines which command will be executed
type CmdType int

const (
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
