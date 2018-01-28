package apiserver

import (
	"net/http"

	"github.com/pkg/errors"
	"golang.org/x/net/websocket"
)

type Logger interface {
	Println(v ...interface{})
}

type EmptyLogger struct{}

func (EmptyLogger) Println(v ...interface{}) {}

type Server struct {
	router         *Router
	wsServer       *websocket.Server
	newSessionFunc func() interface{}
	log            Logger
}

type ServerOpts struct {
	Router       *Router
	NewSessionFn func() interface{}
	Logger       Logger
}

func NewServer(opts ServerOpts) (*Server, error) {
	if opts.Router == nil {
		return nil, errors.New(`router not defined`)
	}
	if opts.NewSessionFn == nil {
		opts.NewSessionFn = func() interface{} {
			return nil
		}
	}
	if opts.Logger == nil {
		opts.Logger = &EmptyLogger{}
	}
	self := &Server{
		router:         opts.Router,
		newSessionFunc: opts.NewSessionFn,
		log:            opts.Logger,
	}
	self.wsServer = &websocket.Server{
		Handler: self.HandleWs,
	}
	self.router.SetLogger(self.log)
	return self, nil
}

func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	self.log.Println(`handle connect`)
	self.wsServer.ServeHTTP(w, req)
}

func (self *Server) HandleWs(ws *websocket.Conn) {
	conn := &Connection{
		ws:      ws,
		onInput: self.router.ProcessPacket,
		onClose: self.onConnectionClose,
		log:     self.log,
		sess:    self.newSessionFunc(),
	}
	conn.Start()
}

func (self *Server) onConnectionClose(conn Conn) {

}
