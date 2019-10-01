package asn1dynamic

import (
	"time"

	"github.com/anton-zolotarev/go-simplejson"
)

type AsnElm interface {
	Encode() ([]byte, error)
}

type AsnDec interface {
	AsnElm
	Decode(sheme *Sheme) (*simplejson.Json, error)
	Parse(data []byte, offset int) ([]byte, bool, error)
}

type AsnPath interface {
	PathNull(path ...string) error
	PathBoolean(val bool, path ...string) error
	PathInteger(val int, path ...string) error
	PathReal(val float64, path ...string) error
	PathEnumerated(val string, path ...string) error
	PathBitString(val BitStr, path ...string) error
	PathUTCTime(val time.Time, path ...string) error
	PathGeneralizedTime(val time.Time, path ...string) error
	PathObjectIdentifier(val OID, path ...string) error
	PathObjectDescriptor(val string, path ...string) error
	PathNumericString(val string, path ...string) error
	PathPrintableString(val string, path ...string) error
	PathIA5String(val string, path ...string) error
	PathUTF8String(val string, path ...string) error
	PathOctetString(val []byte, path ...string) error

	PathSequence(path ...string) (out AsnSeq, err error)
	PathChoice(path ...string) (out AsnChoice, err error)
	PathAny(path ...string) (out AsnAny, err error)
}

type AsnSeq interface {
	AsnElm
	AsnPath
	SeqFieldByName(name string, el AsnElm, err error) error
	SeqField(el AsnElm, err error) error
	SeqItem(el AsnElm, err error) error

	SetNull(name string) error
	SetBoolean(name string, val bool) error
	SetInteger(name string, val int) error
	SetReal(name string, val float64) error
	SetEnumerated(name string, val string) error
	SetBitString(name string, val BitStr) error
	SetUTCTime(name string, val time.Time) error
	SetGeneralizedTime(name string, val time.Time) error
	SetObjectIdentifier(name string, val OID) error
	SetObjectDescriptor(name string, val string) error
	SetNumericString(name string, val string) error
	SetPrintableString(name string, val string) error
	SetIA5String(name string, val string) error
	SetUTF8String(name string, val string) error
	SetOctetString(name string, val []byte) error

	SetSequence(name string) (out AsnSeq, err error)
	SetChoice(name string) (out AsnChoice, err error)
	SetAny(name string) (out AsnAny, err error)

	AddBoolean(val bool) error
	AddInteger(val int) error
	AddReal(val float64) error
	AddEnumerated(val string) error
	AddBitString(val BitStr) error
	AddUTCTime(val time.Time) error
	AddGeneralizedTime(val time.Time) error
	AddObjectIdentifier(val OID) error
	AddObjectDescriptor(val string) error
	AddNumericString(val string) error
	AddPrintableString(val string) error
	AddIA5String(val string) error
	AddUTF8String(val string) error
	AddOctetString(val []byte) error

	AddSequence() (out AsnSeq, err error)
	AddChoice() (out AsnChoice, err error)
	AddAny() (out AsnAny, err error)
}

type AsnChoice interface {
	AsnElm
	AsnPath
	ChoiceSetByName(name string, el AsnElm, err error) error
	ChoiceSet(el AsnElm, err error) error

	ChoiceNull(name string) error
	ChoiceBoolean(name string, val bool) error
	ChoiceInteger(name string, val int) error
	ChoiceReal(name string, val float64) error
	ChoiceEnumerated(name string, val string) error
	ChoiceBitString(name string, val BitStr) error
	ChoiceUTCTime(name string, val time.Time) error
	ChoiceGeneralizedTime(name string, val time.Time) error
	ChoiceObjectIdentifier(name string, val OID) error
	ChoiceObjectDescriptor(name string, val string) error
	ChoiceNumericString(name string, val string) error
	ChoicePrintableString(name string, val string) error
	ChoiceIA5String(name string, val string) error
	ChoiceUTF8String(name string, val string) error
	ChoiceOctetString(name string, val []byte) error

	ChoiceSequence(name string) (out AsnSeq, err error)
	ChoiceChoice(name string) (out AsnChoice, err error)
	ChoiceAny(name string) (out AsnAny, err error)
}

