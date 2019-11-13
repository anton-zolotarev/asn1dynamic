package asn1dynamic

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/bits"
	"reflect"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/anton-zolotarev/go-simplejson"
)

func unsafe_slice2str(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

func base128IntLength(n int64) int {
	if n == 0 {
		return 1
	}

	l := 0
	for i := n; i > 0; i >>= 7 {
		l++
	}
	return l
}

func appendBase128Int(dst []byte, n int64) []byte {
	l := base128IntLength(n)
	for i := l - 1; i >= 0; i-- {
		o := byte(n >> uint(i*7))
		o &= 0x7f
		if i != 0 {
			o |= 0x80
		}
		dst = append(dst, o)
	}
	return dst
}

func lengthInt(i int) (numBytes int) {
	numBytes = 1
	for i > 127 || i < -128 {
		numBytes++
		i >>= 8
	}
	return
}

func appendInt(dst []byte, val int) []byte {
	num := lengthInt(val)
	for ; num > 0; num-- {
		dst = append(dst, byte(val>>uint((num-1)*8)))
	}
	return dst
}

func encodeInt(val int) []byte {
	return appendInt(make([]byte, 0, lengthInt(val)), val)
}

func checkType(tag int, sheme *Sheme) error {
	if sheme == nil {
		return encodeShemeErr("'%s' no sheme description", typeName(tag))
	}
	if sheme.Type() != typeName(tag) {
		return encodeTypeErr(typeName(tag), sheme)
	}
	return nil
}

func makeType(sheme *Sheme, tag int) (*AsnData, error) {
	if err := checkType(tag, sheme); err != nil {
		return nil, err
	}

	out := AsnData{sheme: sheme}
	out.tag.tagClass = classUniversal
	out.tag.tagNumber = tag
	return &out, nil
}

func makeTag(tag int, constr bool) *AsnData {
	out := AsnData{}
	out.tag.tagClass = classContextSpecific
	out.tag.tagNumber = tag
	if constr {
		out.tag.tagConstructed = true
		out.sub = make([]*AsnData, 1)
	}
	return &out
}

func markAsTag(th *AsnData, tag int) {
	th.tag.tagClass = classContextSpecific
	th.tag.tagNumber = tag
}

func findField(sheme *Sheme, name string) (*Sheme, error) {
	sh := sheme.Field(name)
	if sh == nil {
		return nil, encodeShemeErr("'%s' does not contain the field '%s'", sheme.Name(), name)
	}
	return sh, nil
}

func findOf(sheme *Sheme) (*Sheme, error) {
	sh := sheme.Of()
	if sh == nil {
		return nil, encodeShemeErr("'%s' does not contain the field '$of'", sheme.Name())
	}
	return sh, nil
}

func this(elm AsnElm) *AsnData {
	return elm.(*AsnData)
}

func (sheme *Sheme) Null() (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagNULL); err == nil {
		out.data = []byte{0x00}
	}
	return out, err
}

func (sheme *Sheme) Boolean(val bool) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagBOOLEAN); err == nil {
		out.data = make([]byte, 1)
		if val {
			out.data[0] = 0xff
		}
	}
	return out, err
}

func (sheme *Sheme) Integer(val int) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagINTEGER); err == nil {
		if !intRestrict(val, sheme) {
			return nil, encodeDataErr("'%s' %s out of range value: %d", sheme.Name(), sheme.Type(), val)
		}
		out.data = encodeInt(val)
	}
	return out, err
}

func (sheme *Sheme) Enumerated(val string) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagENUMERATED); err == nil {
		itm, ok := sheme.FieldAttr()[val]
		if !ok {
			return nil, encodeDataErr("'%s' %s wrong value: '%s'", sheme.Name(), sheme.Type(), val)
		}
		out.data = encodeInt(simplejson.Wrap(itm).MustInt())
	}
	return out, err
}

func (sheme *Sheme) BitString(val BitStr) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagBIT_STR); err == nil {
		out.data = make([]byte, len(val.Bytes)+1)
		out.data[0] = byte((8 - val.BitLength%8) % 8)
		copy(out.data[1:], val.Bytes)
	}
	return out, err
}

func (sheme *Sheme) UTCTime(val time.Time) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagUTCTime); err == nil {
		formatStr := sheme.FormatAttr()
		if formatStr == "" {
			formatStr = "0601021504Z0700"
		}
		tm := val.Format(formatStr)
		out.data = make([]byte, len(tm))
		copy(out.data, tm)
	}
	return out, err
}

func (sheme *Sheme) GeneralizedTime(val time.Time) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagGeneralizedTime); err == nil {
		const formatStr = "20060102150405Z0700"
		tm := val.Format(formatStr)
		out.data = make([]byte, len(tm))
		copy(out.data, tm)
	}
	return out, err
}

