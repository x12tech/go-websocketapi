package apiserver

import "encoding/json"

type ErrorCommandData struct {
	Type    string `json:"type"`
	Message string `json:"msg"`
}

func ApiError(typ, message string) []CommandOut {
	return []CommandOut{
		{
			Name: `error`,
			Data: &ErrorCommandData{
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
