package asn1dynamic

import (
	"io"

	"github.com/anton-zolotarev/go-simplejson"
)

type Sheme struct {
	name string
	obj  *simplejson.Json
}

func (s *Sheme) init() error {
	if obj, err := s.obj.Compile(); err != nil {
		return err
	} else {
		s.obj = obj
	}

	return nil
}

func (s *Sheme) String() string {
	b, _ := s.obj.MarshalJSON()
	return string(b)
}

func (s *Sheme) Name() string {
	return s.name
}

func (s *Sheme) Class(class string) *Sheme {
	obj := s.obj.GetPath(class)
	if obj.Empty() {
		return nil
	}
	return &Sheme{obj: obj, name: class}
}

func (s *Sheme) Type() string {
	tp, _ := s.obj.Get("$type").String()
	return tp
}

func (s *Sheme) Id() int {
	tp, _ := s.obj.Get("$id").Int()
	return tp
}

func (s *Sheme) Index() int {
	if tp, err := s.obj.Get("$tag").Int(); err == nil {
		return tp
	}
	return s.Id()
}

func (s *Sheme) Optional() bool {
	tp, _ := s.obj.Get("$optional").Bool()
	return tp
}

func (s *Sheme) Tagged() bool {
	_, tp := s.obj.CheckGet("$tag")
	return tp
}

func (s *Sheme) DefAttr() interface{} {
	return s.obj.Get("$default").Interface()
}

func (s *Sheme) MinAttr() int {
	tp, _ := s.obj.Get("$min").Int()
	return tp
}

func (s *Sheme) MaxAttr() int {
	tp, _ := s.obj.Get("$max").Int()
	return tp
}

func (s *Sheme) FormatAttr() string {
	tp, _ := s.obj.Get("$format").String()
	return tp
}

func (s *Sheme) FieldAttr() map[string]interface{} {
	tp, _ := s.obj.Get("$field").Map()
	return tp
}

func (s *Sheme) OfAttr() map[string]interface{} {
	tp, _ := s.obj.Get("$of").Map()
	return tp
}

func (s *Sheme) Field(name string) *Sheme {
	if fld := s.FieldAttr(); fld != nil {
		if itm, ok := fld[name].(map[string]interface{}); ok {
			sh := Wrap(itm)
			sh.name = name
			return sh
		}
	}
	return nil
}

func (s *Sheme) Of() *Sheme {
	if fld := s.OfAttr(); fld != nil {
		return Wrap(fld)
	}
	return nil
}

func (s *Sheme) SeqItems() []*Sheme {
	fld := s.FieldAttr()
	ret := make([]*Sheme, len(fld))

	for k, v := range fld {
		if obj, ok := v.(map[string]interface{}); ok {
			sh := Wrap(obj)
			sh.name = k
			if id := sh.Id(); id < len(ret) {
				ret[id] = sh
			}
		}
	}
	return ret
}

func (s *Sheme) EnumItems() map[int]string {
	fld := s.FieldAttr()
	ret := make(map[int]string)
	for k, v := range fld {
		if id, err := simplejson.Wrap(v).Int(); err == nil {
			ret[id] = k
		}
	}
	return ret
}

func Wrap(itm map[string]interface{}) *Sheme {
	return &Sheme{obj: simplejson.Wrap(itm)}
}

func NewSheme(data []byte) (*Sheme, error) {
	obj, err := simplejson.NewJson(data)
	if err != nil {
		return nil, err
	}

	sh := Sheme{obj: obj}
	if err = sh.init(); err != nil {
		return nil, err
	}
	return &sh, nil
}

func NewShemeReader(rd io.Reader) (*Sheme, error) {
	obj, err := simplejson.NewFromReader(rd)
	if err != nil {
		return nil, err
	}

	sh := Sheme{obj: obj}
	if err = sh.init(); err != nil {
		return nil, err
	}
	return &sh, nil
}
