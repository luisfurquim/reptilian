package reptilian

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	//   "Goose"
)

func init() {
	DeflateHeader = [10]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 1}
	reBoolTrue = regexp.MustCompile("^true(?:[^a-zA-Z0-9]|$)")
	reBoolFalse = regexp.MustCompile("^false(?:[^a-zA-Z0-9]|$)")
	reNumber = regexp.MustCompile("^([+-]{0,1}(?:(?:[0-9]+(?:\\.[0-9]+){0,1})|(?:\\.[0-9]+)))(?:[^0-9]|$)")
	reUInt = regexp.MustCompile("^([0-9]+)(?:[^0-9]|$)")
	reName = regexp.MustCompile("^/([\x21\x22\x24\x26\x27\x2a-\x2e\x30-\x3b\x3d\x3f-\x5a\x5c\x5e-\x7a\x7c\x7e]+)")
	reIdent = regexp.MustCompile("^([\x21\x22\x24\x26\x27\x2a-\x2e\x30-\x3b\x3d\x3f-\x5a\x5c\x5e-\x7a\x7c\x7e]+)")
	//   reString       = regexp.MustCompile("^\\(([^\\)]*)\\)")
	reDicStart = regexp.MustCompile("^<<")
	reDicEnd = regexp.MustCompile("^>>")
	reObj = regexp.MustCompile("^obj(?:[^a-zA-Z0-9]|$)")
	reEndObj = regexp.MustCompile("^endobj(?:[^a-zA-Z0-9]|$)")
	reStream = regexp.MustCompile("^stream(?:[^a-zA-Z0-9]|$)")
	reEndStream = regexp.MustCompile("^endstream(?:[^a-zA-Z0-9]|$)")
	reBinEndStream = regexp.MustCompile("(?m)^(.*?)([\x0d\x0a]+)endstream(?:[^a-zA-Z0-9]|$)")
	reXref = regexp.MustCompile("^xref(?:[^a-zA-Z0-9]|$)")
	reStartXref = regexp.MustCompile("^startxref(?:[^a-zA-Z0-9]|$)")

	//2500280029003c003e005b005d007b007d002f0023
	//21-7e
}

var DeflateHeader [10]byte

func (str *StreamReader) Read(p []byte) (n int, err error) {
	Goose.Logf(6, "A) n:%d, err:%s, len(p):%d", n, err, len(p))
	n = 0
	if (int64(n) + str.Off - 10) >= int64(len(str.Buf)) {
		err = io.EOF
		Goose.Logf(6, "B) n:%d, err:%s, len(p):%d", n, err, len(p))
		return
	}
	err = nil

	Goose.Logf(6, "C) n:%d, err:%s, len(p):%d", n, err, len(p))
	for ; int64(n) < (10 - str.Off); n++ {
		p[n] = DeflateHeader[str.Off+int64(n)]
	}

	Goose.Logf(6, "D) n:%d, err:%s, len(p):%d", n, err, len(p))
	for ; (n < len(p)) && ((int64(n) + str.Off - 10) < int64(len(str.Buf))); n++ {
		p[n] = str.Buf[str.Off+int64(n)-10]
	}
	Goose.Logf(6, "E) n:%d, err:%s, len(p):%d", n, err, len(p))
	str.Off += int64(n)
	Goose.Logf(6, "F) n:%d, err:%s, len(p):%d", n, err, len(p))
	return
}

func NewStreamReaderRFC1950(buf []byte) *StreamReader {
	return &StreamReader{Buf: buf}
}

func NewStreamReaderRFC1951(buf []byte) *StreamReader {
	return &StreamReader{Buf: buf[2:]}
}

var Skip StateHnd = func(buf *Buf) Token {
	var t BlankT
	Goose.Logf(3, "Skip")
	buf.Off++
	t = BlankT(true)
	return &t
}

