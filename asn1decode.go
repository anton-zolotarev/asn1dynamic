package asn1dynamic

import (
	"bytes"
	"fmt"

	"github.com/anton-zolotarev/go-simplejson"
)

const (
	tagEOC             = 0x00
	tagBOOLEAN         = 0x01
	tagINTEGER         = 0x02
	tagBIT_STR         = 0x03
	tagOCTET_STR       = 0x04
	tagNULL            = 0x05
	tagOID             = 0x06
	tagObjDescriptor   = 0x07
	tagEXTERNAL        = 0x08
	tagREAL            = 0x09
	tagENUMERATED      = 0x0A
	tagEMBEDDED_PDV    = 0x0B
	tagUTF8String      = 0x0C
	tagSEQUENCE        = 0x10
	tagSET             = 0x11
	tagNumericString   = 0x12
	tagPrintableString = 0x13
	tagTeletexString   = 0x14
	tagVideotexString  = 0x15
	tagIA5String       = 0x16
	tagUTCTime         = 0x17
	tagGeneralizedTime = 0x18
	tagGraphicString   = 0x19
	tagVisibleString   = 0x1A
	tagGeneralString   = 0x1B
	tagUniversalString = 0x1C
	tagBMPString       = 0x1E

	// Virtual
	tagCHOICE = -1
	tagANY    = -2
)

const (
	classUniversal       = 0
	classApplication     = 1
	classContextSpecific = 2
	classPrivate         = 3
)

type AsnTag struct {
	tagClass       int
	tagNumber      int
	tagConstructed bool

	implicit bool
	explicit bool
	tagged   bool
	taggedN  int
}

type AsnData struct {
	sheme *Sheme
	fdata []byte
	data  []byte
	len   int
	tag   AsnTag
	sub   []*AsnData
}

type AsnContext struct {
	parent *AsnContext
	tag    *AsnData
	od     string
}

var (
	debug    bool
	implicit bool
	explicit bool
)

func ImplicitMode() {
	implicit = true
	explicit = false
}

func ExplicitMode() {
	explicit = true
	implicit = false
}

func Debug(on ...bool) bool {
	if len(on) > 0 {
		debug = on[0]
	}
	return debug
}

func debugHex(data []byte) {
	if debug {
		var buff bytes.Buffer
		buff.WriteString(fmt.Sprintf("len %d\n", len(data)))
		for i := 0; i < len(data); i++ {
			buff.WriteString(fmt.Sprintf("%02X ", data[i]))
		}
		fmt.Println(buff.String())
	}
}

func debugPrint(frm string, arg ...interface{}) {
	if debug {
		fmt.Printf(frm, arg...)
		fmt.Print("\n")
	}
}

func debugPrintln(arg ...interface{}) {
	if debug {
		fmt.Println(arg...)
	}
}

func Errorf(frm string, arg ...interface{}) error {
	err := fmt.Errorf(frm, arg...)
	debugPrint(err.Error())
	return err
}

func decodeTypeErr(tgn string, sheme *Sheme) error {
	return Errorf("decode: processing '%s' expected %s field but got %s", sheme.Name(), sheme.Type(), tgn)
}

func decodeDataErr(frm string, arg ...interface{}) error {
	return Errorf("decode: invalid value. %s", fmt.Sprintf(frm, arg...))
}

func decodeShemeErr(frm string, arg ...interface{}) error {
	return Errorf("decode: invalid sheme. %s", fmt.Sprintf(frm, arg...))
}

