package asn1dynamic

import "fmt"

func encodeTypeErr(tgn string, sheme *Sheme) error {
	return Errorf("encode: processing '%s' expected %s field but got %s", sheme.Name(), sheme.Type(), tgn)
}

func encodeDataErr(frm string, arg ...interface{}) error {
	return Errorf("encode: invalid value. %s", fmt.Sprintf(frm, arg...))
}

func encodeShemeErr(frm string, arg ...interface{}) error {
	return Errorf("encode: invalid sheme. %s", fmt.Sprintf(frm, arg...))
}

func appendTagAndLength(th *AsnData, dst []byte) []byte {
	b := uint8(th.tag.tagClass) << 6
	if th.tag.tagConstructed {
		b |= 0x20
	}
	if th.tag.tagNumber >= 31 {
		b |= 0x1f
		dst = append(dst, b)
		dst = appendBase128Int(dst, int64(th.tag.tagNumber))
	} else {
		b |= uint8(th.tag.tagNumber)
		dst = append(dst, b)
	}

	if th.len >= 128 {
		l := lengthInt(th.len)
		dst = append(dst, 0x80|byte(l))
		dst = appendInt(dst, th.len)
	} else {
		dst = append(dst, byte(th.len))
	}

	return dst
}

func (th *AsnData) preprocess(parent *AsnData, idx int) int {
	th.len = 0

	debugPrint("Prepare: %s (%s)", th.tag.typeName(), th.sheme.Name())
	if th.tag.tagClass == classUniversal && th.tag.tagNumber < tagEOC && len(th.sub) == 1 {
		parent.sub[idx] = th.sub[0]
		th = parent.sub[idx]
		debugPrint("Cast: %s (%s)", th.tag.typeName(), th.sheme.Name())
		return th.preprocess(parent, idx)
	}

	if th.tag.tagged {
		if th.tag.implicit {
			debugPrint("implicit %d", th.tag.taggedN)
			th.tag.tagClass = classContextSpecific
			th.tag.tagNumber = th.tag.taggedN
		} else {
			debugPrint("explicit %d", th.tag.taggedN)
			th.tag.tagged = false
			parent.sub[idx] = makeTag(classContextSpecific, th.tag.taggedN, 1)
			parent.sub[idx].sub[0] = th
			th = parent.sub[idx]
		}
	}

	if th.tag.tagConstructed {
		debugPrint("[")
		for i := 0; i < len(th.sub); i++ {
			if th.sub[i] != nil {
				len := th.sub[i].preprocess(th, i)
				if len >= 128 {
					th.len += lengthInt(len)
				}
				th.len += len + 2
			}
		}
		debugPrint("]")
	} else {
		th.len += len(th.data)
	}

	if th.tag.tagNumber >= 31 {
		th.len += base128IntLength(int64(th.tag.tagNumber))
	}

	return th.len
}

func (th *AsnData) encode(dst []byte) ([]byte, error) {
	var err error
	pos := len(dst)
	dst = appendTagAndLength(th, dst)

	debugPrint("Encode: %s len: %d", th.tag.typeName(), th.len)
	if th.tag.tagConstructed {
		debugPrint("[")
		for i := 0; i < len(th.sub) && err == nil; i++ {
			if th.sub[i] != nil {
				dst, err = th.sub[i].encode(dst)
			} else {
				fld := th.sheme.FieldList()
				if sh := fld.FindID(i); sh != nil && !sh.Optional() {
					err = encodeShemeErr("'%s' miss not optional field '%s'", th.sheme.Name(), sh.Name())
				}
			}
		}
		debugPrint("]")
	} else {
		dst = append(dst, th.data...)
	}
	th.fdata = dst[pos:]
	return dst, err
}

func (th *AsnData) Encode() ([]byte, error) {
	len := th.preprocess(nil, 0)
	if len >= 128 {
		len += lengthInt(len)
	}
	len += 2
	out := make([]byte, 0, len)
	return th.encode(out)
}
