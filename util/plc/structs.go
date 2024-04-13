package plc

import (
	"bytes"
	"encoding/json"
	"fmt"

	cid "github.com/ipfs/go-cid"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	cbg "github.com/whyrusleeping/cbor-gen"
)

//go:generate go run ./gen

type Op struct {
	Type                string             `json:"type" cborgen:"type,const=plc_operation"`
	RotationKeys        []string           `json:"rotationKeys" cborgen:"rotationKeys"`
	VerificationMethods map[string]string  `json:"verificationMethods" cborgen:"verificationMethods"`
	AlsoKnownAs         []string           `json:"alsoKnownAs" cborgen:"alsoKnownAs"`
	Services            map[string]Service `json:"services" cborgen:"services"`
	Prev                *string            `json:"prev" cborgen:"prev"`
	Sig                 *string            `json:"sig" cborgen:"sig,omitempty"`
}

type Service struct {
	Type     string `json:"type" cborgen:"type"`
	Endpoint string `json:"endpoint" cborgen:"endpoint"`
}

type Tombstone struct {
	Type string  `json:"type" cborgen:"type,const=plc_tombstone"`
	Prev string  `json:"prev" cborgen:"prev"`
	Sig  *string `json:"sig" cborgen:"sig,omitempty"`
}

type LegacyCreateOp struct {
	Type        string  `json:"type" cborgen:"type,const=create"`
	SigningKey  string  `json:"signingKey" cborgen:"signingKey"`
	RecoveryKey string  `json:"recoveryKey" cborgen:"recoveryKey"`
	Handle      string  `json:"handle" cborgen:"handle"`
	Service     string  `json:"service" cborgen:"service"`
	Prev        *string `json:"prev" cborgen:"prev"`
	Sig         *string `json:"sig" cborgen:"sig,omitempty"`
}

func (op *LegacyCreateOp) AsUnsignedOp() Op {
	return Op{
		Type:         "plc_operation",
		Prev:         op.Prev,
		AlsoKnownAs:  []string{fmt.Sprintf("at://%s", op.Handle)},
		RotationKeys: []string{op.RecoveryKey},
		Services: map[string]Service{
			"atproto_pds": {
				Type:     "AtprotoPersonalDataServer",
				Endpoint: op.Service,
			}},
		VerificationMethods: map[string]string{
			"atproto": op.SigningKey,
		},
	}
}

type OperationKind interface {
	CID() (cid.Cid, error)
}

type Operation struct {
	Value OperationKind
}

type OperationLogEntry struct {
	DID       string    `json:"did"`
	Operation Operation `json:"operation"`
	CID       string    `json:"cid"`
	Nullified bool      `json:"nullified"`
	CreatedAt string    `json:"createdAt"`
}

func unmarshal[T any](b []byte) (T, error) {
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (o *Operation) UnmarshalJSON(b []byte) error {
	var partial struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(b, &partial); err != nil {
		return err
	}

	switch partial.Type {
	case "create":
		v, err := unmarshal[LegacyCreateOp](b)
		if err != nil {
			return err
		}
		o.Value = v
		return nil
	case "plc_operation":
		v, err := unmarshal[Op](b)
		if err != nil {
			return err
		}
		o.Value = v
		return nil
	case "plc_tombstone":
		v, err := unmarshal[Tombstone](b)
		if err != nil {
			return err
		}
		o.Value = v
		return nil
	default:
		return fmt.Errorf("unsupported 'type' value: %q", partial.Type)
	}
}

func (o Operation) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

func calculateCid(v cbg.CBORMarshaler) (cid.Cid, error) {
	b := bytes.NewBuffer(nil)
	if err := v.MarshalCBOR(b); err != nil {
		return cid.Cid{}, fmt.Errorf("marshaling as CBOR: %w", err)
	}
	return cid.V1Builder{
		Codec:  uint64(multicodec.DagCbor),
		MhType: multihash.SHA2_256,
	}.Sum(b.Bytes())
}

func (o Op) CID() (cid.Cid, error) {
	return calculateCid(&o)
}

func (o Tombstone) CID() (cid.Cid, error) {
	return calculateCid(&o)
}

func (o LegacyCreateOp) CID() (cid.Cid, error) {
	return calculateCid(&o)
}