type AsnAny interface {
	AsnElm
	AsnPath
	AnySetByName(name string, el AsnElm, err error) error
	AnySet(el AsnElm, err error) error

	AnyNull(name string) error
	AnyBoolean(name string, val bool) error
	AnyInteger(name string, val int) error
	AnyReal(name string, val float64) error
	AnyEnumerated(name string, val string) error
	AnyBitString(name string, val BitStr) error
	AnyUTCTime(name string, val time.Time) error
	AnyGeneralizedTime(name string, val time.Time) error
	AnyObjectIdentifier(name string, val OID) error
	AnyObjectDescriptor(name string, val string) error
	AnyNumericString(name string, val string) error
	AnyPrintableString(name string, val string) error
	AnyIA5String(name string, val string) error
	AnyUTF8String(name string, val string) error
	AnyOctetString(name string, val []byte) error

	AnySequence(name string) (out AsnSeq, err error)
	AnyChoice(name string) (out AsnChoice, err error)
}

func setByType(th *AsnData, elm AsnElm, err error) error {
	switch th.sheme.Type() {
	case "SEQUENCE":
		return th.SeqField(elm, err)
	case "CHOICE":
		return th.ChoiceSet(elm, err)
	case "ANY":
		return th.AnySet(elm, err)
	}
	return encodeTypeErr("Mount", th.sheme)
}

func makePath(th *AsnData, path ...string) (*AsnData, error) {
	if len(path) == 0 {
		return th, nil
	}

	var out AsnElm
	if sh, err := findField(th.sheme, path[0]); err == nil {
		switch sh.Type() {
		case "SEQUENCE":
			out, err = sh.Sequence()
		case "CHOICE":
			out, err = sh.Choice()
		case "ANY":
			out, err = sh.Any()
		default:
			return nil, encodeTypeErr(path[0], sh)
		}
	} else {
		return nil, err
	}
	if err := setByType(th, out, nil); err != nil {
		return nil, err
	}
	return makePath(this(out), path[1:]...)
}

func (th *AsnData) Null(name string) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.Null()
	}
	return
}

func (th *AsnData) PathNull(path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.Null(path[len(path)-1])
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) SetNull(name string) error {
	return th.SeqField(th.Null(name))
}

func (th *AsnData) ChoiceNull(name string) error {
	return th.ChoiceSet(th.Null(name))
}

func (th *AsnData) AnyNull(name string) error {
	return th.AnySet(th.Null(name))
}

func (th *AsnData) Boolean(name string, val bool) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.Boolean(val)
	}
	return
}

func (th *AsnData) PathBoolean(val bool, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.Boolean(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddBoolean(val bool) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.Boolean(val))
	}
	return err
}

func (th *AsnData) SetBoolean(name string, val bool) error {
	return th.SeqField(th.Boolean(name, val))
}

func (th *AsnData) ChoiceBoolean(name string, val bool) error {
	return th.ChoiceSet(th.Boolean(name, val))
}

func (th *AsnData) AnyBoolean(name string, val bool) error {
	return th.AnySet(th.Boolean(name, val))
}

func (th *AsnData) Integer(name string, val int) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.Integer(val)
	}
	return
}

func (th *AsnData) PathInteger(val int, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.Integer(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddInteger(val int) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.Integer(val))
	}
	return err
}

func (th *AsnData) SetInteger(name string, val int) error {
	return th.SeqField(th.Integer(name, val))
}

func (th *AsnData) ChoiceInteger(name string, val int) error {
	return th.ChoiceSet(th.Integer(name, val))
}

func (th *AsnData) AnyInteger(name string, val int) error {
	return th.AnySet(th.Integer(name, val))
}

