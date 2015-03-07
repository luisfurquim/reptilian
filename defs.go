package reptilian

import (
	"fmt"
	"github.com/luisfurquim/goose"
	"io"
)

type Buf struct {
	Raw       []byte
	Off       int64
	State     []int
	StateLvl  int
	Data      ArrayT
	DataStack ArrayT
	Obj       []*ObjT
	Trailer   *DicT
}

type Token interface {
	TkPrint() string
}

type Doc struct {
	Hnd     io.Reader
	Version string
	Buf     Buf
	Index   map[string][]*ObjT
}

type StateHnd func(buf *Buf) Token

var sm []map[byte]StateHnd

type BlankT bool
type BooleanT bool
type NumberT []byte
type IntT int32
type RealT float32
type NameT []byte
type IdentT []byte
type HexIdT []byte
type StringT []byte
type ArrayT []Token
type DicT struct {
	NewName  string
	LastName string
	Dic      map[string]Token
}

//type StreamT          []byte//killit?
type IndObjT int32
type ObjT struct {
	Version int
	Valid   bool
	Header  DicT
	Stream  ArrayT
}

type StreamReader struct {
	Buf []byte
	Off int64
}

func (b BlankT) TkPrint() string {
	return "_"
}
func (b BlankT) String() string {
	return b.TkPrint()
}

func (b BooleanT) TkPrint() string {
	if b {
		return "true "
	}
	return "false "
}
func (b BooleanT) String() string {
	return b.TkPrint()
}

func (n NumberT) TkPrint() string {
	return string(n) + " "
}
func (n NumberT) String() string {
	return n.TkPrint()
}

func (n IntT) TkPrint() string {
	return fmt.Sprintf("%d ", int32(n))
}
func (n IntT) String() string {
	return n.TkPrint()
}

func (n RealT) TkPrint() string {
	return fmt.Sprintf("%f ", float32(n))
}
func (n RealT) String() string {
	return n.TkPrint()
}

func (i IdentT) TkPrint() string {
	return string(i) + " "
}
func (i IdentT) String() string {
	return i.TkPrint()
}

func (h HexIdT) TkPrint() string {
	return "<" + string(h) + "> "
}
func (h HexIdT) String() string {
	return h.TkPrint()
}

func (n NameT) TkPrint() string {
	return "/" + string(n) + " "
}
func (n NameT) String() string {
	return n.TkPrint()
}

func (s StringT) TkPrint() string {
	var ss string
	ss = string(s)
	if len(ss) > 5 {
		ss = ss[:5]
	}
	return ss + " "
}
func (s StringT) String() string {
	return s.TkPrint()
}

func (a ArrayT) TkPrint() string {
	var s string
	s = "[ "
	for _, val := range a {
		s += val.TkPrint()
	}

	return s + "] "
}
func (a ArrayT) String() string {
	return a.TkPrint()
}

func (d DicT) TkPrint() string {
	var s, nn string
	s = "< "
	nn = string(d.NewName)
	if len(nn) > 0 {
		s += "PENDING: " + nn + " "
	}

	for key, val := range d.Dic {
		s += "/" + key + ":" + val.TkPrint()
	}

	return s + "> "
}
func (d DicT) String() string {
	return d.TkPrint()
}

/*
func (s StreamT) TkPrint() string {
   return fmt.Sprintf("%q", s)
}
*/

func (o ObjT) TkPrint() string {
	if o.Valid {
		return fmt.Sprintf("{(%d) %s %s} ", o.Version, o.Header, o.Stream)
	}
	return fmt.Sprintf("*** INVALID *** {(%d) %s %s} ", o.Version, o.Header, o.Stream)
}
func (o ObjT) String() string {
	return o.TkPrint()
}

func (o IndObjT) TkPrint() string {
	return fmt.Sprintf("{%d} ", int32(o))
}
func (o IndObjT) String() string {
	return o.TkPrint()
}

func (buf *Buf) Push(t Token, stat int) {
	buf.StateLvl++
	if len(buf.State) <= buf.StateLvl {
		buf.State = append(buf.State, stat)
	} else {
		buf.State[buf.StateLvl] = stat
	}
	if len(buf.DataStack) <= buf.StateLvl {
		buf.DataStack = append(buf.DataStack, t)
	} else {
		buf.DataStack[buf.StateLvl] = t
	}
}

func (buf *Buf) Pop() bool {
	if buf.StateLvl <= 0 {
		return false
	}
	buf.StateLvl--
	return true
}

const (
	StStd = iota
	StObj
	StObjHdr
	StObjStream
	StObjEnd
	StStream
	StTrailer
	StStartXref
)

var Goose goose.Alert

