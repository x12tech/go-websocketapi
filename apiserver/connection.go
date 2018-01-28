package apiserver

import (
	"io"

	"golang.org/x/net/websocket"
)

type Conn interface {
	Send([]byte) error
	Session() interface{}
	Close()
}

type onInputFunc func(Conn, []byte)

type Connection struct {
	sess    interface{}
	onInput onInputFunc
	onClose func(Conn)
	ws      *websocket.Conn
	log     Logger
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
		self.onInput(self, buf[0:n])
	}
}

func (self *Connection) Send(buf []byte) error {
	_, err := self.ws.Write(buf)
	if err != nil {
		self.Close()
	}
	return err
}

func (self *Connection) Session() interface{} {
	return self.sess
}

type Closer interface {
	Close()
}

func (self *Connection) Close() {
	if sessionCloser, ok := self.sess.(Closer); ok {
		sessionCloser.Close()
	}
	self.onClose(self)
	self.ws.Close()
}

type FakeConn struct {
	SessionValue interface{}
	Written      [][]byte
}

func NewFakeConn() *FakeConn {
	return &FakeConn{}
}

func (self *FakeConn) Send(buf []byte) error {
	self.Written = append(self.Written, buf)
	return nil
}

func (self *FakeConn) Session() interface{} {
	return self.SessionValue
}

func (*FakeConn) Close() {
}