func (sheme *Sheme) ObjectIdentifier(val OID) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagOID); err == nil {
		out.data = appendBase128Int(out.data[:0], int64(val[0]*40+val[1]))
		for i := 2; i < len(val); i++ {
			out.data = appendBase128Int(out.data, int64(val[i]))
		}
	}
	return out, err
}

func (sheme *Sheme) ObjectDescriptor(val string) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagObjDescriptor); err == nil {
		out.data = make([]byte, len(val))
		copy(out.data, val)
	}
	return out, err
}

func (sheme *Sheme) NumericString(val string) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagNumericString); err == nil {
		for i := 0; i < len(val); i++ {
			if !isNumeric(val[i]) {
				return nil, encodeDataErr("%s %s contains invalid character: %c", sheme.Name(), sheme.Type(), val[i])
			}
		}
		if !strRestrict(val, sheme) {
			return nil, encodeDataErr("%s %s contains invalid length: %d", sheme.Name(), sheme.Type(), len(val))
		}
		out.data = make([]byte, len(val))
		copy(out.data, val)
	}
	return out, err
}

func (sheme *Sheme) PrintableString(val string) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagPrintableString); err == nil {
		for i := 0; i < len(val); i++ {
			if !isPrintable(val[i], true, true) {
				return nil, encodeDataErr("%s %s contains invalid character: %c", sheme.Name(), sheme.Type(), val[i])
			}
		}
		if !strRestrict(val, sheme) {
			return nil, encodeDataErr("%s %s contains invalid length: %d", sheme.Name(), sheme.Type(), len(val))
		}
		out.data = make([]byte, len(val))
		copy(out.data, val)
	}
	return out, err
}

func (sheme *Sheme) IA5String(val string) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagIA5String); err == nil {
		for i := 0; i < len(val); i++ {
			if val[i] >= utf8.RuneSelf {
				return nil, encodeDataErr("%s %s contains invalid character: %c", sheme.Name(), sheme.Type(), val[i])
			}
		}
		if !strRestrict(val, sheme) {
			return nil, encodeDataErr("%s %s contains invalid length: %d", sheme.Name(), sheme.Type(), len(val))
		}
		out.data = make([]byte, len(val))
		copy(out.data, val)
	}
	return out, err
}

func (sheme *Sheme) UTF8String(val string) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagUTF8String); err == nil {
		if !strRestrict(val, sheme) {
			return nil, encodeDataErr("%s %s contains invalid length: %d", sheme.Name(), sheme.Type(), len(val))
		}
		out.data = make([]byte, len(val))
		copy(out.data, val)
	}
	return out, err
}

func (sheme *Sheme) OctetString(val []byte) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagOCTET_STR); err == nil {
		if !strRestrict(unsafe_slice2str(val), sheme) {
			return nil, encodeDataErr("%s %s contains invalid length: %d", sheme.Name(), sheme.Type(), len(val))
		}
		out.data = make([]byte, len(val))
		copy(out.data, val)
	}
	return out, err
}

func (sheme *Sheme) Sequence() (AsnSeq, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagSEQUENCE); err == nil {
		out.tag.tagConstructed = true
		fld := sheme.FieldAttr()
		if fld != nil {
			out.sub = make([]*AsnData, len(fld))
		} else if sheme.OfAttr() == nil {
			return nil, encodeShemeErr("Sequence '%s' does not contain $field or $of", sheme.Name())
		}
	}
	return out, err
}

func (th *AsnData) SeqFieldByName(name string, el AsnElm, err error) error {
	if err != nil {
		return err
	}
	dt := this(el)
	if th.sheme.Type() != "SEQUENCE" {
		return encodeShemeErr("'%s' does not a SEQUENCE", th.sheme.Name())
	}
	sh := th.sheme.Field(name)
	if sh == nil {
		return encodeShemeErr("'%s' does not contain '%s'", th.sheme.Name(), name)
	}
	if debug && !reflect.DeepEqual(sh.obj.Interface(), dt.sheme.obj.Interface()) {
		return encodeShemeErr("incompatible interfaces '%s' and '%s'", th.sheme.Name(), name)
	}
	id := sh.ID()
	if id >= len(th.sub) || th.sub[id] != nil {
		return encodeShemeErr("'%s' corrupt field id '%s'", th.sheme.Name(), name)
	}
	th.sub[id] = dt
	return nil
}

func (th *AsnData) SeqField(el AsnElm, err error) error {
	if err != nil {
		return err
	}
	return th.SeqFieldByName(this(el).sheme.Name(), el, err)
}

func (th *AsnData) SeqItem(el AsnElm, err error) error {
	if err != nil {
		return err
	}
	dt := this(el)
	if th.sheme.Type() != "SEQUENCE" {
		return encodeShemeErr("'%s' does not a SEQUENCE", th.sheme.Name())
	}
	sh := th.sheme.Of()
	if sh == nil {
		return encodeShemeErr("'%s' does not contain '$of' field", th.sheme.Name())
	}
	if debug && !reflect.DeepEqual(sh.obj.Interface(), dt.sheme.obj.Interface()) {
		return encodeShemeErr("incompatible interfaces '%s' and '%s'", th.sheme.Name(), dt.sheme.Name())
	}
	th.sub = append(th.sub, dt)
	return nil
}

