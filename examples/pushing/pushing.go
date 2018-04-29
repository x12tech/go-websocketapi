package main

import (
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/x12tech/go-websocketapi/apiserver"
)

type commandHandler struct {
	sessions map[string]*Session
}

type pingRequest struct {
	Ping string `json:"ping"`
}

type pongResponce struct {
	Pong string `json:"pong"`
}

type serverPush string

func (serverPush) CmdName() string {
	return `serverPushCommand`
}

func (pongResponce) CmdName() string {
	return `pong`
}

type Session struct {
	SessionKey string `json:"sessionKey"`
	conn       apiserver.Conn
	counter    int
	stopCh     chan struct{}
}

func (sess *Session) CmdName() string {
	return `Session`
}

func (sess *Session) Close() {
	log.Print(`disconnected`)
	sess.stopCh <- struct{}{}
}

func (sess *Session) Start(conn apiserver.Conn) {
	sess.conn = conn
	sess.stopCh = make(chan struct{}, 1)
	go sess.sendLoop()
}

func (sess *Session) sendLoop() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			sess.counter++
			sess.conn.Send(serverPush(`pushdata ` + strconv.Itoa(sess.counter)))
		case <-sess.stopCh:
			return
		}
	}
}

func sessionFromConn(conn apiserver.Conn) *Session {
	return conn.Session().(*Session)
}

func (commandHandler) Echo(conn apiserver.Conn, request *pingRequest) (pongResponce, error) {
	return pongResponce{Pong: request.Ping}, nil
}

type CreateSessionArgs struct {
	SessionKey string `json:"sessionKey"`
}

func (self *commandHandler) CreateSession(conn apiserver.Conn, args CreateSessionArgs) (*Session, error) {
	var (
		session *Session
		exist   bool
	)
	if session, exist = self.sessions[args.SessionKey]; !exist {
		session = &Session{
			SessionKey: RandStringRunes(10),
		}
		self.sessions[session.SessionKey] = session
	}
	session.conn.Close()
	conn.SetSession(session)
	session.Start(conn)
	return session, nil
}

func main() {
	handler := &commandHandler{
		sessions: make(map[string]*Session),
	}

	router := apiserver.NewRouter()
	router.RegisterApiHandler(0, `Ping`, handler.Echo)
	router.RegisterApiHandler(0, `CreateSession`, handler.CreateSession)

	srv, err := apiserver.NewServer(apiserver.ServerOpts{
		Router: router,
		Logger: log.New(os.Stderr, "", log.LstdFlags),
	})
	if err != nil {
		panic(err)
	}
	log.Println(http.ListenAndServe(`:9091`, srv))
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
