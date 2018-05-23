package apiserver

import (
	"reflect"
	"strings"
)

type Describer struct {
	typeMap      map[reflect.Type]string
	visitedTypes map[reflect.Type]struct{}
}

func NewDescriber(tm map[reflect.Type]string) *Describer {
	return &Describer{
		typeMap: tm,
	}
}

func (self *Describer) Describe(t reflect.Type) interface{} {
	self.visitedTypes = make(map[reflect.Type]struct{})
	return self.describeType(t)
}

func (self *Describer) visited(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	if _, ok := self.visitedTypes[t]; ok {
		return true
	}
	self.visitedTypes[t] = struct{}{}
	return false
}

func (self *Describer) unvisited(t reflect.Type) {
	delete(self.visitedTypes, t)
}

func (self *Describer) describeStruct(t reflect.Type) map[string]interface{} {
	descr := make(map[string]interface{})
	nFields := t.NumField()

	for i := 0; i < nFields; i++ {
		f := t.Field(i)
		name, opt := getFiledName(f)
		if name == `` || name == `-` {
			continue
		}
		typ := f.Type
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		if opt {
			name = name + `?`
		}
		descr[name] = self.describeType(typ)
	}
	return descr
}

func (self *Describer) describeType(t reflect.Type) interface{} {
	if t == nil {
		return nil
	}
	if t.Kind() == reflect.Ptr {
		return self.describeType(t.Elem())
	}
	if self.visited(t) {
		return `...`
	}
	defer self.unvisited(t)
	if descr, ok := self.typeMap[t]; ok {
		return descr
	}
	if t.Kind() == reflect.Struct {
		return self.describeStruct(t)
	}
	if t.Kind() == reflect.Slice {
		return []interface{}{
			self.describeType(t.Elem()),
		}
	}
	if t.Kind() == reflect.Map {
		return map[string]interface{}{
			`MAP[ ` + t.Key().String() + ` ]`: self.describeType(t.Elem()),
		}
	}
	return t.String()
}

func getFiledName(fld reflect.StructField) (string, bool) {
	if fld.PkgPath != `` {
		return ``, false
	}
	js := strings.Split(fld.Tag.Get(`json`), `,`)
	if len(js) > 0 && js[0] != `` {
		if len(js) > 1 && js[1] == `omitempty` {
			return js[0], true
		}
		return js[0], false
	}
	return fld.Name, false
}