func (sheme *Sheme) Choice() (AsnChoice, error) {
	var out *AsnData
	var err error
	if sheme.Type() != "CHOICE" {
		return nil, encodeShemeErr("'%s' does not a CHOICE", sheme.Name())
	}
	out = makeTag(0, true)
	out.sheme = sheme
	return out, err
}

func (th *AsnData) ChoiceSetByName(name string, el AsnElm, err error) error {
	if err != nil {
		return err
	}
	dt := this(el)
	if th.sheme.Type() != "CHOICE" {
		return encodeShemeErr("'%s' does not a CHOICE", th.sheme.Name())
	}
	sh := th.sheme.Field(name)
	if sh == nil {
		return encodeShemeErr("'%s' does not contain '%s'", th.sheme.Name(), name)
	}
	if debug && !reflect.DeepEqual(sh.obj.Interface(), dt.sheme.obj.Interface()) {
		return encodeShemeErr("incompatible interfaces '%s' and '%s'", sh.Name(), name)
	}
	th.tag.tagNumber = sh.Index()
	th.sub[0] = dt
	return nil
}

func (th *AsnData) ChoiceSet(el AsnElm, err error) error {
	if err != nil {
		return err
	}
	return th.ChoiceSetByName(this(el).sheme.Name(), el, err)
}

func (sheme *Sheme) Any() (AsnAny, error) {
	var out *AsnData
	var err error
	if sheme.Type() != "ANY" {
		return nil, encodeShemeErr("'%s' does not a ANY", sheme.Name())
	}
	out = &AsnData{sheme: sheme}
	out.tag.tagged = sheme.Tagged()
	out.tag.taggedN = sheme.Index()
	return out, err
}

func (th *AsnData) AnySetByName(name string, el AsnElm, err error) error {
	if err != nil {
		return err
	}
	dt := this(el)
	if th.sheme.Type() != "ANY" {
		return encodeShemeErr("'%s' does not a ANY", th.sheme.Name())
	}
	sh := th.sheme.Field(name)
	if sh == nil {
		return encodeShemeErr("'%s' does not contain '%s'", th.sheme.Name(), name)
	}
	if debug && !reflect.DeepEqual(sh.obj.Interface(), dt.sheme.obj.Interface()) {
		return encodeShemeErr("incompatible interfaces '%s' and '%s'", sh.Name(), name)
	}
	// th.tag = dt.tag
	// th.sub = dt.sub
	*th = *dt
	return nil
}

func (th *AsnData) AnySet(el AsnElm, err error) error {
	if err != nil {
		return err
	}
	return th.AnySetByName(this(el).sheme.Name(), el, err)
}

func lowest_set_bit(value int) int {
	offset := bits.Len(uint(value&-value)) - 1

	if offset < 0 {
		offset = 0
	}
	return offset
}

// https://github.com/eerimoq/asn1tools/blob/master/asn1tools/codecs/ber.py
func (sheme *Sheme) Real(val float64) (AsnElm, error) {
	var out *AsnData
	var err error
	if out, err = makeType(sheme, tagREAL); err == nil {
		if math.IsInf(val, 1) {
			out.data = []byte{0x40}
		} else if math.IsInf(val, -1) {
			out.data = []byte{0x41}
		} else if math.IsNaN(val) {
			out.data = []byte{0x42}
		} else if val == 0.0 {
			// out.data
		} else {
			negative_bit := byte(0)
			if val < 0 {
				negative_bit = 0x40
				val *= -1
			}

			mantissa, exponent := math.Frexp(math.Abs(val))
			mantissa_i := int(mantissa * math.Pow(2, 53))
			lowest_set_bit := lowest_set_bit(mantissa_i)
			mantissa_i >>= uint(lowest_set_bit)
			mantissa_i |= (0x80 << (8 * ((uint(bits.Len(uint(mantissa_i))) / 8) + 1)))
			mantissa_d, _ := hex.DecodeString(fmt.Sprintf("%x", uint(mantissa_i))[2:])
			exponent = (52 - lowest_set_bit - exponent)
			var exponent_d []byte

			if -129 < exponent && exponent < 128 {
				exponent_d = []byte{byte(0x80 | negative_bit), byte((0xff - exponent) & 0xff)}
			} else if -32769 < exponent && exponent < 32768 {
				exponent = ((0xffff - exponent) & 0xffff)
				exponent_d = []byte{0x81 | negative_bit, byte((exponent >> 8) & 0xff), byte(exponent & 0xff)}
			} else {
				encodeDataErr("REAL exponent out of range")
			}

			out.data = append(exponent_d, mantissa_d...)
		}
	}
	return out, err
}