var reBoolTrue *regexp.Regexp
var ChkBoolTrue StateHnd = func(buf *Buf) Token {
	var t BooleanT
	Goose.Logf(3, "ChkBoolTrue")
	if reBoolTrue.Match(buf.Raw[buf.Off:]) {
		buf.Off += 4
		buf.State[buf.StateLvl] = StStd
		t = BooleanT(true)
		return &t
	}
	return nil
}

var reBoolFalse *regexp.Regexp
var ChkBoolFalse StateHnd = func(buf *Buf) Token {
	var t BooleanT
	Goose.Logf(3, "ChkBoolFalse")
	if reBoolFalse.Match(buf.Raw[buf.Off:]) {
		buf.Off += 5
		buf.State[buf.StateLvl] = StStd
		t = BooleanT(false)
		return &t
	}
	return nil
}

var reNumber *regexp.Regexp
var ChkNumber StateHnd = func(buf *Buf) Token {
	var r RealT
	var f float32
	var n IntT
	var i int
	var m [][]byte

	Goose.Logf(3, "ChkNumber")
	m = reNumber.FindSubmatch(buf.Raw[buf.Off:])
	if len(m) == 2 {
		buf.Off += int64(len(m[1]))
		buf.State[buf.StateLvl] = StStd
		for j := 0; j < len(m[1]); j++ {
			if m[1][j] == '.' {
				fmt.Sscanf(string(m[1]), "%f", &f)
				r = RealT(f)
				return &r
			}
		}
		fmt.Sscanf(string(m[1]), "%d", &i)
		n = IntT(i)
		return &n
	}
	return nil
}

var reName *regexp.Regexp
var ChkName StateHnd = func(buf *Buf) Token {
	var t NameT
	var m [][]byte
	Goose.Logf(3, "ChkName")
	m = reName.FindSubmatch(buf.Raw[buf.Off:])
	if len(m) == 2 {
		buf.Off += int64(len(m[1]) + 1)
		buf.State[buf.StateLvl] = StStd
		t = NameT(m[1])
		return &t
	}
	return nil
}

var ChkString StateHnd = func(buf *Buf) Token {
	var t StringT
	var sz int64
	var hexsz int
	var tgt int64
	var esc bool
	var oct byte
	var hex byte
	var nmb byte
	var b byte
	var threshold int64

	Goose.Logf(3, "ChkString")

	sz = 1
	tgt = 1
	esc = false
	oct = 0
	hex = 0
	if len(buf.Raw) > 60 {
		threshold = 60
	} else {
		threshold = int64(len(buf.Raw))
	}

	Goose.Logf(3, "Original at %d (%o): %s", buf.Off, buf.Off, buf.Raw[buf.Off:buf.Off+threshold])

	for (buf.Off + sz) < int64(len(buf.Raw)) {
		b = buf.Raw[buf.Off+sz]
		if oct > 0 {
			if oct <= 3 {
				oct++
				if (b >= '0') && (b <= '7') {
					nmb = (nmb << 3) | (b - '0')
					sz++
					continue
				}
				Goose.Logf(1, "Invalid octal number at %d", buf.Off+sz-int64(oct))
				return nil
			}
			buf.Raw[buf.Off+tgt] = nmb
			tgt++
			nmb = 0
			if (b >= '0') && (b <= '7') {
				oct = 1
			} else {
				oct = 0
			}
			continue
		}

		if hex > 0 {
			if hex <= 2 {
				hex++
				if (b >= '0') && (b <= '9') {
					nmb = (nmb << 4) | (b - '0')
				} else if (b >= 'a') && (b <= 'f') {
					nmb = (nmb << 4) | (b - 'a')
				} else if (b >= 'A') && (b <= 'F') {
					nmb = (nmb << 4) | (b - 'A')
				} else {
					if hexsz > 0 {
						Goose.Logf(1, "Invalid hexa number at %d", buf.Off+sz-int64(hex))
						return nil
					}
					hex = 0
				}
				sz++
				continue
			}
			buf.Raw[buf.Off+tgt] = nmb
			tgt++
			nmb = 0
			if ((b >= '0') && (b <= '9')) || ((b >= 'a') && (b <= 'f')) || ((b >= 'A') && (b <= 'F')) {
				hex = 1
			} else if b == '>' {
				hex = 0
				sz++
			} else {
				if hexsz > 0 {
					Goose.Logf(1, "Invalid.1 hexa number at %d", buf.Off+sz-int64(hex))
					return nil
				}
				hex = 0
			}
			continue
		}

		if esc {
			if b == '0' {
				oct = 1
			} else if (b == '\\') || (b == '<') || (b == '(') || (b == ')') {
				buf.Raw[buf.Off+tgt] = b
				tgt++
			} else if (b == '\r') && ((buf.Off + sz + 1) < int64(len(buf.Raw))) && (buf.Raw[buf.Off+sz+1] == '\n') {
				sz++
			}
			esc = false
		} else if b == '<' {
			hex = 1
			hexsz = 0
		} else if b == '\\' {
			esc = true
		} else if b == ')' {
			Goose.Logf(3, "Final at %d (%o): %s", buf.Off, buf.Off, buf.Raw[buf.Off+1:buf.Off+tgt])
			t = StringT(buf.Raw[buf.Off+1 : buf.Off+tgt])
			buf.Off += sz + 1
			buf.State[buf.StateLvl] = StStd
			return &t
		} else {
			if tgt < sz { // avoid uneeded cpu cache dirty
				buf.Raw[buf.Off+tgt] = b
			}
			tgt++
		}
		sz++
	}
	return nil
}

