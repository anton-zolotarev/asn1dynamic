package asn1dynamic

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"
	"unicode/utf8"
)

type OID []int

func (oi OID) String() string {
	var s string
	for i, v := range oi {
		if i > 0 {
			s += "."
		}
		s += strconv.Itoa(v)
	}
	return s
}

// BitString is the structure to use when you want an ASN.1 BIT STRING type. A
// bit string is padded up to the nearest byte in memory and the number of
// valid bits is recorded. Padding bits will be zero.
type BitStr struct {
	Bytes     []byte // bits packed into bytes.
	BitLength int    // length in bits.
}

// At returns the bit at the given index. If the index is out of range it
// returns false.
func (b BitStr) At(i int) int {
	if i < 0 || i >= b.BitLength {
		return 0
	}
	x := i / 8
	y := 7 - uint(i%8)
	return int(b.Bytes[x]>>y) & 1
}

// RightAlign returns a slice where the padding bits are at the beginning. The
// slice may share memory with the BitString.
func (b BitStr) RightAlign() []byte {
	shift := uint(8 - (b.BitLength % 8))
	if shift == 8 || len(b.Bytes) == 0 {
		return b.Bytes
	}

	a := make([]byte, len(b.Bytes))
	a[0] = b.Bytes[0] >> shift
	for i := 1; i < len(b.Bytes); i++ {
		a[i] = b.Bytes[i-1] << (8 - shift)
		a[i] |= b.Bytes[i] >> shift
	}
	return a
}

// parseBase128Int parses a base-128 encoded int from the given offset in the
// given byte slice. It returns the value and the new offset.
func parseBase128Int(bytes []byte, initOffset int) (ret, offset int, err error) {
	offset = initOffset
	var ret64 int64
	for shifted := 0; offset < len(bytes); shifted++ {
		// 5 * 7 bits per byte == 35 bits of data
		// Thus the representation is either non-minimal or too large for an int32
		if shifted == 5 {
			err = decodeDataErr("base 128 integer too large")
			return
		}
		ret64 <<= 7
		b := bytes[offset]
		ret64 |= int64(b & 0x7f)
		offset++
		if b&0x80 == 0 {
			ret = int(ret64)
			// Ensure that the returned value fits in an int on all platforms
			if ret64 > math.MaxInt32 {
				err = decodeDataErr("base 128 integer too large")
			}
			return
		}
	}
	err = decodeDataErr("truncated base 128 integer")
	return
}

func checkInteger(th *AsnData) error {
	if th.len == 0 {
		return decodeDataErr("'%s' int len", th.tag.typeName())
	}
	if th.len == 1 {
		return nil
	}
	if (th.data[0] == 0 && th.data[1]&0x80 == 0) || (th.data[0] == 0xff && th.data[1]&0x80 == 0x80) {
		return decodeDataErr("'%s' integer not minimally-encoded", th.tag.typeName())
	}
	return nil
}

// isNumeric reports whether the given b is in the ASN.1 NumericString set.
func isNumeric(b byte) bool {
	return '0' <= b && b <= '9' ||
		b == ' '
}

// isPrintable reports whether the given b is in the ASN.1 PrintableString set.
// If asterisk is allowAsterisk then '*' is also allowed, reflecting existing
// practice. If ampersand is allowAmpersand then '&' is allowed as well.
func isPrintable(b byte, asterisk bool, ampersand bool) bool {
	return 'a' <= b && b <= 'z' ||
		'A' <= b && b <= 'Z' ||
		'0' <= b && b <= '9' ||
		'\'' <= b && b <= ')' ||
		'+' <= b && b <= '/' ||
		b == ' ' ||
		b == ':' ||
		b == '=' ||
		b == '?' ||
		// This is technically not allowed in a PrintableString.
		// However, x509 certificates with wildcard strings don't
		// always use the correct string type so we permit it.
		(bool(asterisk) && b == '*') ||
		// This is not technically allowed either. However, not
		// only is it relatively common, but there are also a
		// handful of CA certificates that contain it. At least
		// one of which will not expire until 2027.
		(bool(ampersand) && b == '&')
}

