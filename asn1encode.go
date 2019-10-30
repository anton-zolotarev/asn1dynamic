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

func (th *AsnData) preprocess() int {
	th.len = 0

	if th.tag.tagged {
		// tag := makeTag(th.tag.taggedN, true)
		// tag.sub[0] = th
		// th = tag
		markAsTag(th, th.tag.taggedN)
	}

	if th.tag.tagConstructed {
		for i := 0; i < len(th.sub); i++ {
			if th.sub[i] != nil {
				th.len += th.sub[i].preprocess() + 2
			}
		}
	} else {
		th.len += len(th.data)
	}

	if th.tag.tagNumber >= 31 {
		th.len += base128IntLength(int64(th.tag.tagNumber))
	}

	if len(th.data) >= 128 {
		th.len += lengthInt(len(th.data))
	}
	return th.len
}

func (th *AsnData) encode(dst []byte) ([]byte, error) {
	var err error
	dst = appendTagAndLength(th, dst)

	if th.tag.tagConstructed {
		for i := 0; i < len(th.sub); i++ {
			if th.sub[i] != nil {
				dst, err = th.sub[i].encode(dst)
			} else {
				fld := th.sheme.FieldList()
				if sh := fld.FindID(i); sh != nil && !sh.Optional() {
					err = encodeShemeErr("'%s' miss not optional field '%s'", th.sheme.Name(), sh.Name())
				}
			}
		}
	} else {
		dst = append(dst, th.data...)
	}
	return dst, err
}

func (th *AsnData) Encode() ([]byte, error) {
	ln := th.preprocess()
	out := make([]byte, 0, ln)
	return th.encode(out)
}