func typeName(tag int) string {
	switch tag {
	case tagEOC:
		return "EOC"
	case tagBOOLEAN:
		return "BOOLEAN"
	case tagINTEGER:
		return "INTEGER"
	case tagBIT_STR:
		return "BIT_STRING"
	case tagOCTET_STR:
		return "OCTET_STRING"
	case tagNULL:
		return "NULL"
	case tagOID:
		return "ObjectIdentifier"
	case tagObjDescriptor:
		return "ObjectDescriptor"
	case tagEXTERNAL:
		return "EXTERNAL"
	case tagREAL:
		return "REAL"
	case tagENUMERATED:
		return "ENUMERATED"
	case tagEMBEDDED_PDV:
		return "EMBEDDED_PDV"
	case tagUTF8String:
		return "UTF8String"
	case tagSEQUENCE:
		return "SEQUENCE"
	case tagSET:
		return "SET"
	case tagNumericString:
		return "NumericString"
	case tagPrintableString:
		return "PrintableString"
	case tagTeletexString:
		return "TeletexString"
	case tagVideotexString:
		return "VideotexString"
	case tagIA5String:
		return "IA5String"
	case tagUTCTime:
		return "UTCTime"
	case tagGeneralizedTime:
		return "GeneralizedTime"
	case tagGraphicString:
		return "GraphicString"
	case tagVisibleString:
		return "VisibleString"
	case tagGeneralString:
		return "GeneralString"
	case tagUniversalString:
		return "UniversalString"
	case tagBMPString:
		return "BMPString"

	case tagCHOICE:
		return "CHOICE"
	case tagANY:
		return "ANY"
	}
	return fmt.Sprint("Universal", tag)
}

func typeTag(tag string) int {
	switch tag {
	case "EOC":
		return tagEOC
	case "BOOLEAN":
		return tagBOOLEAN
	case "INTEGER":
		return tagINTEGER
	case "BIT_STRING":
		return tagBIT_STR
	case "OCTET_STRING":
		return tagOCTET_STR
	case "NULL":
		return tagNULL
	case "ObjectIdentifier":
		return tagOID
	case "ObjectDescriptor":
		return tagObjDescriptor
	case "EXTERNAL":
		return tagEXTERNAL
	case "REAL":
		return tagREAL
	case "ENUMERATED":
		return tagENUMERATED
	case "EMBEDDED_PDV":
		return tagEMBEDDED_PDV
	case "UTF8String":
		return tagUTF8String
	case "SEQUENCE":
		return tagSEQUENCE
	case "SET":
		return tagSET
	case "NumericString":
		return tagNumericString
	case "PrintableString":
		return tagPrintableString
	case "TeletexString":
		return tagTeletexString
	case "VideotexString":
		return tagVideotexString
	case "IA5String":
		return tagIA5String
	case "UTCTime":
		return tagUTCTime
	case "GeneralizedTime":
		return tagGeneralizedTime
	case "GraphicString":
		return tagGraphicString
	case "VisibleString":
		return tagVisibleString
	case "GeneralString":
		return tagGeneralString
	case "UniversalString":
		return tagUniversalString
	case "BMPString":
		return tagBMPString

	case "CHOICE":
		return tagCHOICE
	case "ANY":
		return tagANY
	}
	return tagEOC
}

func (th *AsnTag) typeName() string {
	switch th.tagClass {
	case classUniversal:
		return typeName(th.tagNumber)
	case classApplication:
		return fmt.Sprint("Application", th.tagNumber)
	case classContextSpecific:
		return fmt.Sprint("[", th.tagNumber, "]")
	case classPrivate:
		return fmt.Sprint("Private", th.tagNumber)
	}
	return "Unknown tag"
}

func (th *AsnTag) parse(data []byte) (pos int, err error) {
	th.tagClass = int(data[pos] >> 6)
	th.tagConstructed = ((data[pos] & 0x20) != 0)
	th.tagNumber = int(data[pos] & 0x1F)
	pos++
	if th.tagNumber == 0x1f {
		th.tagNumber, pos, err = parseBase128Int(data, pos)
		if err != nil {
			return
		}
		// Tags should be encoded in minimal form.
		if th.tagNumber < 0x1f {
			err = Errorf("non-minimal tag")
			return
		}
	}
	if pos >= len(data) {
		err = Errorf("truncated tag or length")
		return
	}
	return
}

func (th *AsnTag) isEOC() bool {
	return th.tagClass == 0x00 && th.tagNumber == 0x00
}