func strRestrict(str string, sheme *Sheme) bool {
	if min := sheme.MinAttr(); min > 0 && len(str) < min {
		return false
	}
	if max := sheme.MaxAttr(); max > 0 && len(str) > max {
		return false
	}
	return true
}

func intRestrict(i int, sheme *Sheme) bool {
	if min := sheme.MinAttr(); min > 0 && i < min {
		return false
	}
	if max := sheme.MaxAttr(); max > 0 && i > max {
		return false
	}
	return true
}

func (th *AsnData) castTag(sheme *Sheme, ctx *AsnContext) *AsnData {
	cast := false
	name := sheme.Type()

	if th.tag.tagClass == classContextSpecific && th.tag.tagNumber == sheme.Index() {
		if !th.tag.tagConstructed {
			cast = true
		} else if len(th.sub) == 1 {
			cast = true
			th = th.sub[0]
		}
	}
	if cast {
		tag := *th
		if typeTag(name) != tagEOC {
			tag.tag.tagNumber = typeTag(name)
			tag.tag.tagClass = classUniversal
		}
		th = &tag
	}
	return th
}

func (th *AsnData) parseNull(sheme *Sheme, ctx *AsnContext) (ret interface{}, err error) {
	debugPrint("parseNull: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagNULL {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
	}
	return
}