func (th *AsnData) Real(name string, val float64) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.Real(val)
	}
	return
}

func (th *AsnData) PathReal(val float64, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.Real(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddReal(val float64) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.Real(val))
	}
	return err
}

func (th *AsnData) SetReal(name string, val float64) error {
	return th.SeqField(th.Real(name, val))
}

func (th *AsnData) ChoiceReal(name string, val float64) error {
	return th.ChoiceSet(th.Real(name, val))
}

func (th *AsnData) AnyReal(name string, val float64) error {
	return th.AnySet(th.Real(name, val))
}

func (th *AsnData) Enumerated(name string, val string) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.Enumerated(val)
	}
	return
}

func (th *AsnData) PathEnumerated(val string, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.Enumerated(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddEnumerated(val string) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.Enumerated(val))
	}
	return err
}

func (th *AsnData) SetEnumerated(name string, val string) error {
	return th.SeqField(th.Enumerated(name, val))
}

func (th *AsnData) ChoiceEnumerated(name string, val string) error {
	return th.ChoiceSet(th.Enumerated(name, val))
}

func (th *AsnData) AnyEnumerated(name string, val string) error {
	return th.AnySet(th.Enumerated(name, val))
}

func (th *AsnData) BitString(name string, val BitStr) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.BitString(val)
	}
	return
}

func (th *AsnData) PathBitString(val BitStr, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.BitString(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddBitString(val BitStr) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.BitString(val))
	}
	return err
}

func (th *AsnData) SetBitString(name string, val BitStr) error {
	return th.SeqField(th.BitString(name, val))
}

func (th *AsnData) ChoiceBitString(name string, val BitStr) error {
	return th.ChoiceSet(th.BitString(name, val))
}

func (th *AsnData) AnyBitString(name string, val BitStr) error {
	return th.AnySet(th.BitString(name, val))
}

func (th *AsnData) UTCTime(name string, val time.Time) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.UTCTime(val)
	}
	return
}

func (th *AsnData) PathUTCTime(val time.Time, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.UTCTime(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddUTCTime(val time.Time) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.UTCTime(val))
	}
	return err
}

func (th *AsnData) SetUTCTime(name string, val time.Time) error {
	return th.SeqField(th.UTCTime(name, val))
}

func (th *AsnData) ChoiceUTCTime(name string, val time.Time) error {
	return th.ChoiceSet(th.UTCTime(name, val))
}

func (th *AsnData) AnyUTCTime(name string, val time.Time) error {
	return th.AnySet(th.UTCTime(name, val))
}

func (th *AsnData) GeneralizedTime(name string, val time.Time) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.GeneralizedTime(val)
	}
	return
}

func (th *AsnData) PathGeneralizedTime(val time.Time, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.GeneralizedTime(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddGeneralizedTime(val time.Time) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.GeneralizedTime(val))
	}
	return err
}

func (th *AsnData) SetGeneralizedTime(name string, val time.Time) error {
	return th.SeqField(th.GeneralizedTime(name, val))
}

func (th *AsnData) ChoiceGeneralizedTime(name string, val time.Time) error {
	return th.ChoiceSet(th.GeneralizedTime(name, val))
}

func (th *AsnData) AnyGeneralizedTime(name string, val time.Time) error {
	return th.AnySet(th.GeneralizedTime(name, val))
}

func (th *AsnData) ObjectIdentifier(name string, val OID) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.ObjectIdentifier(val)
	}
	return
}

func (th *AsnData) PathObjectIdentifier(val OID, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.ObjectIdentifier(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddObjectIdentifier(val OID) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.ObjectIdentifier(val))
	}
	return err
}

func (th *AsnData) SetObjectIdentifier(name string, val OID) error {
	return th.SeqField(th.ObjectIdentifier(name, val))
}

