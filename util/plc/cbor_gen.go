// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package plc

import (
	"fmt"
	"io"
	"math"
	"sort"

	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf
var _ = cid.Undef
var _ = math.E
var _ = sort.Sort

func (t *Service) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{162}); err != nil {
		return err
	}

	// t.Type (string) (string)
	if uint64(len("type")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"type\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("type"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("type")); err != nil {
		return err
	}

	if uint64(len(t.Type)) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Type was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Type))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Type)); err != nil {
		return err
	}

	// t.Endpoint (string) (string)
	if uint64(len("endpoint")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"endpoint\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("endpoint"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("endpoint")); err != nil {
		return err
	}

	if uint64(len(t.Endpoint)) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Endpoint was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Endpoint))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Endpoint)); err != nil {
		return err
	}
	return nil
}

func (t *Service) UnmarshalCBOR(r io.Reader) (err error) {
	*t = Service{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("Service: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadString(cr)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.Type (string) (string)
		case "type":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Type = string(sval)
			}
			// t.Endpoint (string) (string)
		case "endpoint":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Endpoint = string(sval)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *Op) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)
	fieldCount := 7

	if t.Sig == nil {
		fieldCount--
	}

	if _, err := cw.Write(cbg.CborEncodeMajorType(cbg.MajMap, uint64(fieldCount))); err != nil {
		return err
	}

	// t.Sig (string) (string)
	if t.Sig != nil {

		if uint64(len("sig")) > cbg.MaxLength {
			return xerrors.Errorf("Value in field \"sig\" was too long")
		}

		if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("sig"))); err != nil {
			return err
		}
		if _, err := cw.WriteString(string("sig")); err != nil {
			return err
		}

		if t.Sig == nil {
			if _, err := cw.Write(cbg.CborNull); err != nil {
				return err
			}
		} else {
			if uint64(len(*t.Sig)) > cbg.MaxLength {
				return xerrors.Errorf("Value in field t.Sig was too long")
			}

			if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(*t.Sig))); err != nil {
				return err
			}
			if _, err := cw.WriteString(string(*t.Sig)); err != nil {
				return err
			}
		}
	}

	// t.Prev (string) (string)
	if uint64(len("prev")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"prev\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("prev"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("prev")); err != nil {
		return err
	}

	if t.Prev == nil {
		if _, err := cw.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if uint64(len(*t.Prev)) > cbg.MaxLength {
			return xerrors.Errorf("Value in field t.Prev was too long")
		}

		if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(*t.Prev))); err != nil {
			return err
		}
		if _, err := cw.WriteString(string(*t.Prev)); err != nil {
			return err
		}
	}

	// t.Type (string) (string)
	if uint64(len("type")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"type\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("type"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("type")); err != nil {
		return err
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("plc_operation"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("plc_operation")); err != nil {
		return err
	}

	// t.Services (map[string]plc.Service) (map)
	if uint64(len("services")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"services\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("services"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("services")); err != nil {
		return err
	}

	{
		if len(t.Services) > 4096 {
			return xerrors.Errorf("cannot marshal t.Services map too large")
		}

		if err := cw.WriteMajorTypeHeader(cbg.MajMap, uint64(len(t.Services))); err != nil {
			return err
		}

		keys := make([]string, 0, len(t.Services))
		for k := range t.Services {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := t.Services[k]

			if uint64(len(k)) > cbg.MaxLength {
				return xerrors.Errorf("Value in field k was too long")
			}

			if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(k))); err != nil {
				return err
			}
			if _, err := cw.WriteString(string(k)); err != nil {
				return err
			}

			if err := v.MarshalCBOR(cw); err != nil {
				return err
			}

		}
	}

	// t.AlsoKnownAs ([]string) (slice)
	if uint64(len("alsoKnownAs")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"alsoKnownAs\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("alsoKnownAs"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("alsoKnownAs")); err != nil {
		return err
	}

	if uint64(len(t.AlsoKnownAs)) > cbg.MaxLength {
		return xerrors.Errorf("Slice value in field t.AlsoKnownAs was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajArray, uint64(len(t.AlsoKnownAs))); err != nil {
		return err
	}
	for _, v := range t.AlsoKnownAs {
		if uint64(len(v)) > cbg.MaxLength {
			return xerrors.Errorf("Value in field v was too long")
		}

		if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(v))); err != nil {
			return err
		}
		if _, err := cw.WriteString(string(v)); err != nil {
			return err
		}

	}

	// t.RotationKeys ([]string) (slice)
	if uint64(len("rotationKeys")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"rotationKeys\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("rotationKeys"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("rotationKeys")); err != nil {
		return err
	}

	if uint64(len(t.RotationKeys)) > cbg.MaxLength {
		return xerrors.Errorf("Slice value in field t.RotationKeys was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajArray, uint64(len(t.RotationKeys))); err != nil {
		return err
	}
	for _, v := range t.RotationKeys {
		if uint64(len(v)) > cbg.MaxLength {
			return xerrors.Errorf("Value in field v was too long")
		}

		if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(v))); err != nil {
			return err
		}
		if _, err := cw.WriteString(string(v)); err != nil {
			return err
		}

	}

	// t.VerificationMethods (map[string]string) (map)
	if uint64(len("verificationMethods")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"verificationMethods\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("verificationMethods"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("verificationMethods")); err != nil {
		return err
	}

	{
		if len(t.VerificationMethods) > 4096 {
			return xerrors.Errorf("cannot marshal t.VerificationMethods map too large")
		}

		if err := cw.WriteMajorTypeHeader(cbg.MajMap, uint64(len(t.VerificationMethods))); err != nil {
			return err
		}

		keys := make([]string, 0, len(t.VerificationMethods))
		for k := range t.VerificationMethods {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := t.VerificationMethods[k]

			if uint64(len(k)) > cbg.MaxLength {
				return xerrors.Errorf("Value in field k was too long")
			}

			if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(k))); err != nil {
				return err
			}
			if _, err := cw.WriteString(string(k)); err != nil {
				return err
			}

			if uint64(len(v)) > cbg.MaxLength {
				return xerrors.Errorf("Value in field v was too long")
			}

			if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(v))); err != nil {
				return err
			}
			if _, err := cw.WriteString(string(v)); err != nil {
				return err
			}

		}
	}
	return nil
}

