package main

import (
	"net/http"

	"log"
	"os"

	"github.com/x12tech/go-websocketapi/apiserver"
)

type commandHandler struct {
	// db connections here
}

type pingRequest struct {
	Ping string `json:"ping"`
}

type pongResponce struct {
	Pong  string `json:"pong"`
	Times int    `json:"times"`
}

func (pongResponce) CmdName() string {
	return `pong`
}

type session struct {
	protoVersion int
	pingTime     int
}

func sessionFromConn(conn apiserver.Conn) *session {
	return conn.Session().(*session)
}

func (commandHandler) Echo(conn apiserver.Conn, request *pingRequest) (pongResponce, error) {
	sess := sessionFromConn(conn)
	sess.pingTime++
	return pongResponce{Pong: request.Ping, Times: sess.pingTime}, nil
}

func main() {
	handler := &commandHandler{}

	router := apiserver.NewRouter()
	router.RegisterApiHandler(0, `ping`, handler.Echo)

	srv, err := apiserver.NewServer(apiserver.ServerOpts{
		Router: router,
		NewSessionFn: func() interface{} {
			return new(session)
		},
		Logger: log.New(os.Stderr, "", log.LstdFlags),
	})
	if err != nil {
		panic(err)
	}
	log.Println(http.ListenAndServe(`:9091`, srv))
}
