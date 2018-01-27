package apiserver_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/x12tech/go-websocketapi/apiserver"
)

type testIn struct {
	Name string `json:"name"`
}

type testOut struct {
	Name string `json:"name"`
}

func (self *testOut) CmdName() string { return `testOut` }

type testOut2 struct {
	Name string `json:"name"`
}

func (self *testOut2) CmdName() string { return `testOut2` }

var _ = Describe("main", func() {
	It(`compiles`, func() {
		f := func(conn apiserver.Conn, arg *testIn) error { return nil }
		h := apiserver.NewHandler(f)
		Expect(h).ToNot(BeNil())
	})
	It(`compiles2`, func() {
		f := func(conn apiserver.Conn, arg *testIn) (*testOut, error) { return nil, nil }
		h := apiserver.NewHandler(f)
		Expect(h).ToNot(BeNil())
	})
	It(`works`, func() {
		f := func(conn apiserver.Conn, arg *testIn) (*testOut, error) {
			return &testOut{
				Name: arg.Name,
			}, nil
		}
		h := apiserver.NewHandler(f)
		Expect(h).ToNot(BeNil())
		cmds, err := h.Call(apiserver.NewFakeConn(), []byte(`{ "Name": "some" }`))
		Expect(err).To(Succeed())
		Expect(len(cmds)).To(Equal(1))
		Expect(cmds[0]).To(Equal(&testOut{Name: `some`}))
	})
	It(`handles error`, func() {
		f := func(conn apiserver.Conn, arg *testIn) (*testOut, error) {
			return nil, errors.New(`some error`)
		}
		h := apiserver.NewHandler(f)
		Expect(h).ToNot(BeNil())
		_, err := h.Call(apiserver.NewFakeConn(), []byte(`{ "Name": "some" }`))
		Expect(err).ToNot(Succeed())
	})
	It(`works`, func() {
		f := func(conn apiserver.Conn, arg *testIn) (*testOut, *testOut2, error) {
			return &testOut{
					Name: arg.Name,
				}, &testOut2{
					Name: arg.Name,
				}, nil
		}
		h := apiserver.NewHandler(f)
		Expect(h).ToNot(BeNil())
		Expect(h.String()).To(Equal(`apiserver_test.testIn -> apiserver_test.testOut apiserver_test.testOut2 `))
		cmds, err := h.Call(apiserver.NewFakeConn(), []byte(`{ "Name": "some" }`))
		Expect(err).To(Succeed())
		Expect(len(cmds)).To(Equal(2))
		Expect(cmds[0]).To(Equal(&testOut{Name: `some`}))
		Expect(cmds[1]).To(Equal(&testOut2{Name: `some`}))
	})

})
