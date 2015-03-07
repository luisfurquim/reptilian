package reptilian

import (
	"io"
	//   "log"
	"errors"
	"fmt"
	"regexp"
	//   "Goose"
	//   "reflect"
)

var reCrlf *regexp.Regexp

func init() {
	reCrlf = regexp.MustCompile("^([\r\n]+)")
}

func (buf *Buf) Add(tk Token) error {
	var a *ArrayT
	var iobj IndObjT

	switch tk.(type) {
	case *BlankT:
		return nil
	}

	switch buf.DataStack[buf.StateLvl].(type) {
	case *ArrayT:
		a = buf.DataStack[buf.StateLvl].(*ArrayT)
		switch tk.(type) {
		case *IdentT:
			id := *tk.(*IdentT)
			if (len(id) == 1) && (id[0] == 'R') && (len(*a) >= 2) {
				switch (*a)[len(*a)-2].(type) {
				case *IntT:
					iobj = IndObjT(*((*a)[len(*a)-2].(*IntT)))
					(*a)[len(*a)-2] = &iobj
					(*a) = (*a)[:len(*a)-1]
					Goose.Logf(6, "Found IObj.a %s lvl: %d, datastack: %s,   data:%s", iobj, buf.StateLvl, buf.DataStack, buf.Data)
				default:
					return errors.New(fmt.Sprintf("Malformed object reference.a IntT expected, got %s", (*a)[len(*a)-2]))
				}
			} else {
				return errors.New(fmt.Sprintf("Ident unexpected, got %s", tk))
			}
		default:
			Goose.Logf(5, "ADD ON ARRAY lvl: %d, datastack: %s,   data:%s", buf.StateLvl, buf.DataStack, buf.Data)
			(*a) = append(*(buf.DataStack[buf.StateLvl].(*ArrayT)), tk)
			Goose.Logf(5, "ADD ON ARRAY lvl: %d, datastack: %s,   data:%s", buf.StateLvl, buf.DataStack, buf.Data)
		}
		return nil
	case *DicT:
		Goose.Logf(5, "ADD ON DICT lvl: %d, datastack: %s,   data:%s", buf.StateLvl, buf.DataStack, buf.Data)
		d := buf.DataStack[buf.StateLvl].(*DicT)
		if len(d.NewName) == 0 {
			switch tk.(type) {
			case *NameT:
				d.NewName = string(*(tk.(*NameT)))
			case *IntT:
				switch d.Dic[d.LastName].(type) {
				case *IntT:
					d.Dic[d.LastName] = &ArrayT{d.Dic[d.LastName], tk}
				default:
					return errors.New(fmt.Sprintf("Name.1 expected, got %s", tk))
				}
			case *IdentT:
				switch d.Dic[d.LastName].(type) {
				case *ArrayT:
					id := *tk.(*IdentT)
					if (len(id) == 1) && (id[0] == 'R') && (len(*(d.Dic[d.LastName].(*ArrayT))) == 2) {
						switch (*(d.Dic[d.LastName].(*ArrayT)))[0].(type) {
						case *IntT:
							iobj = IndObjT(*((*(d.Dic[d.LastName].(*ArrayT)))[0].(*IntT)))
							d.Dic[d.LastName] = &iobj
							Goose.Logf(6, "Found IObj.d %s:%s lvl: %d, datastack: %s,   data:%s", d.LastName, iobj, buf.StateLvl, buf.DataStack, buf.Data)
						default:
							return errors.New(fmt.Sprintf("Malformed object reference.a IntT expected, got %s", (*(d.Dic[d.LastName].(*ArrayT)))[0]))
						}
					} else {
						return errors.New(fmt.Sprintf("Name.2 expected, got %s", tk))
					}
					//                        d.Dic[d.LastName] = &ArrayT{d.Dic[d.LastName], tk}
				default:
					return errors.New(fmt.Sprintf("Name.3 expected, got %s", tk))
				}
			default:
				return errors.New(fmt.Sprintf("Name.4 expected, got %s", tk))
			}
		} else {
			d.Dic[d.NewName] = tk
			d.LastName = d.NewName
			d.NewName = ""
		}
		Goose.Logf(4, "ADD ON DICT lvl: %d, datastack: %s,   data:%s", buf.StateLvl, buf.DataStack, buf.Data)
		return nil
	}
	//   Goose.Logf(5,"lvl: %d, datastack top: %s,   data:%s",buf.StateLvl,buf.DataStack[buf.StateLvl],buf.Data)
	return errors.New("Undefined type")
}

