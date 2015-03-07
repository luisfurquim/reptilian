package reptext

import (
	//   "errors"
	"fmt"
	"github.com/luisfurquim/goose"
	"github.com/luisfurquim/reptilian"
	"regexp"
)

type Doc struct {
	reptilian.Doc
}

type TxtT struct {
	X, Y int
	Txt  []string
}

var reLines *regexp.Regexp
var Goose goose.Alert

func init() {
	reLines = regexp.MustCompile("[\r\n]+")
}

func getString(line string) string {
	var buf reptilian.Buf

	buf = reptilian.Buf{
		Raw:      []byte(line),
		Off:      0,
		State:    []int{reptilian.StObjStream},
		StateLvl: 0,
	}
	t := reptilian.ChkString(&buf)
	if t == nil {
		Goose.Logf(1, "nil string at %s", line)
	} else {
		switch t.(type) {
		case *reptilian.StringT:
			return string(*(t.(*reptilian.StringT)))
		default:
			Goose.Logf(1, "corrupted string at %s", line)
		}
	}
	return ""
}

func (d *Doc) Text(n int) ([]TxtT, error) {
	var err error
	var s string
	var line string
	var txt []TxtT
	var inTxt bool
	var i int
	var l1, l2 byte

	s, err = d.Content(n)
	if err != nil {
		return nil, err
	}

	txt = []TxtT{}
	inTxt = false
	for _, line = range reLines.Split(s, -1) {
		if inTxt {
			if line == "ET" {
				inTxt = false
				i++
				continue
			}
			if len(line) > 2 {
				l1 = line[len(line)-1]
				l2 = line[len(line)-2]
				if l2 == 'T' {
					if l1 == 'j' {
						txt[i].Txt = append(txt[i].Txt, getString(line))
					} else if line[len(line)-1] == 'J' {
						Goose.Logf(1, "TJ formated string at %s", line)
						txt[i].Txt = append(txt[i].Txt, line)
					} else if line[len(line)-1] == '*' {
						txt[i].Txt = append(txt[i].Txt, "\n")
					} else if line[len(line)-1] == 'd' {
						fmt.Sscanf(line, "%d %d", &txt[i].X, &txt[i].Y)
						Goose.Logf(1, "Td formated string at %s", line)
					} else if line[len(line)-1] == 'D' {
						Goose.Logf(1, "TD formated string at %s", line)
					}
				} else if l1 == '\'' {
					txt[i].Txt = append(txt[i].Txt, getString(line))
				} else if l1 == '"' {
				}
			}
		} else {
			if line == "BT" {
				inTxt = true
				txt = append(txt, TxtT{Txt: []string{}})
			}
		}
	}

	return txt, nil
}
