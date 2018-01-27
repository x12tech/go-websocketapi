package apiserver_test

import (
	"io"
	"net"

	"net/http"

	"context"

	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/x12tech/go-websocketapi/apiserver"
	"golang.org/x/net/websocket"
)

type testEchoRequest struct {
	Ping string `json:"ping"`
}

type testEchoResponce struct {
	Pong string `json:"pong"`
}

func (testEchoResponce) CmdName() string {
	return `test_echo_responce`
}

var _ = Describe("main", func() {
	var (
		router     *apiserver.Router
		server     *apiserver.Server
		httpserver *http.Server
		port       int
	)
	BeforeEach(func() {
		router = apiserver.NewRouter()
		server = apiserver.NewServer(router, func() interface{} {
			return ``
		})
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
	It(`works with no arguments`, func() {
		router.RegisterApiHandler(0, `cmdname`, func(conn apiserver.Conn) error {
			return nil
		})
		c := Connect()
		c.Send([]byte(`
			{ "cid": 123, "cmds":[{  "name" : "cmdname" }]}
		`))
		Expect(c.Await()).To(MatchJSON(`{ "cid" : 123, "cmds": null}`))
	})
	It(`works with ptr argument`, func() {
		router.RegisterApiHandler(0, `cmdname`, func(conn apiserver.Conn, req *testEchoRequest) error {
			Expect(req.Ping).To(Equal(`test`))
			return nil
		})
		c := Connect()
		c.Send([]byte(`
			{ "cid": 123, "cmds":[{  "name" : "cmdname", "data" : {"ping" : "test"} }]}
		`))
		Expect(c.Await()).To(MatchJSON(`{ "cid" : 123, "cmds": null}`))
	})
	It(`works with value argument`, func() {
		router.RegisterApiHandler(0, `cmdname`, func(conn apiserver.Conn, req testEchoRequest) error {
			Expect(req.Ping).To(Equal(`test`))
			return nil
		})
		c := Connect()
		c.Send([]byte(`
			{ "cid": 123, "cmds":[{  "name" : "cmdname", "data" : {"ping" : "test"} }]}
		`))
		Expect(c.Await()).To(MatchJSON(`{ "cid" : 123, "cmds": null}`))
	})
	It(`works with value return`, func() {
		router.RegisterApiHandler(0, `cmdname`, func(conn apiserver.Conn) (testEchoResponce, error) {
			return testEchoResponce{Pong: `test`}, nil
		})
		c := Connect()
		c.Send([]byte(`
			{ "cid": 123, "cmds":[{  "name" : "cmdname" }]}
		`))
		Expect(c.Await()).To(MatchJSON(
			`{ "cid" : 123, "cmds": [{ "name": "test_echo_responce", "data" : { "pong" : "test" }  }]}
		`))
	})
	It(`works with ptr return`, func() {
		router.RegisterApiHandler(0, `cmdname`, func(conn apiserver.Conn) (*testEchoResponce, error) {
			return &testEchoResponce{Pong: `test`}, nil
		})
		c := Connect()
		c.Send([]byte(`
			{ "cid": 123, "cmds":[{  "name" : "cmdname" }]}
		`))
		Expect(c.Await()).To(MatchJSON(
			`{ "cid" : 123, "cmds": [{ "name": "test_echo_responce", "data" : { "pong" : "test" }  }]}
		`))
	})
	It(`works with slice ptr return`, func() {
		router.RegisterApiHandler(0, `cmdname`, func(conn apiserver.Conn) ([]*testEchoResponce, error) {
			return []*testEchoResponce{
				{Pong: `test1`},
				{Pong: `test2`},
			}, nil
		})
		c := Connect()
		c.Send([]byte(`
			{ "cid": 123, "cmds":[{  "name" : "cmdname" }]}
		`))
		Expect(c.Await()).To(MatchJSON(
			`{ "cid" : 123, "cmds": [
					{ "name": "test_echo_responce", "data" : { "pong" : "test1" }  },
					{ "name": "test_echo_responce", "data" : { "pong" : "test2" }  }
					]}
		`))
	})
	It(`works with slice value return`, func() {
		router.RegisterApiHandler(0, `cmdname`, func(conn apiserver.Conn) ([]testEchoResponce, error) {
			return []testEchoResponce{
				{Pong: `test1`},
				{Pong: `test2`},
			}, nil
		})
		c := Connect()
		c.Send([]byte(`
			{ "cid": 123, "cmds":[{  "name" : "cmdname" }]}
		`))
		Expect(c.Await()).To(MatchJSON(
			`{ "cid" : 123, "cmds": [
					{ "name": "test_echo_responce", "data" : { "pong" : "test1" }  },
					{ "name": "test_echo_responce", "data" : { "pong" : "test2" }  }
					]}
		`))
	})
})

type ApiClient struct {
	ws *websocket.Conn
}

func Dial(addr string) (*ApiClient, error) {
	ws, err := websocket.Dial(`ws://`+addr+`/`, ``, `http://127.0.0.1/`)
	if err != nil {
		return nil, err
	}
	return &ApiClient{ws}, nil
}

func Upgrade(url, rwc io.ReadWriteCloser) (*ApiClient, error) {
	conf, err := websocket.NewConfig(`/`, ``)
	if err != nil {
		return nil, err
	}
	ws, err := websocket.NewClient(conf, rwc)
	if err != nil {
		return nil, err
	}
	return &ApiClient{ws}, nil
}

func (cli *ApiClient) Await() ([]byte, error) {
	buf := make([]byte, 65536)
	n, err := cli.ws.Read(buf)
	return buf[:n], err
}

func (cli *ApiClient) Send(buf []byte) error {
	_, err := cli.ws.Write(buf)
	return err
}

func ListenSomeTcpPort() (list net.Listener, port int, err error) {
	listener, err := net.Listen(`tcp`, `127.0.0.1:`)
	if err != nil {
		return nil, 0, err
	}
	tcpl, ok := listener.(*net.TCPListener)
	if !ok {
		return nil, 0, errors.New(`tcpl is not *net.TCPListener`)
	}
	port = tcpl.Addr().(*net.TCPAddr).Port
	return listener, port, nil
}
