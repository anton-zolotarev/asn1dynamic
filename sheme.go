package asn1dynamic

import (
	"container/list"
	"fmt"
	"io"

	"github.com/anton-zolotarev/go-simplejson"
)

type Sheme struct {
	name string
	obj  *simplejson.Json
}

func check(obj *simplejson.Json, name string) error {
	mp, _ := obj.Map()
	for k, v := range mp {
		fld := simplejson.Wrap(v)
		if k == "$type" {
			tp := fld.MustString()
			if tp == "CHOICE" || tp == "SEQUENCE" || tp == "ANY" {
				_, f1 := mp["$field"]
				_, f2 := mp["$of"]
				if !f1 && !f2 {
					return fmt.Errorf("'%s' (%s) miss '$field' or '$of'", name, tp)
				}
			}
		}
		if k == "$field" {
			var tags map[int]bool
			var ids map[int]bool
			fldmp, _ := fld.Map()
			for k := range fldmp {
				if tn, err := fld.GetPath(k, "$tag").Int(); err == nil {
					if len(tags) == 0 {
						tags = make(map[int]bool)
					}
					// if _, f := tags[tn]; f {
					// 	return fmt.Errorf("$tag '%d' in '%s' field already exists", tn, k)
					// }
					tags[tn] = true
				}
			}
			for k := range fldmp {
				if tn, err := fld.GetPath(k, "$id").Int(); err == nil {
					if len(ids) == 0 {
						ids = make(map[int]bool)
					}
					if _, f := ids[tn]; f {
						return fmt.Errorf("$id '%d' in '%s' field already exists", tn, k)
					}
					if _, f := tags[tn]; !f && !implicit {
						fld.Get(k).Set("$implicit", true)
					}
					ids[tn] = true
				} else if km, err := fld.Get(k).Map(); err == nil {
					_, f1 := km["$field"]
					_, f2 := km["$of"]
					if !f1 && !f2 {
						return fmt.Errorf("'%s' miss '$id' field", k)
					}
				}
			}
		}
		if err := check(fld, k); err != nil {
			return err
		}
	}
	return nil
}

func (s *Sheme) init() error {
	obj, err := s.obj.Compile()
	if err != nil {
		return err
	}
	if err = check(obj, ""); err != nil {
		return err
	}

	s.obj = obj
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

func (s *Sheme) ID() int {
	tp, _ := s.obj.Get("$id").Int()
	return tp
}

func (s *Sheme) Index() int {
	if tp, err := s.obj.Get("$tag").Int(); err == nil {
		return tp
	}
	return s.ID()
}

func (s *Sheme) Optional() bool {
	tp, _ := s.obj.Get("$optional").Bool()
	return tp
}

func (s *Sheme) Implicit() bool {
	tp, _ := s.obj.Get("$implicit").Bool()
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
			return Wrap(itm, name)
		}
	}
	return nil
}

func (s *Sheme) Of() *Sheme {
	if fld := s.OfAttr(); fld != nil {
		return Wrap(fld, s.name)
	}
	return nil
}

type fieldList struct {
	cur *list.Element
	lst *list.List
}

func (fl *fieldList) Len() int {
	return fl.lst.Len()
}

func (fl *fieldList) Begin() *Sheme {
	if fl.cur = fl.lst.Front(); fl.cur != nil {
		return fl.cur.Value.(*Sheme)
	}
	return nil
}

func (fl *fieldList) Next() *Sheme {
	if fl.cur = fl.cur.Next(); fl.cur != nil {
		return fl.cur.Value.(*Sheme)
	}
	return nil
}

func (fl *fieldList) FindIndex(idx int) *Sheme {
	for el := fl.Begin(); el != nil; el = fl.Next() {
		debugPrint("FindIndex %d == %d", idx, el.Index())
		if idx == el.Index() {
			return el
		}
	}
	return nil
}

func (fl *fieldList) FindID(idx int) *Sheme {
	for el := fl.Begin(); el != nil; el = fl.Next() {
		if idx == el.ID() {
			return el
		}
	}
	return nil
}

func (fl *fieldList) Add(sh *Sheme) {
	var fnd *list.Element
	id := sh.ID()
	for el := fl.lst.Front(); el != nil; el = el.Next() {
		elsh := el.Value.(*Sheme)
		if id > elsh.ID() {
			fnd = el
		}
	}
	if fnd != nil {
		fl.lst.InsertAfter(sh, fnd)
	} else {
		fl.lst.PushFront(sh)
	}
}

func (s *Sheme) FieldList() fieldList {
	fld := s.FieldAttr()
	ret := fieldList{lst: list.New()}

	for k, v := range fld {
		if obj, ok := v.(map[string]interface{}); ok {
			ret.Add(Wrap(obj, k))
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

func Wrap(itm map[string]interface{}, name ...string) *Sheme {
	out := &Sheme{obj: simplejson.Wrap(itm)}
	if len(name) > 0 {
		out.name = name[0]
	}
	return out
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