func (th *AsnData) reset() {
	th.sub = th.sub[0:0]
}

func (th *AsnData) RawData() []byte {
	return th.fdata
}

func (th *AsnData) Parse(data []byte) ([]byte, bool, error) {
	if len(data) < 2 {
		return data, false, nil
	}
	// считываем тег
	pos, err := th.tag.parse(data)
	if err != nil {
		return data, false, err
	}
	// считываем длину
	th.len = int(data[pos] & 0x7F)
	if th.len != int(data[pos]) {
		buf := 0
		for ; pos-1 < th.len; pos++ {
			buf = (buf * 256) + int(data[pos+1])
		}
		th.len = buf
	}
	pos++

	if len(data)-pos < th.len {
		return data, false, nil
	}

	th.reset()
	th.fdata = data[:pos+th.len]
	th.data = th.fdata[pos:]
	debugPrint("Parse: %s len: %d", th.tag.typeName(), th.len)
	if th.tag.tagConstructed {
		debugPrint("[")
		buf := th.data
		for ok := true; len(buf) > 0 && ok; {
			var asn AsnData
			if buf, ok, err = asn.Parse(buf); ok {
				th.sub = append(th.sub, &asn)
			}
		}
		debugPrint("]")
		if err != nil {
			return data, false, err
		}
		if len(buf) > 0 {
			return data, false, nil
		}
	}
	return data[len(th.fdata):], true, nil
}

func (th *AsnData) decode(sheme *Sheme, ctx *AsnContext) (res interface{}, err error) {
	if sheme == nil {
		return nil, Errorf("Error sheme is nil")
	}

	if debug {
		defer func() {
			if err == nil {
				debugPrintln(sheme.Type(), "result:", res, "\n")
			}
		}()
	}

	markTag(th, sheme)

	switch typeTag(sheme.Type()) {
	case tagNULL:
		return th.parseNull(sheme, ctx)
	case tagBOOLEAN:
		return th.parseBool(sheme, ctx)
	case tagINTEGER:
		return th.parseInt64(sheme, ctx)
	case tagENUMERATED:
		return th.parseEnumerated(sheme, ctx)
	case tagREAL:
		return th.parseReal(sheme, ctx)
	case tagUTF8String:
		return th.parseUTF8String(sheme, ctx)
	case tagNumericString:
		return th.parseNumericString(sheme, ctx)
	case tagPrintableString:
		return th.parsePrintableString(sheme, ctx)
	case tagOCTET_STR:
		return th.parseOctetString(sheme, ctx)
	case tagBIT_STR:
		return th.parseBitString(sheme, ctx)
	case tagOID:
		return th.parseObjectIdentifier(sheme, ctx)
	case tagObjDescriptor:
		return th.parseObjectDescriptor(sheme, ctx)
	case tagUTCTime:
		return th.parseUTCTime(sheme, ctx)
	case tagGeneralizedTime:
		return th.parseGeneralizedTime(sheme, ctx)
	case tagSEQUENCE:
		if sheme.OfAttr() != nil {
			return th.parseSequenceOf(sheme, ctx)
		}
		if sheme.FieldAttr() != nil {
			return th.parseSequence(sheme, ctx)
		}
		return nil, decodeShemeErr("Sequence '%s' does not contain '$field' or '$of'", sheme.Name())
	case tagCHOICE:
		return th.parseChoice(sheme, ctx)
	case tagANY:
		return th.parseAny(sheme, ctx)
	}
	return nil, Errorf("AsnData.decode: '%s' of unknown type '%s'", sheme.Name(), sheme.Type())
}

func (th *AsnData) Decode(sheme *Sheme) (*simplejson.Json, error) {
	ctx := &AsnContext{}
	ret, err := th.decode(sheme, ctx)
	if err != nil {
		return nil, err
	}
	return simplejson.Wrap(ret), nil
}

func NewDecoder() AsnElm {
	return &AsnData{}
}