func (th *AsnData) ChoiceObjectIdentifier(name string, val OID) error {
	return th.ChoiceSet(th.ObjectIdentifier(name, val))
}

func (th *AsnData) AnyObjectIdentifier(name string, val OID) error {
	return th.AnySet(th.ObjectIdentifier(name, val))
}

func (th *AsnData) ObjectDescriptor(name string, val string) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.ObjectDescriptor(val)
	}
	return
}

func (th *AsnData) PathObjectDescriptor(val string, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.ObjectDescriptor(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddObjectDescriptor(val string) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.ObjectDescriptor(val))
	}
	return err
}

func (th *AsnData) SetObjectDescriptor(name string, val string) error {
	return th.SeqField(th.ObjectDescriptor(name, val))
}

func (th *AsnData) ChoiceObjectDescriptor(name string, val string) error {
	return th.ChoiceSet(th.ObjectDescriptor(name, val))
}

func (th *AsnData) AnyObjectDescriptor(name string, val string) error {
	return th.AnySet(th.ObjectDescriptor(name, val))
}

func (th *AsnData) NumericString(name string, val string) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.NumericString(val)
	}
	return
}

func (th *AsnData) PathNumericString(val string, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.NumericString(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddNumericString(val string) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.NumericString(val))
	}
	return err
}

func (th *AsnData) SetNumericString(name string, val string) error {
	return th.SeqField(th.NumericString(name, val))
}

func (th *AsnData) ChoiceNumericString(name string, val string) error {
	return th.ChoiceSet(th.NumericString(name, val))
}

func (th *AsnData) AnyNumericString(name string, val string) error {
	return th.AnySet(th.NumericString(name, val))
}

func (th *AsnData) PrintableString(name string, val string) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.PrintableString(val)
	}
	return
}

func (th *AsnData) PathPrintableString(val string, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.PrintableString(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddPrintableString(val string) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.PrintableString(val))
	}
	return err
}

func (th *AsnData) SetPrintableString(name string, val string) error {
	return th.SeqField(th.PrintableString(name, val))
}

func (th *AsnData) ChoicePrintableString(name string, val string) error {
	return th.ChoiceSet(th.PrintableString(name, val))
}

func (th *AsnData) AnyPrintableString(name string, val string) error {
	return th.AnySet(th.PrintableString(name, val))
}

func (th *AsnData) IA5String(name string, val string) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.IA5String(val)
	}
	return
}

func (th *AsnData) PathIA5String(val string, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.IA5String(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddIA5String(val string) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.IA5String(val))
	}
	return err
}

func (th *AsnData) SetIA5String(name string, val string) error {
	return th.SeqField(th.IA5String(name, val))
}

func (th *AsnData) ChoiceIA5String(name string, val string) error {
	return th.ChoiceSet(th.IA5String(name, val))
}

func (th *AsnData) AnyIA5String(name string, val string) error {
	return th.AnySet(th.IA5String(name, val))
}

func (th *AsnData) UTF8String(name string, val string) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.UTF8String(val)
	}
	return
}

func (th *AsnData) PathUTF8String(val string, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.UTF8String(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddUTF8String(val string) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.UTF8String(val))
	}
	return err
}

func (th *AsnData) SetUTF8String(name string, val string) error {
	return th.SeqField(th.UTF8String(name, val))
}

func (th *AsnData) ChoiceUTF8String(name string, val string) error {
	return th.ChoiceSet(th.UTF8String(name, val))
}

func (th *AsnData) AnyUTF8String(name string, val string) error {
	return th.AnySet(th.UTF8String(name, val))
}

func (th *AsnData) OctetString(name string, val []byte) (out AsnElm, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.OctetString(val)
	}
	return
}

func (th *AsnData) PathOctetString(val []byte, path ...string) error {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.OctetString(path[len(path)-1], val)
		return setByType(pth, el, err)
	}
	return err
}