func (d *Doc) mkList(name string, n int) error {
	var typ string
	var ok bool
	var iKid int
	var kids []Token

	if _, ok = d.Buf.Obj[n].Header.Dic["Type"]; !ok {
		errors.New("Unknown type")
	}

	Goose.Logf(7, "Walking on object (%s) %s", typ, d.Buf.Obj[n])

	switch d.Buf.Obj[n].Header.Dic["Type"].(type) {
	case *NameT:
		typ = string(*(d.Buf.Obj[n].Header.Dic["Type"].(*NameT)))
		Goose.Logf(7, "Walking on object of recognized type %s", typ)
	default:
		Goose.Logf(1, "Walking on object of type %s", d.Buf.Obj[n].Header.Dic["Type"])
		return errors.New("Type info corrupted")
	}

	if typ == "Page" {
		Goose.Logf(4, "Found page %d", len(d.Index[name]))
		d.Index[name] = append(d.Index[name], d.Buf.Obj[n])
		return nil
	}

	if typ != "Pages" {
		return errors.New("Not a Page type")
	}

	switch d.Buf.Obj[n].Header.Dic["Kids"].(type) {
	case *ArrayT:
		kids = *(d.Buf.Obj[n].Header.Dic["Kids"].(*ArrayT))
	default:
		return errors.New("Corrupted Kid list")
	}

	for _, kid := range kids {
		switch kid.(type) {
		case *IndObjT:
			iKid = int(*(kid.(*IndObjT)))
			if iKid > len(d.Buf.Obj) {
				return errors.New("Kid reference overflow")
			}
			d.mkList(name, iKid)
		default:
			return errors.New("Invalid kid reference")
		}
	}

	return nil
}

func SMWalk(buf *Buf, sz int64) {
	var ok bool
	var Hnd StateHnd
	var tk Token
	var err error

	for (buf.StateLvl > 0) || (buf.State[buf.StateLvl] > 0) || (buf.Off < (sz - 6)) {
		if Hnd, ok = sm[buf.State[buf.StateLvl]][buf.Raw[buf.Off]]; ok {
			tk = Hnd(buf)
			if tk == nil {
				Goose.Fatalf(1, "Error nil token at %d (%o)", buf.Off, buf.Off)
			}
			Goose.Logf(3, "%d: %s", buf.Off, tk)
			err = buf.Add(tk)
			if (err != nil) && (err.Error() != "Can't Add items") {
				Goose.Fatalf(1, "Error (off=%d/%o) adding token: %s", buf.Off, buf.Off, err)
			}
		} else {
			Goose.Logf(1, "Undef Hnd: Off=%d (%o),  %s", buf.Off, buf.Off, buf.Raw[buf.Off:buf.Off+10])
			Goose.Logf(1, "OBJECTS: ")
			for i, o := range buf.Obj {
				if o != nil {
					Goose.Logf(1, "%d:    %s", i, o)
				}
			}
			return
		}
	}
}

func New(Hnd io.Reader, sz int64) (*Doc, error) {
	var d Doc
	var n, size int
	var err error
	//   var ok     bool
	var cat *DicT
	//   var pages *ObjT
	var count *IntT

	// check size and panic if big

	d = Doc{
		Hnd: Hnd,
		Buf: Buf{
			Raw:   make([]byte, sz, sz),
			Off:   0,
			State: []int{0},
			Data:  ArrayT(make([]Token, 0, sz/10)),
			Obj:   []*ObjT{},
		},
	}
	d.Buf.DataStack = ArrayT{&d.Buf.Data}

	for err == nil {
		n, err = Hnd.Read(d.Buf.Raw[n:])
	}

	if err != io.EOF {
		if Goose > 0 {
			Goose.Logf(1, "Error reading PDF: %q", err)
		}
		return nil, err
	}

	if string(d.Buf.Raw[:5]) != "%PDF-" {
		err = errors.New("Not a PDF file:")
		if Goose > 0 {
			Goose.Logf(1, "%s", err)
		}
		return nil, err
	}

	found := false
	for d.Buf.Off = 5; !found; d.Buf.Off++ {
		m := reCrlf.FindSubmatch(d.Buf.Raw[d.Buf.Off:])
		if len(m) == 2 {
			found = true
			if len(m[1]) == 2 {
				d.Buf.Off++
			}
		}
	}
	d.Version = string(d.Buf.Raw[5:d.Buf.Off])
	found = false
	for ; !found; d.Buf.Off++ {
		m := reCrlf.FindSubmatch(d.Buf.Raw[d.Buf.Off:])
		if len(m) == 2 {
			found = true
			if len(m[1]) == 2 {
				d.Buf.Off++
			}
		}
	}

	Goose.Logf(3, "version=%s", d.Version)

	SMWalk(&d.Buf, sz)

	d.Index = make(map[string][]*ObjT)

	cat, err = d.Catalog()
	if err == nil {
		for name, entry := range cat.Dic {
			switch entry.(type) {
			case *IndObjT:
				n = int(*(entry.(*IndObjT)))
				if n < len(d.Buf.Obj) {
					switch d.Buf.Obj[n].Header.Dic["Count"].(type) {
					case *IntT:
						count = d.Buf.Obj[n].Header.Dic["Count"].(*IntT)
						size = int(*count)
						Goose.Logf(1, "Page count: %s", count)
					default:
						Goose.Logf(1, "Page count: %s", d.Buf.Obj[n].Header.Dic)
						return nil, errors.New("Corrupted page count")
					}
					if size > 0 {
						d.Index[name] = make([]*ObjT, 0, size)
						d.mkList(name, n)
					}
				}
			}
		}
	}

	/*
	   if pages, ok = d.Index["Pages"]; !ok {
	      pages, err = d.Pages()
	      if err == nil {
	         d.Index["Pages"] = pages
	      }
	   }
	*/

	if _, ok := d.Index["Pages"]; !ok {
		return nil, errors.New("Corrupted PDF! Pages not found!")
	}

	Goose.Logf(3, "Final Off=%d (%o)", d.Buf.Off, d.Buf.Off)
	Goose.Logf(7, "Index: %#v", d.Index)

	return &d, nil
}