var ChkArrayStart StateHnd = func(buf *Buf) Token {
	var t BlankT
	var a ArrayT
	Goose.Logf(3, "ChkArrayStart")
	a = make(ArrayT, 0, 16)
	buf.Add(&a)
	buf.Off++
	buf.State[buf.StateLvl] = StStd
	buf.StateLvl++
	if len(buf.State) <= buf.StateLvl {
		buf.State = append(buf.State, StStd)
	} else {
		buf.State[buf.StateLvl] = StStd
	}
	if len(buf.DataStack) <= buf.StateLvl {
		buf.DataStack = append(buf.DataStack, &a)
	} else {
		buf.DataStack[buf.StateLvl] = &a
	}
	t = BlankT(true)
	return &t
}

var ChkArrayEnd StateHnd = func(buf *Buf) Token {
	var t BlankT
	Goose.Logf(3, "ChkArrayEnd")
	buf.Off++
	if !buf.Pop() {
		return nil
	}
	t = BlankT(true)
	return &t
}

var reDicStart *regexp.Regexp
var ChkDicStart StateHnd = func(buf *Buf) Token {
	var t BlankT
	var d *DicT

	Goose.Logf(3, "ChkDicStart")
	if reDicStart.Match(buf.Raw[buf.Off:]) {
		d = &DicT{Dic: make(map[string]Token, 16)}
		buf.Add(d)
		Goose.Logf(5, "DicStart -> lvl: %d, datastack: %s", buf.StateLvl, buf.DataStack)
		buf.Off += int64(2)
		buf.State[buf.StateLvl] = StStd
		buf.Push(d, StStd)
		Goose.Logf(5, "DicStart -> lvl: %d, datastack: %s", buf.StateLvl, buf.DataStack)
		t = BlankT(true)
		return &t
	}
	return ChkHexId(buf)
}

var reDicEnd *regexp.Regexp
var ChkDicEnd StateHnd = func(buf *Buf) Token {
	var t BlankT
	Goose.Logf(3, "ChkDicEnd")
	if reDicEnd.Match(buf.Raw[buf.Off:]) {
		buf.Off += int64(2)
		if len(buf.DataStack[buf.StateLvl].(*DicT).NewName) > 0 {
			return nil
		}
		if !buf.Pop() {
			return nil
		}
		Goose.Logf(5, "lvl: %d, datastack: %s,   data:%s", buf.StateLvl, buf.DataStack, buf.Data)
		t = BlankT(true)
		return &t
	}
	return nil
}