func (t *Op) UnmarshalCBOR(r io.Reader) (err error) {
	*t = Op{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("Op: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadString(cr)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.Sig (string) (string)
		case "sig":

			{
				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}

					sval, err := cbg.ReadString(cr)
					if err != nil {
						return err
					}

					t.Sig = (*string)(&sval)
				}
			}
			// t.Prev (string) (string)
		case "prev":

			{
				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}

					sval, err := cbg.ReadString(cr)
					if err != nil {
						return err
					}

					t.Prev = (*string)(&sval)
				}
			}
			// t.Type (string) (string)
		case "type":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Type = string(sval)
			}
			// t.Services (map[string]plc.Service) (map)
		case "services":

			maj, extra, err = cr.ReadHeader()
			if err != nil {
				return err
			}
			if maj != cbg.MajMap {
				return fmt.Errorf("expected a map (major type 5)")
			}
			if extra > 4096 {
				return fmt.Errorf("t.Services: map too large")
			}

			t.Services = make(map[string]Service, extra)

			for i, l := 0, int(extra); i < l; i++ {

				var k string

				{
					sval, err := cbg.ReadString(cr)
					if err != nil {
						return err
					}

					k = string(sval)
				}

				var v Service

				{

					if err := v.UnmarshalCBOR(cr); err != nil {
						return xerrors.Errorf("unmarshaling v: %w", err)
					}

				}

				t.Services[k] = v

			}
			// t.AlsoKnownAs ([]string) (slice)
		case "alsoKnownAs":

			maj, extra, err = cr.ReadHeader()
			if err != nil {
				return err
			}

			if extra > cbg.MaxLength {
				return fmt.Errorf("t.AlsoKnownAs: array too large (%d)", extra)
			}

			if maj != cbg.MajArray {
				return fmt.Errorf("expected cbor array")
			}

			if extra > 0 {
				t.AlsoKnownAs = make([]string, extra)
			}

			for i := 0; i < int(extra); i++ {
				{
					var maj byte
					var extra uint64
					var err error
					_ = maj
					_ = extra
					_ = err

					{
						sval, err := cbg.ReadString(cr)
						if err != nil {
							return err
						}

						t.AlsoKnownAs[i] = string(sval)
					}

				}
			}
			// t.RotationKeys ([]string) (slice)
		case "rotationKeys":

			maj, extra, err = cr.ReadHeader()
			if err != nil {
				return err
			}

			if extra > cbg.MaxLength {
				return fmt.Errorf("t.RotationKeys: array too large (%d)", extra)
			}

			if maj != cbg.MajArray {
				return fmt.Errorf("expected cbor array")
			}

			if extra > 0 {
				t.RotationKeys = make([]string, extra)
			}

			for i := 0; i < int(extra); i++ {
				{
					var maj byte
					var extra uint64
					var err error
					_ = maj
					_ = extra
					_ = err

					{
						sval, err := cbg.ReadString(cr)
						if err != nil {
							return err
						}

						t.RotationKeys[i] = string(sval)
					}

				}
			}
			// t.VerificationMethods (map[string]string) (map)
		case "verificationMethods":

			maj, extra, err = cr.ReadHeader()
			if err != nil {
				return err
			}
			if maj != cbg.MajMap {
				return fmt.Errorf("expected a map (major type 5)")
			}
			if extra > 4096 {
				return fmt.Errorf("t.VerificationMethods: map too large")
			}

			t.VerificationMethods = make(map[string]string, extra)

			for i, l := 0, int(extra); i < l; i++ {

				var k string

				{
					sval, err := cbg.ReadString(cr)
					if err != nil {
						return err
					}

					k = string(sval)
				}

				var v string

				{
					sval, err := cbg.ReadString(cr)
					if err != nil {
						return err
					}

					v = string(sval)
				}

				t.VerificationMethods[k] = v

			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *Tombstone) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)
	fieldCount := 3

	if t.Sig == nil {
		fieldCount--
	}

	if _, err := cw.Write(cbg.CborEncodeMajorType(cbg.MajMap, uint64(fieldCount))); err != nil {
		return err
	}

	// t.Sig (string) (string)
	if t.Sig != nil {

		if uint64(len("sig")) > cbg.MaxLength {
			return xerrors.Errorf("Value in field \"sig\" was too long")
		}

		if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("sig"))); err != nil {
			return err
		}
		if _, err := cw.WriteString(string("sig")); err != nil {
			return err
		}

		if t.Sig == nil {
			if _, err := cw.Write(cbg.CborNull); err != nil {
				return err
			}
		} else {
			if uint64(len(*t.Sig)) > cbg.MaxLength {
				return xerrors.Errorf("Value in field t.Sig was too long")
			}

			if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(*t.Sig))); err != nil {
				return err
			}
			if _, err := cw.WriteString(string(*t.Sig)); err != nil {
				return err
			}
		}
	}

	// t.Prev (string) (string)
	if uint64(len("prev")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"prev\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("prev"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("prev")); err != nil {
		return err
	}

	if uint64(len(t.Prev)) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Prev was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Prev))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Prev)); err != nil {
		return err
	}

	// t.Type (string) (string)
	if uint64(len("type")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"type\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("type"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("type")); err != nil {
		return err
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("plc_tombstone"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("plc_tombstone")); err != nil {
		return err
	}
	return nil
}

