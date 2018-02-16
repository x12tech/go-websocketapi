package apiserver

import (
	"encoding/json"
	"reflect"
	"sort"
)

type CmdNamer interface {
	CmdName() string
}

type handlerValue struct {
	Version int
	Handler *handler
}

type handlerValues []handlerValue

func (self handlerValues) Len() int           { return len(self) }
func (self handlerValues) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self handlerValues) Less(i, j int) bool { return self[i].Version > self[j].Version }

type IRouter interface {
	RegisterApiHandler(version int, command string, handler handlerFunc)
}

type Router struct {
	commandHandlers map[string]*handlerValues
	getVersion      func(conn Conn) int
	log             Logger
}

func NewRouter() *Router {
	return &Router{
		commandHandlers: make(map[string]*handlerValues),
		getVersion:      func(conn Conn) int { return 0 },
	}
}

func (self *Router) SetLogger(l Logger) {
	self.log = l
}

// handlerFunc Must be func(*Conn,*SomeType) *SomeRetType,error
// Or func(*Conn,*SomeType) *SomeRetType,*SomeOtherRetType,error
// Or func(*Conn,*SomeType) []interface{},error
// Or func(*Conn,*SomeType) error
func (self *Router) RegisterApiHandler(version int, command string, handler handlerFunc) {
	handlers := self.commandHandlers[command]
	if handlers == nil {
		handlers = new(handlerValues)
		self.commandHandlers[command] = handlers
	}
	*handlers = append(*handlers, handlerValue{version, NewHandler(handler, nil)})
	sort.Sort(*handlers)
}

type MiddlewareFunc func(conn Conn) (commands []CmdNamer, next bool)

func (self *Router) RegisterApiHandlerWithMiddleware(version int, command string, handler handlerFunc, middle []MiddlewareFunc) {
	handlers := self.commandHandlers[command]
	if handlers == nil {
		handlers = new(handlerValues)
		self.commandHandlers[command] = handlers
	}
	*handlers = append(*handlers, handlerValue{version, NewHandler(handler, middle)})
	sort.Sort(*handlers)
}

func (self *Router) RegisterGetVersion(cb func(conn Conn) int) {
	self.getVersion = cb
}

func (self *Router) With(mw MiddlewareFunc) *middlewareWrapper {
	return &middlewareWrapper{
		funcs:  []MiddlewareFunc{mw},
		router: self,
	}
}

type middlewareWrapper struct {
	funcs  []MiddlewareFunc
	router *Router
}

func (self *middlewareWrapper) With(mw MiddlewareFunc) *middlewareWrapper {
	return &middlewareWrapper{
		funcs:  append(self.funcs, mw),
		router: self.router,
	}
}

func (self *middlewareWrapper) RegisterApiHandler(version int, command string, handler handlerFunc) {
	self.router.RegisterApiHandlerWithMiddleware(version, command, handler, self.funcs)
}

func (self *Router) ProcessCommand(conn Conn, version int, command string, data json.RawMessage) (res []CommandOut) {
	if handlers, ok := self.commandHandlers[command]; ok {
		found := false
		for _, handler := range *handlers {
			if handler.Version <= self.getVersion(conn) {
				cmds, err := handler.Handler.Call(conn, data)
				res = make([]CommandOut, 0, len(cmds))
				if err != nil {
					res = apiErrorCommands(`exec_error`, err.Error())
				} else {
					for _, cmd := range cmds {
						if cmd != nil {
							res = append(res, CommandOut{
								Name: cmd.CmdName(),
								Data: cmd,
							})
						}
					}
				}
				found = true
				break
			}
		}
		if !found {
			res = apiErrorCommands(`command_handler_not_found`, `command_handler_not_found version`)
		}
	} else {
		res = apiErrorCommands(`command_handler_not_found`, `command_handler_not_found at all`)
	}
	return
}

func (self *Router) ProcessPacket(conn Conn, packetBuf []byte) {
	var packet *PacketIn
	err := json.Unmarshal(packetBuf, &packet)
	if err != nil {
		errBuf, _ := json.Marshal(&PacketOut{
			Commands: apiErrorCommands("cannot parse command", err.Error()),
		})
		conn.Send(errBuf)
		return
	}
	out := &PacketOut{
		Cid: packet.Cid,
	}
	for _, cmd := range packet.Commands {
		res := self.ProcessCommand(conn, 0, cmd.Name, cmd.Data)
		out.Commands = append(out.Commands, res...)
	}
	ret, err := json.Marshal(out)
	if err != nil {
		panic(err)
	} else {
		conn.Send(ret)
	}
}

func MarshallCommands(cmds ...CmdNamer) []byte {
	packet := PacketOut{
		Commands: make([]CommandOut, 0, len(cmds)),
	}
	for _, cmd := range cmds {
		packet.Commands = append(packet.Commands, CommandOut{Name: cmd.CmdName(), Data: cmd})
	}
	buf, err := json.Marshal(packet)
	if err != nil {
		errPacket := PacketOut{
			Commands: apiErrorCommands(`internal_error`, err.Error()),
		}
		buf, _ = json.Marshal(errPacket)
	}
	return buf
}

type ServerCommandDesciption struct {
	Name           string
	ReplayCommands []string
	Params         interface{}
}

type ClientCommandDesciption struct {
	Name   string
	Params interface{}
}

func (self *Router) DescribeApi(tm map[reflect.Type]string) (scmds []*ServerCommandDesciption, ccmds []*ClientCommandDesciption) {
	serverCommands := make(map[string]*ServerCommandDesciption)
	clientTypes := make(map[reflect.Type]struct{})
	describer := NewDescriber(tm)
	for name, holder := range self.commandHandlers {
		handler := (*holder)[0].Handler
		replay := make([]string, 0)
		for _, out := range handler.Output {
			if out.isSlice {
				clientTypes[out.elemType] = struct{}{}
				replay = append(replay, reflect.New(out.elemType).Interface().(CmdNamer).CmdName())
			} else {
				clientTypes[out.typ] = struct{}{}
				replay = append(replay, reflect.New(out.typ).Interface().(CmdNamer).CmdName())
			}
		}
		serverCommands[name] = &ServerCommandDesciption{
			Name:           name,
			ReplayCommands: replay,
			Params:         describer.Describe(handler.Input),
		}
	}
	serverCommandsSlice := make([]string, 0, len(serverCommands))
	for k := range serverCommands {
		serverCommandsSlice = append(serverCommandsSlice, k)
	}

	sort.Strings(serverCommandsSlice)
	for _, k := range serverCommandsSlice {
		scmds = append(scmds, serverCommands[k])
	}

	clientCommands := make(map[string]*ClientCommandDesciption)
	for t := range clientTypes {
		name := reflect.New(t).Interface().(CmdNamer).CmdName()
		clientCommands[name] = &ClientCommandDesciption{
			Name:   name,
			Params: describer.Describe(t),
		}
	}

	clientCommandsSlice := make([]string, 0, len(clientCommands))
	for k := range clientCommands {
		clientCommandsSlice = append(clientCommandsSlice, k)
	}

	sort.Strings(clientCommandsSlice)
	for _, k := range clientCommandsSlice {
		ccmds = append(ccmds, clientCommands[k])
	}

	return
}