var reObj *regexp.Regexp
var ChkObj StateHnd = func(buf *Buf) Token {
	var o *ObjT
	var b BlankT
	Goose.Logf(3, "ChkObj")
	if reObj.Match(buf.Raw[buf.Off:]) {
		o = &ObjT{}
		buf.Add(o)
		buf.Off += int64(3)
		buf.State[buf.StateLvl] = StStd
		buf.Push(o, StObj)
		b = BlankT(true)
		return &b
	}
	return nil
}

var reEndObj *regexp.Regexp
var ChkEndObj StateHnd = func(buf *Buf) Token {
	var t BlankT
	var a *ArrayT
	var n, v int

	Goose.Logf(3, "ChkEndObj")
	if reEndObj.Match(buf.Raw[buf.Off:]) {
		buf.Off += int64(6)
		if !buf.Pop() {
			return nil
		}

		switch buf.DataStack[buf.StateLvl].(type) {
		case *ArrayT:
			a = buf.DataStack[buf.StateLvl].(*ArrayT)
			if len(*a) >= 2 {
				switch (*a)[len(*a)-3].(type) {
				case *IntT:
					n = int(*((*a)[len(*a)-3].(*IntT)))
					switch (*a)[len(*a)-2].(type) {
					case *IntT:
						v = int(*((*a)[len(*a)-2].(*IntT)))
						if len(buf.Obj) <= n {
							buf.Obj = append(buf.Obj, make([]*ObjT, n-len(buf.Obj)+1, n-len(buf.Obj)+1)...)
						}
						if buf.Obj[n] != nil {
							Goose.Logf(1, "REPLACING OBJ %d, lvl: %d", n, buf.StateLvl)
							Goose.Logf(5, "REPLACING OBJ -> datastack: %s,   data:%s", buf.DataStack, buf.Data)
						}
						buf.Obj[n] = (*a)[len(*a)-1].(*ObjT)
						buf.Obj[n].Version = v
						buf.Obj[n].Valid = true
						Goose.Logf(6, "STORING OBJ %d, lvl: %d, datastack top: %s", n, buf.StateLvl, buf.DataStack[buf.StateLvl])
					default:
						Goose.Logf(1, "Malformed OBJ %d, lvl: %d, datastack top: %s", n, buf.StateLvl, buf.DataStack[buf.StateLvl])
						return nil
					}
				default:
					Goose.Logf(1, "Malformed OBJ lvl: %d, datastack top: %s", n, buf.StateLvl, buf.DataStack[buf.StateLvl])
					return nil
				}
			} else {
				return nil
			}
		}

		Goose.Logf(5, "lvl: %d, datastack: %s,   data:%s", buf.StateLvl, buf.DataStack, buf.Data)
		t = BlankT(true)
		return &t
	}
	return nil
}

var reUInt *regexp.Regexp
var ChkIndRef StateHnd = func(buf *Buf) Token {
	var i IndObjT
	var n int32
	var b BlankT
	var m [][]byte
	Goose.Logf(3, "ChkIndRef")
	m = reUInt.FindSubmatch(buf.Raw[buf.Off:])
	if len(m) == 2 {
		fmt.Sscanf(string(m[1]), "%d", &n)
		buf.Off += int64(len(m[1]))
		buf.State[buf.StateLvl] = StObjEnd
		i = IndObjT(n)
		buf.DataStack[buf.StateLvl] = &i
		b = BlankT(true)
		return &b
	}
	return nil
}

