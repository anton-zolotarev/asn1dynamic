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

func check(sh *Sheme, name string) error {
	tp := sh.Type()
	of := sh.OfAttr()
	fl := sh.FieldAttr()
	switch tp {
	case "":
		return fmt.Errorf("miss '$type' in '%s'", name)
	case "CHOICE", "SEQUENCE", "ANY":
		if of == nil && fl == nil {
			return fmt.Errorf("miss '$field' or '$of' in '%s' (%s)", name, tp)
		}
	default:
		of = nil
		fl = nil
	}

	if of != nil {
		return check(Wrap(of), name)
	}

	if fl != nil {
		var ids map[int]bool
		var tgs map[int]bool
		fld := NewFieldList(fl)
		if fld.Len() == 0 {
			if tp != "ANY" {
				return fmt.Errorf("cannot find any $field in '%s' (%s)", name, tp)
			}
		} else {
			for sh := fld.Begin(); sh != nil; sh = fld.Next() {
				if tp != "ANY" {
					if len(ids) == 0 {
						ids = make(map[int]bool)
					}
					id, err := sh.obj.Get("$id").Int()
					if err != nil {
						return fmt.Errorf("miss '$id' in '%s' field of '%s' (%s)", sh.Name(), name, tp)
					}
					if _, f := ids[id]; f {
						return fmt.Errorf("duplicate $id '%d' in '%s' field of '%s' (%s)", id, sh.Name(), name, tp)
					}
					ids[id] = true
				}
				if tp == "CHOICE" {
					if len(tgs) == 0 {
						tgs = make(map[int]bool)
					}
					id := sh.Index()
					if _, f := tgs[id]; f {
						return fmt.Errorf("duplicate $tag '%d' in '%s' field of '%s' (%s)", id, sh.Name(), name, tp)
					}
					sh.obj.Set("$tag", id)
					tgs[id] = true
				}
				if err := check(sh, sh.Name()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Sheme) init() error {
	obj, err := s.obj.Compile()
	if err != nil {
		return err
	}

	mp, _ := obj.Map()
	for k, v := range mp {
		if j, ok := v.(map[string]interface{}); ok {
			if err = check(Wrap(j), k); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("invalid object '%s'", k)
		}
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
	return s.obj.Get("$type").MustString()
}

func (s *Sheme) TypeEn() int {
	return typeTag(s.Type())
}

func (s *Sheme) ID() int {
	return s.obj.Get("$id").MustInt()
}

func (s *Sheme) Index() int {
	if tp, err := s.obj.Get("$tag").Int(); err == nil {
		return tp
	}
	return s.ID()
}

func (s *Sheme) Optional() bool {
	return s.obj.Get("$optional").MustBool()
}

func (s *Sheme) Implicit() bool {
	if s.obj.Get("$explicit").MustBool() {
		return false
	}
	return s.obj.Get("$implicit").MustBool()
}

func (s *Sheme) Explicit() bool {
	if s.obj.Get("$implicit").MustBool() {
		return false
	}
	return s.obj.Get("$explicit").MustBool()
}

func (s *Sheme) Tagged() bool {
	_, tp := s.obj.CheckGet("$tag")
	return tp
}

func (s *Sheme) DefAttr() interface{} {
	return s.obj.Get("$default").Interface()
}

func (s *Sheme) MinAttr() int {
	return s.obj.Get("$min").MustInt()
}

func (s *Sheme) MaxAttr() int {
	return s.obj.Get("$max").MustInt()
}

func (s *Sheme) FormatAttr() string {
	return s.obj.Get("$format").MustString()
}

func (s *Sheme) FieldAttr() map[string]interface{} {
	return s.obj.Get("$field").MustMap()
}

func (s *Sheme) OfAttr() map[string]interface{} {
	return s.obj.Get("$of").MustMap()
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

func NewFieldList(fld map[string]interface{}) fieldList {
	ret := fieldList{lst: list.New()}

	for k, v := range fld {
		if obj, ok := v.(map[string]interface{}); ok {
			ret.Add(Wrap(obj, k))
		}
	}

	return ret
}

func (s *Sheme) FieldList() fieldList {
	return NewFieldList(s.FieldAttr())
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
