package reptilian

import (
	"errors"
)

func (d *Doc) getObj(ref *IndObjT) (*ObjT, error) {
	var i int

	i = int(*ref) - 1
	if len(d.Buf.Obj) < i {
		return nil, errors.New("Invalid object reference")
	}
	return d.Buf.Obj[i], nil
}

func getType(o *ObjT) (string, error) {
	Goose.Logf(1, "Dic: %s", o.Header)
	ident := o.Header.Dic["Type"]
	switch ident.(type) {
	case *NameT:
		return string(*ident.(*NameT)), nil
	default:
		return "", errors.New("Unknown type")
	}
}

func (d *Doc) getCheckingType(ref *IndObjT, t string) (*ObjT, error) {
	var i int
	var o *ObjT
	var s string
	var err error

	i = int(*ref)
	if len(d.Buf.Obj) < i {
		return nil, errors.New("Invalid '" + t + "' reference")
	}
	o, err = d.getObj(ref)
	if err != nil {
		return nil, err
	}

	s, err = getType(o)
	if err != nil {
		return nil, err
	}

	if s == t {
		return o, nil
	}
	return nil, errors.New("Corrupted '" + t + "'")
}

func (d *Doc) Catalog() (*DicT, error) {
	var o *ObjT
	var err error

	Goose.Logf(1, "trailer: %s", d.Buf.Trailer)
	info := d.Buf.Trailer.Dic["Info"]
	Goose.Logf(1, "info: %s", info)
	switch info.(type) {
	case *IndObjT:
		o, err = d.getCheckingType(info.(*IndObjT), "Catalog")
		if err != nil {
			return nil, err
		}

		return &o.Header, nil
	default:
		return nil, errors.New("Catalog not found")
	}
}

func (d *Doc) Pages() (*ObjT, error) {

	root := d.Buf.Trailer.Dic["Root"]
	switch root.(type) {
	case *IndObjT:
		return d.getCheckingType(root.(*IndObjT), "Pages")

	default:
		// Corrupted PDF: Root not found!
		// Let's try the catalog...
		info, err := d.Catalog()
		if err != nil {
			return nil, err
		}
		pages, ok := info.Dic["Pages"]
		if !ok {
			return nil, errors.New("Pages not found")
		}
		switch pages.(type) {
		case *IndObjT:
			return d.getCheckingType(pages.(*IndObjT), "Page")
		default:
			return nil, errors.New("Pages not found")
		}
	}
}

func (d *Doc) Content(n int) (string, error) {
	var i int

	switch d.Index["Pages"][n].Header.Dic["Contents"].(type) {
	case *IndObjT:
		i = int(*(d.Index["Pages"][n].Header.Dic["Contents"].(*IndObjT)))
		if i > len(d.Buf.Obj) {
			return "", errors.New("Invalid page content reference")
		}

		switch d.Buf.Obj[i].Stream[0].(type) {
		case *StringT:
			return string(*(d.Buf.Obj[i].Stream[0].(*StringT))), nil
		default:
			return "", errors.New("Page content not found")
		}

	default:
		return "", errors.New("Corrupted page content")
	}
}