var ChkDicHeader StateHnd = func(buf *Buf) Token {
	var t BlankT

	Goose.Logf(3, "ChkDicHeader")
	if reDicStart.Match(buf.Raw[buf.Off:]) {
		buf.DataStack[buf.StateLvl].(*ObjT).Header = DicT{Dic: make(map[string]Token, 16)}
		Goose.Logf(5, "DicHeader -> lvl: %d, datastack: %s", buf.StateLvl, buf.DataStack)
		buf.Off += int64(2)
		buf.State[buf.StateLvl] = StObjHdr
		buf.Push(&buf.DataStack[buf.StateLvl].(*ObjT).Header, StStd)
		Goose.Logf(5, "DicHeader -> lvl: %d, datastack: %s", buf.StateLvl, buf.DataStack)
		t = BlankT(true)
		return &t
	}
	return nil
}

var reStream *regexp.Regexp
var reBinEndStream *regexp.Regexp
var ChkStream StateHnd = func(buf *Buf) Token {
	var b BlankT
	var s StringT
	var stream []byte
	var txt []byte
	var sz int64
	var m [][]byte
	var state int
	var native bool

	const (
		ststd = iota
		stcrlf
	)
	Goose.Logf(3, "ChkStream")
	if reStream.Match(buf.Raw[buf.Off:]) {
		b = BlankT(true)
		buf.DataStack[buf.StateLvl].(*ObjT).Stream = ArrayT{}
		Goose.Logf(5, "Stream -> lvl: %d, datastack: %s", buf.StateLvl, buf.DataStack)
		buf.Off += int64(6)
		buf.State[buf.StateLvl] = StObjEnd
		native = true
		if _, ok := buf.DataStack[buf.StateLvl].(*ObjT).Header.Dic["Filter"]; ok {
			native = false
		} else if _, ok := buf.DataStack[buf.StateLvl].(*ObjT).Header.Dic["Type"]; ok {
			native = false
		}
		if !native {
			Goose.Logf(4, "Has filter %s", buf.Raw[buf.Off:buf.Off+271])
			m = reCrlf.FindSubmatch(buf.Raw[buf.Off:])
			if len(m) == 2 {
				Goose.Logf(3, "skip...CRLF")
				buf.Off += int64(len(m[1]))
			}

			state = ststd
		Found:
			for sz = 0; true; sz++ {
				switch state {
				case ststd:
					if (buf.Raw[buf.Off+sz] == '\r') || (buf.Raw[buf.Off+sz] == '\n') {
						Goose.Logf(4, "FOUND CRLF, sz=%d, %s", sz, buf.Raw[buf.Off+sz:buf.Off+sz+10])
						state = stcrlf
					}
				case stcrlf:
					Goose.Logf(5, "len:%d,  off:%d,   sz:%d,  %s", len(buf.Raw), buf.Off, sz, buf.Raw[buf.Off+sz:buf.Off+sz+10])
					if bytes.Equal(buf.Raw[buf.Off+sz:buf.Off+sz+9], []byte{'e', 'n', 'd', 's', 't', 'r', 'e', 'a', 'm'}) {
						break Found
					} else if (buf.Raw[buf.Off+sz] != '\r') && (buf.Raw[buf.Off+sz] != '\n') {
						state = ststd
					}
				}
			}
			if (buf.Raw[buf.Off+sz-2] == '\r') && (buf.Raw[buf.Off+sz-1] == '\n') {
				stream = buf.Raw[buf.Off : buf.Off+sz-2]
			} else {
				stream = buf.Raw[buf.Off : buf.Off+sz-1]
			}

			if filter, ok := buf.DataStack[buf.StateLvl].(*ObjT).Header.Dic["Filter"]; ok {
				r, err := gzip.NewReader(NewStreamReaderRFC1951(stream))
				if err != nil {
					r, err = gzip.NewReader(NewStreamReaderRFC1950(stream))
					if err != nil {
						f, err := os.OpenFile("./comp1", os.O_CREATE|os.O_RDWR, 0666)
						if err != nil {
							log.Fatalf("open %s", err)
						}
						io.Copy(f, NewStreamReaderRFC1951(stream))

						f, err = os.OpenFile("./comp0", os.O_CREATE|os.O_RDWR, 0666)
						if err != nil {
							log.Fatalf("open %s", err)
						}
						io.Copy(f, NewStreamReaderRFC1950(stream))
						Goose.Logf(1, "%s.1: error %s, %s", filter, err, buf.Raw[buf.Off-1:buf.Off+sz])
						os.Exit(1)
					}
				}
				txt, err = ioutil.ReadAll(r)
				if (err != nil) && (err != io.EOF) && (err != io.ErrUnexpectedEOF) {
					ioutil.WriteFile("./comprimido", []byte(s), 0777)
					Goose.Logf(1, "%s.2: error %s, %s", filter, err, buf.Raw[buf.Off-1:buf.Off+sz])
					os.Exit(1)
				}
				isXObject := false
				if Type, ok := buf.DataStack[buf.StateLvl].(*ObjT).Header.Dic["Type"]; ok {
					switch Type.(type) {
					case *NameT:
						if string(*(Type.(*NameT))) == "XObject" {
							isXObject = true
						} else {
							Goose.Logf(1, "data: %s", Type)
						}
					}
				}
				if !isXObject {
					delete(buf.DataStack[buf.StateLvl].(*ObjT).Header.Dic, "Length")
					delete(buf.DataStack[buf.StateLvl].(*ObjT).Header.Dic, "Filter")
					Goose.Logf(2, "GZIP METADATA: %s", buf.DataStack[buf.StateLvl].(*ObjT).Header)
					Goose.Logf(2, "GZIP: %s", txt)
				} else {
					Goose.Logf(2, "XObject METADATA: %s", buf.DataStack[buf.StateLvl].(*ObjT).Header)
					Goose.Logf(2, "XObject Subtype: %s", buf.DataStack[buf.StateLvl].(*ObjT).Header.Dic["Subtype"])
					//               Goose.Logf(1,"GZIP: %s",txt)
				}
				s = StringT(txt)
			} else {
				s = StringT(stream)
			}
			buf.DataStack[buf.StateLvl].(*ObjT).Stream = ArrayT{&s}
			buf.Off += sz + 9
			return &b
		}
		buf.Push(&buf.DataStack[buf.StateLvl].(*ObjT).Stream, StStd)
		Goose.Logf(5, "Stream -> lvl: %d, datastack: %s", buf.StateLvl, buf.DataStack)
		return &b
	}
	return nil
}

