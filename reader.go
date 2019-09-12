package asn1dynamic

import (
	"fmt"
	"io"
)

type asnReader struct {
	reader io.Reader
	buff1  []byte
	buff2  []byte
}

func NewDataReader(r io.Reader, size int) asnReader {
	return asnReader{reader: r, buff1: make([]byte, 512), buff2: make([]byte, 0, size)}
}

func (rd *asnReader) Read() (AsnDec, error) {
	dec := NewDecoder()

	ln, err := rd.reader.Read(rd.buff1)
	debugPrint("TCP read: %d tail: %d", ln, len(rd.buff2))
	if err != nil || ln == 0 {
		return nil, err
	}
	rd.buff2 = append(rd.buff2, rd.buff1[:ln]...)

	tail, ok, err := dec.Parse(rd.buff2, 0)
	if err != nil {
		rd.buff2 = rd.buff2[0:0]
		return nil, fmt.Errorf("TCP Decode: %s", out.err.Error())
	}

	rd.buff2 = append(rd.buff2[0:0], tail...)

	if ok {
		return dec, nil
	}

	return nil, nil
}
