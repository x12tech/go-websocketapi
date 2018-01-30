package apiserver

import "encoding/json"

type ErrorCommand struct {
	Type    string `json:"type"`
	Message string `json:"msg"`
}

func (ErrorCommand) CmdName() string {
	return `Error`
}

func (e ErrorCommand) Error() string {
	return e.Type + `: ` + e.Message
}

func ApiError(typ, message string) *ErrorCommand {
	return &ErrorCommand{
		Type:    typ,
		Message: message,
	}
}

func apiErrorCommands(typ, message string) []CommandOut {
	return []CommandOut{
		{
			Name: `Error`,
			Data: &ErrorCommand{
				Type:    typ,
				Message: message,
			},
		},
	}
}

type CommandIn struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data,omitempty"`
}

type PacketIn struct {
	Commands []CommandIn `json:"cmds"`
	Cid      int32       `json:"cid,omitempty"`
}

type CommandOut struct {
	Name string      `json:"name"`
	Data interface{} `json:"data,omitempty"`
}

type PacketOut struct {
	Commands []CommandOut `json:"cmds"`
	Cid      int32        `json:"cid,omitempty"`
}