var reEndStream *regexp.Regexp
var ChkEndStream StateHnd = func(buf *Buf) Token {
	var t BlankT
	Goose.Logf(3, "ChkEndStream")
	if reEndStream.Match(buf.Raw[buf.Off:]) {
		buf.Off += int64(9)
		if !buf.Pop() {
			return nil
		}
		Goose.Logf(5, "lvl: %d, datastack: %s,   data:%s", buf.StateLvl, buf.DataStack, buf.Data)
		t = BlankT(true)
		return &t
	}
	return nil
}

var reIdent *regexp.Regexp
var ChkIdent StateHnd = func(buf *Buf) Token {
	var t IdentT
	var m [][]byte
	Goose.Logf(3, "ChkIdent")
	m = reIdent.FindSubmatch(buf.Raw[buf.Off:])
	if len(m) == 2 {
		buf.Off += int64(len(m[1]))
		buf.State[buf.StateLvl] = StStd
		t = IdentT(m[1])
		return &t
	}
	return nil
}

func SkipBlank(section string, buf *Buf) bool {
	if (buf.Raw[buf.Off] != '\r') && (buf.Raw[buf.Off] != '\n') && (buf.Raw[buf.Off] != ' ') && (buf.Raw[buf.Off] != '\t') {
		Goose.Logf(1, "Malformed %s: expected \\s*(CR|CRLF|LF)", section)
		return true
	}
	for ; (buf.Off < int64(len(buf.Raw))) && ((buf.Raw[buf.Off] == '\r') || (buf.Raw[buf.Off] == '\n') || (buf.Raw[buf.Off] == ' ') || (buf.Raw[buf.Off] == '\t')); buf.Off++ {
	}
	return false
}