func init() {
	sm = []map[byte]StateHnd{
		map[byte]StateHnd{ // StStd
			' ':  Skip,
			'\t': Skip,
			'\r': Skip,
			'\n': Skip,
			't':  ChkBoolTrue,
			'f':  ChkBoolFalse,
			'+':  ChkNumber,
			'-':  ChkNumber,
			'.':  ChkNumber,
			'0':  ChkNumber,
			'1':  ChkNumber,
			'2':  ChkNumber,
			'3':  ChkNumber,
			'4':  ChkNumber,
			'5':  ChkNumber,
			'6':  ChkNumber,
			'7':  ChkNumber,
			'8':  ChkNumber,
			'9':  ChkNumber,
			'R':  ChkIdent,
			'/':  ChkName,
			'(':  ChkString,
			'o':  ChkObj,
			'e':  ChkEndObj,
			'<':  ChkDicStart,
			'>':  ChkDicEnd,
			'[':  ChkArrayStart,
			']':  ChkArrayEnd,
			'x':  ChkXref,
		},

		map[byte]StateHnd{ // StObj
			' ':  Skip,
			'\t': Skip,
			'\r': Skip,
			'\n': Skip,
			'<':  ChkDicHeader,
			'0':  ChkIndRef,
			'1':  ChkIndRef,
			'2':  ChkIndRef,
			'3':  ChkIndRef,
			'4':  ChkIndRef,
			'5':  ChkIndRef,
			'6':  ChkIndRef,
			'7':  ChkIndRef,
			'8':  ChkIndRef,
			'9':  ChkIndRef,
		},

		map[byte]StateHnd{ // StObjHdr
			' ':  Skip,
			'\t': Skip,
			'\r': Skip,
			'\n': Skip,
			's':  ChkStream,
			'e':  ChkEndObj,
		},

		map[byte]StateHnd{ // StObjStream
			' ':  Skip,
			'\t': Skip,
			'\r': Skip,
			'\n': Skip,
			's':  ChkStream,
			'0':  ChkNumber,
			'1':  ChkNumber,
			'2':  ChkNumber,
			'3':  ChkNumber,
			'4':  ChkNumber,
			'5':  ChkNumber,
			'6':  ChkNumber,
			'7':  ChkNumber,
			'8':  ChkNumber,
			'9':  ChkNumber,
			'/':  ChkName,
			'(':  ChkString,
			'e':  ChkEndStream,
		},

		map[byte]StateHnd{ // StObjEnd
			' ':  Skip,
			'\t': Skip,
			'\r': Skip,
			'\n': Skip,
			'e':  ChkEndObj,
		},

		map[byte]StateHnd{ // StStream
			' ':  Skip,
			'\t': Skip,
			'\r': Skip,
			'\n': Skip,
			'<':  ChkDicStart,
			'0':  ChkNumber,
			'1':  ChkNumber,
			'2':  ChkNumber,
			'3':  ChkNumber,
			'4':  ChkNumber,
			'5':  ChkNumber,
			'6':  ChkNumber,
			'7':  ChkNumber,
			'8':  ChkNumber,
			'9':  ChkNumber,
			'/':  ChkName,
			'(':  ChkString,
			'a':  ChkIdent,
			'b':  ChkIdent,
			'c':  ChkIdent,
			'd':  ChkIdent,
			'e':  ChkEndStream,
			'f':  ChkIdent,
			'g':  ChkIdent,
			'h':  ChkIdent,
			'i':  ChkIdent,
			'j':  ChkIdent,
			'k':  ChkIdent,
			'l':  ChkIdent,
			'm':  ChkIdent,
			'n':  ChkIdent,
			'o':  ChkIdent,
			'p':  ChkIdent,
			'q':  ChkIdent,
			'r':  ChkIdent,
			's':  ChkIdent,
			't':  ChkIdent,
			'u':  ChkIdent,
			'v':  ChkIdent,
			'w':  ChkIdent,
			'x':  ChkIdent,
			'y':  ChkIdent,
			'z':  ChkIdent,
			'A':  ChkIdent,
			'B':  ChkIdent,
			'C':  ChkIdent,
			'D':  ChkIdent,
			'E':  ChkIdent,
			'F':  ChkIdent,
			'G':  ChkIdent,
			'H':  ChkIdent,
			'I':  ChkIdent,
			'J':  ChkIdent,
			'K':  ChkIdent,
			'L':  ChkIdent,
			'M':  ChkIdent,
			'N':  ChkIdent,
			'O':  ChkIdent,
			'P':  ChkIdent,
			'Q':  ChkIdent,
			'R':  ChkIdent,
			'S':  ChkIdent,
			'T':  ChkIdent,
			'U':  ChkIdent,
			'V':  ChkIdent,
			'W':  ChkIdent,
			'X':  ChkIdent,
			'Y':  ChkIdent,
			'Z':  ChkIdent,
		},

		map[byte]StateHnd{ //  StTrailer
			' ':  Skip,
			'\t': Skip,
			'\r': Skip,
			'\n': Skip,
			'<':  ChkTrailerStart,
		},

		map[byte]StateHnd{ //  StStartXref
			' ':  Skip,
			'\t': Skip,
			'\r': Skip,
			'\n': Skip,
			's':  ChkStartXref,
		},
	}
}
