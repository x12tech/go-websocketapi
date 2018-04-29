package apiserver

import (
	"io"
	"sync"

	"golang.org/x/net/websocket"
)

type Conn interface {
	// public method for pushes
	Send(cmds ...CmdNamer) error
	// private method for write responces
	send([]byte) error
	Session() interface{}
	SetSession(v interface{})
	Close()
}

type onInputFunc func(Conn, []byte)

type Connection struct {
	sess      interface{}
	onInput   onInputFunc
	onClose   func(Conn)
	ws        *websocket.Conn
	log       Logger
	cmdLogger CmdLogger
}

func (self *Connection) Start() {
	buf := make([]byte, 8000)
	for {
		n, err := self.ws.Read(buf)
		if err != nil && err != io.EOF {
			self.Close()
			break
		}
		if err == io.EOF {
			self.Close()
			break
		}
		self.log.Println(`IN:`, string(buf[0:n]))
		self.onInput(self, buf[0:n])
	}
}

func (self *Connection) send(buf []byte) error {
	self.log.Println(`OUT:`, string(buf))
	_, err := self.ws.Write(buf)
	if err != nil {
		self.Close()
	}
	return err
}
func (self *Connection) Send(cmds ...CmdNamer) error {
	packet := PacketOut{
		Commands: make([]CommandOut, 0, len(cmds)),
	}
	for _, cmd := range cmds {
		packet.Commands = append(packet.Commands, CommandOut{Name: cmd.CmdName(), Data: cmd})
	}
	if self.cmdLogger != nil {
		self.cmdLogger.LogPush(self.Session(), &packet)
	}
	buf := marshallPacket(packet)
	return self.send(buf)
}

func (self *Connection) SetSession(v interface{}) {
	self.sess = v
}

func (self *Connection) Session() interface{} {
	return self.sess
}

type Closer interface {
	Close()
}

func (self *Connection) Close() {
	self.log.Println(`Close()`)
	self.log.Println(self.sess)
	if sessionCloser, ok := self.sess.(Closer); ok {
		sessionCloser.Close()
	}
	self.onClose(self)
	self.ws.Close()
}

type FakeConn struct {
	SessionValue interface{}
	Written      [][]byte
	Mu           sync.Mutex
}

func NewFakeConn() *FakeConn {
	return &FakeConn{}
}

func (self *FakeConn) send(buf []byte) error {
	self.Mu.Lock()
	defer self.Mu.Unlock()
	self.Written = append(self.Written, buf)
	return nil
}

func (self *FakeConn) Send(cmds ...CmdNamer) error {
	packet := PacketOut{
		Commands: make([]CommandOut, 0, len(cmds)),
	}
	for _, cmd := range cmds {
		packet.Commands = append(packet.Commands, CommandOut{Name: cmd.CmdName(), Data: cmd})
	}
	buf := marshallPacket(packet)
	return self.send(buf)
}

func (self *FakeConn) Session() interface{} {
	return self.SessionValue
}

func (self *FakeConn) SetSession(v interface{}) {
	self.SessionValue = v
}

func (*FakeConn) Close() {
}