var reXref *regexp.Regexp
var ChkXref StateHnd = func(buf *Buf) Token {
	var m [][]byte
	var ndx, qt int
	var t BlankT

	Goose.Logf(3, "ChkXref")
	m = reXref.FindSubmatch(buf.Raw[buf.Off:])
	if len(m) == 1 {
		buf.Off += int64(len(m[0]))
		buf.State[buf.StateLvl] = StTrailer
		for ; (buf.Raw[buf.Off] == '\r') || (buf.Raw[buf.Off] == '\n'); buf.Off++ {
		}

		//      Goose.Logf(1,"len(m[0]): %d,   m[0]=%s,  nextraw:%x %x %x",len(m[0]),m[0],buf.Raw[buf.Off],buf.Raw[buf.Off+1],buf.Raw[buf.Off+2])
		//      os.Exit(1)

		for (buf.Raw[buf.Off] >= '0') && (buf.Raw[buf.Off] <= '9') {
			for ndx = 0; buf.Off < int64(len(buf.Raw)); buf.Off++ {
				if (buf.Raw[buf.Off] >= '0') && (buf.Raw[buf.Off] <= '9') {
					ndx = (ndx * 10) + int(buf.Raw[buf.Off]-'0')
				} else {
					break
				}
			}

			for ; buf.Off < int64(len(buf.Raw)); buf.Off++ {
				if (buf.Raw[buf.Off] != ' ') && (buf.Raw[buf.Off] != '\t') {
					break
				}
			}

			if (buf.Raw[buf.Off] < '0') || (buf.Raw[buf.Off] > '9') {
				Goose.Logf(1, "Malformed xref: object list size not found")
				return nil
			}

			for qt = 0; buf.Off < int64(len(buf.Raw)); buf.Off++ {
				if (buf.Raw[buf.Off] >= '0') && (buf.Raw[buf.Off] <= '9') {
					qt = (qt * 10) + int(buf.Raw[buf.Off]-'0')
				} else {
					break
				}
			}

			if qt < 1 {
				Goose.Logf(1, "Malformed xref: expected |object list|>0")
				return nil
			}

			if ndx > len(buf.Obj) {
				Goose.Logf(1, "Malformed xref: |object list| < object index")
				return nil
			}

			if SkipBlank("xref", buf) {
				return nil
			}

			Goose.Logf(1, "xref obj ndx: %d,   qt:%d", ndx, qt)
			for i := 0; i < qt; i++ {
				Goose.Logf(3, "xref obj: %d %c", ndx, buf.Raw[buf.Off+17])
				if buf.Raw[buf.Off+17] == 'f' {
					if buf.Obj[ndx] != nil {
						(*buf.Obj[ndx]).Valid = false
						Goose.Logf(3, "xref obj: %d invalidated", ndx)
					}
				} else if buf.Raw[buf.Off+17] != 'n' {
					Goose.Logf(1, "Malformed xref: expected f|n, got: %s", buf.Raw[buf.Off-5:buf.Off+20])
					return nil
				}
				buf.Off += 20
				ndx++
			}
		}

		if string(buf.Raw[buf.Off:buf.Off+7]) != "trailer" {
			Goose.Logf(1, "Expected trailer, got: %s", buf.Raw[buf.Off-5:buf.Off+20])
		}
		buf.Off += 7

		if SkipBlank("trailer", buf) {
			return nil
		}

		// retornar mas no estado trailer

		Goose.Logf(3, "End xref: %s", buf.Raw[buf.Off-5:buf.Off+20])
		//os.Exit(1)
		t = BlankT(true)
		return &t
	}
	return nil
}

