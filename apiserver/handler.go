package apiserver

import (
	"encoding/json"
	"reflect"

	"github.com/pkg/errors"
)

type handlerFunc interface{}

type handler struct {
	Func       reflect.Value
	Input      reflect.Type
	InputPtr   bool
	Output     []handlerOut
	Middleware []MiddlewareFunc
}

type handlerOut struct {
	typ      reflect.Type
	isSlice  bool
	elemType reflect.Type
}

var connectionType = reflect.TypeOf((*Conn)(nil)).Elem()
var namerType = reflect.TypeOf((*CmdNamer)(nil)).Elem()

func NewHandler(f handlerFunc, middleware []MiddlewareFunc) *handler {
	funcValue := reflect.ValueOf(f)
	funcType := funcValue.Type()
	if funcType.Kind() != reflect.Func {
		panic(`argument must be function`)
	}
	if funcType.NumIn() != 1 && funcType.NumIn() != 2 {
		panic(funcType.String() + `must be 1 or 2 arguments`)
	}
	if !funcType.In(0).Implements(connectionType) {
		panic(`first argument must be Conn`)
	}
	if funcType.NumOut() < 1 {
		panic(`must return 1 output or more `)
	}
	h := new(handler)
	h.Func = funcValue
	h.Middleware = middleware
	if funcType.NumIn() == 2 {
		h.Input, h.InputPtr = ptrType(funcType.In(1))
	}
	for i := 0; i < funcType.NumOut()-1; i++ {
		var isSlice bool = false
		var elemType reflect.Type
		outType := funcType.Out(i)
		if !outType.Implements(namerType) && outType.Kind() != reflect.Slice {
			panic(outType.String() + ` must implement interface CmdNamer{ CmdName() string }`)
		} else if !outType.Implements(namerType) && outType.Kind() == reflect.Slice {
			elemOutType, _ := ptrType(outType.Elem())
			if !elemOutType.Implements(namerType) {
				panic(elemOutType.String() + ` must implement interface CmdNamer{ CmdName() string }`)
			}
			isSlice = true
			elemType = elemOutType
		}
		if outType.Kind() == reflect.Ptr {
			outType = outType.Elem()
		}
		h.Output = append(h.Output, handlerOut{outType, isSlice, elemType})
	}
	return h
}

func (self *handler) Call(conn Conn, data []byte) ([]CmdNamer, error) {
	if conn == nil {
		return nil, errors.New(`call with nil Conn`)
	}
	out := make([]CmdNamer, 0, 10)
	for _, mw := range self.Middleware {
		res, cont := mw(conn)
		out = append(out, res...)
		if !cont {
			return out, nil
		}
	}
	var inputValue reflect.Value
	if self.Input != nil {
		inputValue = reflect.New(self.Input)
		input := inputValue.Interface()
		err := json.Unmarshal(data, input)
		if err != nil {
			return nil, err
		}
		if !self.InputPtr {
			inputValue = inputValue.Elem()
		}
	}
	var output []reflect.Value
	if self.Input != nil {
		output = self.Func.Call([]reflect.Value{reflect.ValueOf(conn), inputValue})
	} else {
		output = self.Func.Call([]reflect.Value{reflect.ValueOf(conn)})
	}
	for i := 0; i < len(self.Output); i++ {
		if self.Output[i].isSlice {
			l := output[i].Len()
			for m := 0; m < l; m++ {
				out = append(out, output[i].Index(m).Interface().(CmdNamer))
			}
		} else {
			out = append(out, output[i].Interface().(CmdNamer))
		}
	}
	var retError error
	var outErrorValue = output[len(self.Output)]
	if !outErrorValue.IsNil() {
		retErrorIface := outErrorValue.Interface()
		if errCmd, ok := retErrorIface.(CmdNamer); ok {
			out = append(out, errCmd)
		} else {
			retError = retErrorIface.(error)
		}
	}

	return out, retError
}

func (self *handler) String() string {
	out := self.Input.String()
	if len(self.Output) > 0 {
		out += ` -> `
		for _, o := range self.Output {
			out += o.typ.String() + ` `
		}
	}
	return out
}

func ptrType(p reflect.Type) (typ reflect.Type, wasPtr bool) {
	if p.Kind() == reflect.Ptr {
		return p.Elem(), true
	}
	return p, false
}