func (t *Tombstone) UnmarshalCBOR(r io.Reader) (err error) {
	*t = Tombstone{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("Tombstone: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadString(cr)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.Sig (string) (string)
		case "sig":

			{
				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}

					sval, err := cbg.ReadString(cr)
					if err != nil {
						return err
					}

					t.Sig = (*string)(&sval)
				}
			}
			// t.Prev (string) (string)
		case "prev":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Prev = string(sval)
			}
			// t.Type (string) (string)
		case "type":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Type = string(sval)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *LegacyCreateOp) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)
	fieldCount := 7

	if t.Sig == nil {
		fieldCount--
	}

	if _, err := cw.Write(cbg.CborEncodeMajorType(cbg.MajMap, uint64(fieldCount))); err != nil {
		return err
	}

	// t.Sig (string) (string)
	if t.Sig != nil {

		if uint64(len("sig")) > cbg.MaxLength {
			return xerrors.Errorf("Value in field \"sig\" was too long")
		}

		if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("sig"))); err != nil {
			return err
		}
		if _, err := cw.WriteString(string("sig")); err != nil {
			return err
		}

		if t.Sig == nil {
			if _, err := cw.Write(cbg.CborNull); err != nil {
				return err
			}
		} else {
			if uint64(len(*t.Sig)) > cbg.MaxLength {
				return xerrors.Errorf("Value in field t.Sig was too long")
			}

			if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(*t.Sig))); err != nil {
				return err
			}
			if _, err := cw.WriteString(string(*t.Sig)); err != nil {
				return err
			}
		}
	}

	// t.Prev (string) (string)
	if uint64(len("prev")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"prev\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("prev"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("prev")); err != nil {
		return err
	}

	if t.Prev == nil {
		if _, err := cw.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if uint64(len(*t.Prev)) > cbg.MaxLength {
			return xerrors.Errorf("Value in field t.Prev was too long")
		}

		if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(*t.Prev))); err != nil {
			return err
		}
		if _, err := cw.WriteString(string(*t.Prev)); err != nil {
			return err
		}
	}

	// t.Type (string) (string)
	if uint64(len("type")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"type\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("type"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("type")); err != nil {
		return err
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("create"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("create")); err != nil {
		return err
	}

	// t.Handle (string) (string)
	if uint64(len("handle")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"handle\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("handle"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("handle")); err != nil {
		return err
	}

	if uint64(len(t.Handle)) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Handle was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Handle))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Handle)); err != nil {
		return err
	}

	// t.Service (string) (string)
	if uint64(len("service")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"service\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("service"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("service")); err != nil {
		return err
	}

	if uint64(len(t.Service)) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Service was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Service))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Service)); err != nil {
		return err
	}

	// t.SigningKey (string) (string)
	if uint64(len("signingKey")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"signingKey\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("signingKey"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("signingKey")); err != nil {
		return err
	}

	if uint64(len(t.SigningKey)) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.SigningKey was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.SigningKey))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.SigningKey)); err != nil {
		return err
	}

	// t.RecoveryKey (string) (string)
	if uint64(len("recoveryKey")) > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"recoveryKey\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("recoveryKey"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("recoveryKey")); err != nil {
		return err
	}

	if uint64(len(t.RecoveryKey)) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.RecoveryKey was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.RecoveryKey))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.RecoveryKey)); err != nil {
		return err
	}
	return nil
}

func (t *LegacyCreateOp) UnmarshalCBOR(r io.Reader) (err error) {
	*t = LegacyCreateOp{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("LegacyCreateOp: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadString(cr)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.Sig (string) (string)
		case "sig":

			{
				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}

					sval, err := cbg.ReadString(cr)
					if err != nil {
						return err
					}

					t.Sig = (*string)(&sval)
				}
			}
			// t.Prev (string) (string)
		case "prev":

			{
				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}

					sval, err := cbg.ReadString(cr)
					if err != nil {
						return err
					}

					t.Prev = (*string)(&sval)
				}
			}
			// t.Type (string) (string)
		case "type":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Type = string(sval)
			}
			// t.Handle (string) (string)
		case "handle":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Handle = string(sval)
			}
			// t.Service (string) (string)
		case "service":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Service = string(sval)
			}
			// t.SigningKey (string) (string)
		case "signingKey":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.SigningKey = string(sval)
			}
			// t.RecoveryKey (string) (string)
		case "recoveryKey":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.RecoveryKey = string(sval)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