var ChkTrailerStart StateHnd = func(buf *Buf) Token {
	var t BlankT

	Goose.Logf(3, "ChkTrailerStart")
	if reDicStart.Match(buf.Raw[buf.Off:]) {
		buf.Trailer = &DicT{Dic: make(map[string]Token, 16)}
		Goose.Logf(3, "TrailerStart -> lvl: %d, datastack: %s", buf.StateLvl, buf.DataStack)
		buf.Off += int64(2)
		buf.State[buf.StateLvl] = StStartXref
		buf.Push(buf.Trailer, StStd)
		Goose.Logf(3, "TrailerStart -> lvl: %d, datastack: %s", buf.StateLvl, buf.DataStack)
		t = BlankT(true)
		return &t
	}
	return nil
}

var ChkHexId StateHnd = func(buf *Buf) Token {
	var h HexIdT
	var start int64

	Goose.Logf(3, "ChkHexId")
	buf.Off++

	if (buf.Raw[buf.Off] < '0') ||
		((buf.Raw[buf.Off] > '9') && (buf.Raw[buf.Off] < 'A')) ||
		((buf.Raw[buf.Off] > 'F') && (buf.Raw[buf.Off] < 'a')) ||
		(buf.Raw[buf.Off] > 'f') {
		Goose.Logf(1, "Malformed HexId, expecting hexa-decimal algarism, got: %s", buf.Raw[buf.Off-2:buf.Off+2])
		return nil
	}

	start = buf.Off
	for ; (buf.Off < int64(len(buf.Raw))) && (((buf.Raw[buf.Off] >= '0') && (buf.Raw[buf.Off] <= '9')) ||
		((buf.Raw[buf.Off] >= 'A') && (buf.Raw[buf.Off] <= 'F')) ||
		((buf.Raw[buf.Off] >= 'a') && (buf.Raw[buf.Off] <= 'f'))); buf.Off++ {
	}
	if buf.Raw[buf.Off] != '>' {
		Goose.Logf(1, "Malformed HexId: %s", buf.Raw[buf.Off-2:buf.Off+2])
		return nil
	}
	h = HexIdT(buf.Raw[start:buf.Off])
	buf.Off++
	return &h
}

var reStartXref *regexp.Regexp
var ChkStartXref StateHnd = func(buf *Buf) Token {
	var t BlankT

	Goose.Logf(3, "ChkStartXref")
	if reStartXref.Match(buf.Raw[buf.Off:]) {
		Goose.Logf(3, "StartXref -> lvl: %d, datastack: %s", buf.StateLvl, buf.DataStack)
		buf.Off += int64(9)
		buf.State[buf.StateLvl] = StStd

		if SkipBlank("startxref", buf) {
			return nil
		}

		n := ChkNumber(buf)
		switch n.(type) {
		case *IntT:
		case *RealT:
			Goose.Logf(1, "malformed StartXref, expecting integer number, got real: %f", float32(*n.(*RealT)))
			return nil
		default:
			Goose.Logf(1, "malformed StartXref, expecting integer number, got: %s", buf.Raw[buf.Off-2:buf.Off+2])
			return nil
		}

		if SkipBlank("startxref", buf) {
			return nil
		}

		if string(buf.Raw[buf.Off:buf.Off+5]) != "%%EOF" {
			Goose.Logf(1, "malformed StartXref, expecting %%%%EOF, got: %s", buf.Raw[buf.Off-2:buf.Off+2])
			return nil
		}
		buf.Off += 5

		if SkipBlank("startxref", buf) {
			return nil
		}

		Goose.Logf(3, "StartXref -> lvl: %d, datastack: %s", buf.StateLvl, buf.DataStack)
		t = BlankT(true)
		return &t
	}
	return nil
}
