package apiserver_test

import (
	"net/http"

	"context"

	"strconv"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/x12tech/go-websocketapi/apiserver"
)

type ActiveSession struct {
	conn    apiserver.Conn
	onClose func()
}

type StillAlive struct {
	Ping string `json:"ping"`
}

func (StillAlive) CmdName() string {
	return `StillAlive`
}

func (self *ActiveSession) Start(c apiserver.Conn) {
	self.conn = c
	time.AfterFunc(time.Millisecond, func() {
		self.conn.Send(StillAlive{Ping: `test`})
	})
	time.AfterFunc(time.Millisecond*3, func() {
		self.conn.Send(StillAlive{Ping: `test 2`})
	})
}

func (self *ActiveSession) Close() {
	self.onClose()
}

var _ = Describe("main", func() {
	var (
		router     *apiserver.Router
		server     *apiserver.Server
		httpserver *http.Server
		err        error
		port       int
	)
	BeforeEach(func() {
		router = apiserver.NewRouter()
		server, err = apiserver.NewServer(apiserver.ServerOpts{
			Router: router,
			NewSessionFn: func() interface{} {
				return new(ActiveSession)
			},
		})
		Expect(err).To(Succeed())
		listener, p, err := ListenSomeTcpPort()
		port = p
		Expect(err).To(Succeed())
		Expect(listener).ToNot(BeNil())

		httpserver = &http.Server{
			Handler: server,
		}
		go httpserver.Serve(listener)
	})
	AfterEach(func() {
		httpserver.Shutdown(context.Background())
	})
	var Connect = func() *ApiClient {
		conn, err := Dial(`127.0.0.1:` + strconv.Itoa(port))
		Expect(err).To(Succeed())
		return conn
	}
	It(`works with server pushes`, func(done Done) {
		router.RegisterApiHandler(0, `start_pushes`, func(conn apiserver.Conn) error {
			sess := conn.Session().(*ActiveSession)
			sess.onClose = func() {
				done <- 1
			}
			sess.Start(conn)
			return nil
		})
		c := Connect()
		c.Send([]byte(`
			{ "cid": 123, "cmds":[{  "name" : "start_pushes" }]}
		`))
		Expect(c.Await()).To(MatchJSON(`{ "cid" : 123, "cmds": null}`))
		Expect(c.Await()).To(MatchJSON(`{ "cmds": [{  "name" : "StillAlive", "data":{"ping":"test"} }]}`))
		Expect(c.Await()).To(MatchJSON(`{ "cmds": [{  "name" : "StillAlive", "data":{"ping":"test 2"} }]}`))
		Expect(c.ws.Close()).To(Succeed())
	})
})