func (th *AsnData) AddOctetString(val []byte) error {
	sh, err := findOf(th.sheme)
	if err == nil {
		return th.SeqItem(sh.OctetString(val))
	}
	return err
}

func (th *AsnData) SetOctetString(name string, val []byte) error {
	return th.SeqField(th.OctetString(name, val))
}

func (th *AsnData) ChoiceOctetString(name string, val []byte) error {
	return th.ChoiceSet(th.OctetString(name, val))
}

func (th *AsnData) AnyOctetString(name string, val []byte) error {
	return th.AnySet(th.OctetString(name, val))
}

func (th *AsnData) Sequence(name string) (out AsnSeq, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.Sequence()
	}
	return
}

func (th *AsnData) PathSequence(path ...string) (out AsnSeq, err error) {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.Sequence(path[len(path)-1])
		return el, setByType(pth, el, err)
	}
	return nil, err
}

func (th *AsnData) AddSequence() (out AsnSeq, err error) {
	sh, err := findOf(th.sheme)
	if err == nil {
		if out, err = sh.Sequence(); err == nil {
			err = th.SeqItem(out, nil)
		}
	}
	return
}

func (th *AsnData) SetSequence(name string) (out AsnSeq, err error) {
	if out, err = th.Sequence(name); err == nil {
		err = th.SeqField(out, nil)
	}
	return
}

func (th *AsnData) ChoiceSequence(name string) (out AsnSeq, err error) {
	if out, err = th.Sequence(name); err == nil {
		err = th.ChoiceSet(out, nil)
	}
	return
}

func (th *AsnData) AnySequence(name string) (out AsnSeq, err error) {
	if out, err = th.Sequence(name); err == nil {
		err = th.AnySet(out, nil)
	}
	return
}

func (th *AsnData) Choice(name string) (out AsnChoice, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.Choice()
	}
	return
}

func (th *AsnData) PathChoice(path ...string) (out AsnChoice, err error) {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.Choice(path[len(path)-1])
		return el, setByType(pth, el, err)
	}
	return nil, err
}

func (th *AsnData) AddChoice() (out AsnChoice, err error) {
	sh, err := findOf(th.sheme)
	if err == nil {
		if out, err = sh.Choice(); err == nil {
			err = th.SeqItem(out, nil)
		}
	}
	return
}

func (th *AsnData) SetChoice(name string) (out AsnChoice, err error) {
	if out, err = th.Choice(name); err == nil {
		err = th.SeqField(out, nil)
	}
	return
}

func (th *AsnData) ChoiceChoice(name string) (out AsnChoice, err error) {
	if out, err = th.Choice(name); err == nil {
		err = th.ChoiceSet(out, nil)
	}
	return
}

func (th *AsnData) AnyChoice(name string) (out AsnChoice, err error) {
	if out, err = th.Choice(name); err == nil {
		err = th.AnySet(out, nil)
	}
	return
}

func (th *AsnData) Any(name string) (out AsnAny, err error) {
	sh, err := findField(th.sheme, name)
	if err == nil {
		out, err = sh.Any()
	}
	return
}

func (th *AsnData) PathAny(path ...string) (out AsnAny, err error) {
	pth, err := makePath(th, path[:len(path)-1]...)
	if err == nil {
		el, err := pth.Any(path[len(path)-1])
		return el, setByType(pth, el, err)
	}
	return nil, err
}

func (th *AsnData) AddAny() (out AsnAny, err error) {
	sh, err := findOf(th.sheme)
	if err == nil {
		if out, err = sh.Any(); err == nil {
			err = th.SeqItem(out, nil)
		}
	}
	return
}

func (th *AsnData) SetAny(name string) (out AsnAny, err error) {
	if out, err = th.Any(name); err == nil {
		err = th.SeqField(out, nil)
	}
	return
}

func (th *AsnData) ChoiceAny(name string) (out AsnAny, err error) {
	if out, err = th.Any(name); err == nil {
		err = th.ChoiceSet(out, nil)
	}
	return
}