func (th *AsnData) parseBool(sheme *Sheme, ctx *AsnContext) (ret bool, err error) {
	debugPrint("parseBool: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagBOOLEAN {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}

	if th.len != 1 {
		err = decodeDataErr("'%s' wrong length %d", th.tag.typeName(), th.len)
		return
	}

	switch th.data[0] {
	case 0:
		ret = false
	case 0xff:
		ret = true
	default:
		err = decodeDataErr("'%s' wrong value %x", th.tag.typeName(), th.data[0])
	}
	return
}

func (th *AsnData) parseInt64(sheme *Sheme, ctx *AsnContext) (ret int64, err error) {
	debugPrint("parseInt64: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagINTEGER {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}

	if err = checkInteger(th); err != nil {
		return
	}

	if th.len > 8 {
		err = decodeDataErr("'%s' integer too large len: %d", th.tag.typeName(), th.len)
		return
	}

	for bytesRead := 0; bytesRead < th.len; bytesRead++ {
		ret <<= 8
		ret |= int64(th.data[bytesRead])
	}

	// Shift up and down in order to sign extend the result.
	ret <<= 64 - uint8(th.len)*8
	ret >>= 64 - uint8(th.len)*8

	if !intRestrict(int(ret), sheme) {
		err = decodeDataErr("'%s' out of range value: %d", th.tag.typeName(), ret)
		ret = 0
	}
	return
}

func (th *AsnData) parseInt32(sheme *Sheme, ctx *AsnContext) (ret int32, err error) {
	debugPrint("parseInt32: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	ret64, err := th.parseInt64(sheme, ctx)
	if err != nil {
		return
	}
	if ret64 != int64(int32(ret64)) {
		err = decodeDataErr("%s integer too large", th.tag.typeName())
		return
	}
	ret = int32(ret64)
	if !intRestrict(int(ret), sheme) {
		err = decodeDataErr("'%s' out of range value: %d", th.tag.typeName(), ret)
		ret = 0
	}
	return
}

func (th *AsnData) parseEnumerated(sheme *Sheme, ctx *AsnContext) (ret string, err error) {
	debugPrint("parseEnumerated: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagENUMERATED {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}

	tag := *th
	tag.tag.tagNumber = tagINTEGER
	val, err := tag.parseInt32(sheme, ctx)
	if err != nil {
		return
	}

	enm := sheme.EnumItems()
	ret, ok := enm[int(val)]
	if !ok {
		err = decodeDataErr("'%s' wrong value: %d", th.tag.typeName(), val)
	}
	return
}

func (th *AsnData) parseBitString(sheme *Sheme, ctx *AsnContext) (ret BitStr, err error) {
	debugPrint("parseBitString: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagBIT_STR {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}
	if th.len == 0 {
		err = decodeDataErr("'%s' zero length", th.tag.typeName())
		return
	}

	paddingBits := int(th.data[0])
	if paddingBits > 7 ||
		th.len == 1 && paddingBits > 0 ||
		th.data[th.len-1]&((1<<th.data[0])-1) != 0 {
		err = decodeDataErr("'%s' invalid padding bits", th.tag.typeName())
		return
	}
	ret.BitLength = (th.len-1)*8 - paddingBits
	ret.Bytes = th.data[1:]
	return
}

func (th *AsnData) parseObjectDescriptor(sheme *Sheme, ctx *AsnContext) (res string, err error) {
	debugPrint("parseObjectDescriptor: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagObjDescriptor {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}
	res = string(th.data)
	ctx.od = res
	return
}

func (th *AsnData) parseObjectIdentifier(sheme *Sheme, ctx *AsnContext) (res OID, err error) {
	debugPrint("parseObjectIdentifier: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagOID {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}
	if th.len == 0 {
		err = decodeDataErr("'%s' zero length", th.tag.typeName())
		return
	}

	// In the worst case, we get two elements from the first byte (which is
	// encoded differently) and then every varint is a single byte long.
	res = make([]int, th.len+1)

	// The first varint is 40*value1 + value2:
	// According to this packing, value1 can take the values 0, 1 and 2 only.
	// When value1 = 0 or value1 = 1, then value2 is <= 39. When value1 = 2,
	// then there are no restrictions on value2.
	v, offset, err := parseBase128Int(th.data, 0)
	if err != nil {
		return
	}
	if v < 80 {
		res[0] = v / 40
		res[1] = v % 40
	} else {
		res[0] = 2
		res[1] = v - 80
	}

	i := 2
	for ; offset < th.len; i++ {
		v, offset, err = parseBase128Int(th.data, offset)
		if err != nil {
			return
		}
		res[i] = v
	}
	res = res[0:i]
	return
}

func (th *AsnData) parseUTCTime(sheme *Sheme, ctx *AsnContext) (ret time.Time, err error) {
	debugPrint("parseUTCTime: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagUTCTime {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}

	s := string(th.data)

	formatStr := sheme.FormatAttr()
	if formatStr == "" {
		formatStr = "0601021504Z0700"
	}

	ret, err = time.Parse(formatStr, s)
	if err != nil {
		formatStr = "060102150405Z0700"
		ret, err = time.Parse(formatStr, s)
	}
	if err != nil {
		return
	}

	if serialized := ret.Format(formatStr); serialized != s {
		err = fmt.Errorf("asn1: time did not serialize back to the original value and may be invalid: given %q, but serialized as %q", s, serialized)
		return
	}

	if ret.Year() >= 2050 {
		// UTCTime only encodes times prior to 2050. See https://tools.ietf.org/html/rfc5280#section-4.1.2.5.1
		ret = ret.AddDate(-100, 0, 0)
	}
	return
}

// parseGeneralizedTime parses the GeneralizedTime from the given byte slice
// and returns the resulting time.
func (th *AsnData) parseGeneralizedTime(sheme *Sheme, ctx *AsnContext) (ret time.Time, err error) {
	debugPrint("parseGeneralizedTime: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagGeneralizedTime {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}

	const formatStr = "20060102150405Z0700"
	s := string(th.data)
	if ret, err = time.Parse(formatStr, s); err != nil {
		return
	}

	if serialized := ret.Format(formatStr); serialized != s {
		err = fmt.Errorf("asn1: time did not serialize back to the original value and may be invalid: given %q, but serialized as %q", s, serialized)
	}
	return
}

func (th *AsnData) parseNumericString(sheme *Sheme, ctx *AsnContext) (ret string, err error) {
	debugPrint("parseNumericString: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagNumericString {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}

	for _, b := range th.data {
		if !isNumeric(b) {
			err = decodeDataErr("'%s' contains invalid character: %c", th.tag.typeName(), b)
			return
		}
	}
	str := string(th.data)
	if !strRestrict(str, sheme) {
		err = decodeDataErr("'%s' contains invalid length: %d", th.tag.typeName(), len(str))
		return
	}

	ret = str
	return
}

func (th *AsnData) parsePrintableString(sheme *Sheme, ctx *AsnContext) (ret string, err error) {
	debugPrint("parsePrintableString: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagPrintableString {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}

	for _, b := range th.data {
		if !isPrintable(b, true, true) {
			err = decodeDataErr("'%s' contains invalid character: %c", th.tag.typeName(), b)
			return
		}
	}
	str := string(th.data)
	if !strRestrict(str, sheme) {
		err = decodeDataErr("'%s' contains invalid length: %d", th.tag.typeName(), len(str))
		return
	}

	ret = str
	return
}

func (th *AsnData) parseIA5String(sheme *Sheme, ctx *AsnContext) (ret string, err error) {
	debugPrint("parseIA5String: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagIA5String {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}
	for _, b := range th.data {
		if b >= utf8.RuneSelf {
			err = decodeDataErr("'%s' contains invalid character: %x", th.tag.typeName(), b)
			return
		}
	}
	str := string(th.data)
	if !strRestrict(str, sheme) {
		err = decodeDataErr("'%s' contains invalid length: %d", th.tag.typeName(), len(str))
		return
	}

	ret = str
	return
}

func (th *AsnData) parseUTF8String(sheme *Sheme, ctx *AsnContext) (ret string, err error) {
	debugPrint("parseUTF8String: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagUTF8String {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}

	if !utf8.Valid(th.data) {
		err = decodeDataErr("'%s' invalid UTF-8 string", th.tag.typeName())
		return
	}
	str := string(th.data)
	if !strRestrict(str, sheme) {
		err = decodeDataErr("'%s' contains invalid length: %d", th.tag.typeName(), len(str))
		return
	}

	ret = str
	return
}

func (th *AsnData) parseOctetString(sheme *Sheme, ctx *AsnContext) (ret []byte, err error) {
	debugPrint("parseOctetString: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagOCTET_STR {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}
	if !strRestrict(string(th.data), sheme) {
		err = decodeDataErr("'%s' contains invalid length: %d", th.tag.typeName(), len(th.data))
		return
	}
	ret = th.data
	return
}

func (th *AsnData) parseSequence(sheme *Sheme, ctx *AsnContext) (ret map[string]interface{}, err error) {
	debugPrint("parseSequence: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagSEQUENCE {
		return nil, decodeTypeErr(tho.tag.typeName(), sheme)
	}
	if !th.tag.tagConstructed {
		return nil, decodeDataErr("'%s' not constructed", tho.tag.typeName())
	}

	ls := sheme.SeqItems()
	if len(ls) == 0 {
		return nil, decodeShemeErr("'%s' cannot find any field in sheme", th.tag.typeName())
	}

	idx := 0
	ret = make(map[string]interface{})
	ctxn := &AsnContext{parent: ctx, tag: th}
	for _, sh := range ls {
		var dt interface{}
		if idx < len(th.sub) {
			dt, err = th.sub[idx].decode(sh, ctxn)
		} else {
			err = fmt.Errorf("miss field %s %s", sh.Name(), sh.Type())
		}

		if err == nil {
			ret[sh.Name()] = dt
			idx++
		} else if sh.Optional() {
			if def := sh.DefAttr(); def != nil {
				ret[sh.Name()] = def
			}
			continue
		} else {
			return nil, err
		}
	}
	return ret, nil
}

func (th *AsnData) parseSequenceOf(sheme *Sheme, ctx *AsnContext) (ret []interface{}, err error) {
	debugPrint("parseSequenceOf: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagSEQUENCE {
		return nil, decodeTypeErr(tho.tag.typeName(), sheme)
	}
	if !th.tag.tagConstructed {
		return nil, decodeDataErr("'%s' not constructed", tho.tag.typeName())
	}

	sh := sheme.Of()

	ret = make([]interface{}, len(th.sub))

	ctxn := &AsnContext{parent: ctx, tag: th}
	for k, v := range th.sub {
		ret[k], err = v.decode(sh, ctxn)
		if err != nil {
			return
		}
	}
	return ret, nil
}

func (th *AsnData) parseChoice(sheme *Sheme, ctx *AsnContext) (ret interface{}, err error) {
	debugPrint("parseChoice: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	ls := sheme.SeqItems()
	if len(ls) == 0 {
		return nil, decodeShemeErr("'%s' cannot find any field in sheme", th.tag.typeName())
	}

	ctxn := &AsnContext{parent: ctx, tag: th}
	num := th.tag.tagNumber
	th = th.castTag(sheme, ctx)

	if num < len(ls) {
		debugPrint("parseChoice: primary %d (%s)", ls[num].Index(), ls[num].Name())
		ret, err = th.decode(ls[num], ctxn)
	}

	if err != nil {
		for id, sh := range ls {
			debugPrint("parseChoice: choice %d (%s)", sh.Index(), sh.Name())
			if ret, err = th.decode(sh, ctxn); err == nil {
				num = id
				break
			}
		}
	}

	if err == nil {
		ret2 := make(map[string]interface{})
		ret2[ls[num].Name()] = ret
		ret = ret2
		return
	}
	return nil, decodeDataErr("'%s' can not parse", th.tag.typeName())
}

func (th *AsnData) parseAny(sheme *Sheme, ctx *AsnContext) (ret interface{}, err error) {
	debugPrint("parseAny: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	if ctx.od == "" {
		return nil, decodeDataErr("'%s' miss ObjectDescriptor", th.tag.typeName())
	}

	sh := sheme.Field(ctx.od)
	if sh == nil {
		return nil, decodeDataErr("'%s' unknown ObjectDescriptor %s", th.tag.typeName(), ctx.od)
	}
	ctxn := &AsnContext{parent: ctx, tag: th}
	return th.decode(sh, ctxn)
}

func (th *AsnData) parseReal(sheme *Sheme, ctx *AsnContext) (ret float64, err error) {
	debugPrint("parseReal: %s (%s) tag %s", sheme.Name(), sheme.Type(), th.tag.typeName())
	debugHex(th.data)
	tho, th := th, th.castTag(sheme, ctx)
	if th.tag.tagNumber != tagREAL {
		err = decodeTypeErr(tho.tag.typeName(), sheme)
		return
	}

	if th.len == 0 {
		ret = 0.0
	} else if th.data[0]&0x80 != 0 {
		ret, err = decode_real_binary(th.data)
	} else if th.data[0]&0x40 != 0 {
		ret, err = decode_real_special(th.data[0])
	} else {
		ret, err = decode_real_decimal(th.data)
	}

	return
}

func decode_real_decimal(data []byte) (float64, error) {
	return strconv.ParseFloat(unsafe_slice2str(data[1:]), 64)
}

func decode_real_special(control byte) (float64, error) {
	switch control {
	case 0x40:
		return strconv.ParseFloat("+Inf", 32)
	case 0x41:
		return strconv.ParseFloat("-Inf", 32)
	case 0x42:
		return strconv.ParseFloat("NaN", 32)
	case 0x43:
		return -0.0, nil
	}
	return 0.0, decodeDataErr("Unsupported special REAL control word %x.", control)
}

func decode_real_binary(data []byte) (float64, error) {
	var exponent int
	offset := 0
	control := data[0]

	if control == 0x80 || control == 0xc0 {
		exponent = int(data[1])
		if exponent&0x80 != 0 {
			exponent -= 0x100
		}
		offset = 2

	} else if control == 0x81 || control == 0xc1 {
		exponent = ((int(data[1]) << 8) | int(data[2]))

		if exponent&0x8000 != 0 {
			exponent -= 0x10000
		}
		offset = 3

	} else {
		return 0.0, decodeDataErr("Unsupported binary REAL control word %x", control)
	}

	// switch (control & 0x30) >> 4 {
	// case 0x00:
	// 	baseF = 1
	// 	break /* base 2 */
	// case 0x01:
	// 	baseF = 3
	// 	break /* base 8 */
	// case 0x02:
	// 	baseF = 4
	// 	break /* base 16 */
	// default:
	// 	/* Reserved field, can't parse now. */
	// }

	mantissa, _ := strconv.ParseInt(hex.EncodeToString(data[offset:]), 16, 64)
	decoded := float64(mantissa) * math.Pow(2, float64(exponent))

	if control&0x40 > 0 {
		decoded *= -1
	}

	return decoded, nil
}
