package apiserver

import (
	"golang.org/x/net/websocket"
	"net/http"
	"github.com/vvoronin/log"
)

type Server struct {
	router         *Router
	wsServer       *websocket.Server
	newSessionFunc func() interface{}
}

func NewServer(router *Router, newSessionFunc func() interface{}) *Server {
	self := &Server{
		router:         router,
		newSessionFunc: newSessionFunc,
	}
	self.wsServer = &websocket.Server{
		Handler: self.HandleWs,
	}
	return self
}

func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request)  {
	log.Println(`Handler`)
	self.wsServer.ServeHTTP(w,req)
}

func (self *Server) HandleWs(ws *websocket.Conn) {
	conn := NewConnection(ws, self.onInput, self.onConnectionClose, self.newSessionFunc())
	conn.Start()
}

func (self *Server) onConnectionClose(conn Conn) {

}

func (self *Server) onInput(conn Conn, buf []byte) {
	self.router.ProcessPacket(conn, buf)
}
