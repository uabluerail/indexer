package repo

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/rs/zerolog"
	"gitlab.com/yawning/secp256k1-voi/secec"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multicodec"
)

type SignatureValidator func(digest []byte, sig []byte) (bool, error)

func parseSigningKey(ctx context.Context, key string) (SignatureValidator, error) {
	log := zerolog.Ctx(ctx)

	// const didKey = "did:key:"

	// if !strings.HasPrefix(key, didKey) {
	// 	return nil, fmt.Errorf("expected the key %q to have prefix %q", key, didKey)
	// }

	// key = strings.TrimPrefix(key, didKey)
	enc, val, err := multibase.Decode(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key data: %w", err)
	}

	if enc != multibase.Base58BTC {
		log.Info().Msgf("unexpected key encoding: %v", enc)
	}

	buf := bytes.NewBuffer(val)
	kind, err := binary.ReadUvarint(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key type: %w", err)
	}
	data, _ := io.ReadAll(buf)

	switch multicodec.Code(kind) {
	case multicodec.P256Pub:
		x, y := elliptic.UnmarshalCompressed(elliptic.P256(), data)
		return func(digest, sig []byte) (bool, error) {
			pk := &ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     x,
				Y:     y,
			}

			if len(sig) != 64 {
				return false, fmt.Errorf("unexpected signature length: %d != 64", len(sig))
			}
			r := big.NewInt(0).SetBytes(sig[:32])
			s := big.NewInt(0).SetBytes(sig[32:])
			return ecdsa.Verify(pk, digest, r, s), nil
		}, nil
	case multicodec.Secp256k1Pub:
		pk, err := secec.NewPublicKey(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse secp256k public key: %w", err)
		}
		return func(digest, sig []byte) (bool, error) {
			return pk.Verify(digest, sig, &secec.ECDSAOptions{
				Hash:            crypto.SHA256,
				Encoding:        secec.EncodingCompact,
				RejectMalleable: true,
			}), nil
		}, nil
	default:
		return nil, fmt.Errorf("unsupported key type %q", multicodec.Code(kind))
	}
}

func verifyCommitSignature(ctx context.Context, data []byte, key string) (bool, error) {
	validateSignature, err := parseSigningKey(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to parse the key: %w", err)
	}

	type Commit struct {
		DID     string
		Version int
		Data    cid.Cid
		Rev     string
		Prev    *cid.Cid
		Sig     []byte
	}

	builder := basicnode.Prototype.Any.NewBuilder()
	if err := (&dagcbor.DecodeOptions{AllowLinks: true}).Decode(builder, bytes.NewReader(data)); err != nil {
		return false, fmt.Errorf("unmarshaling commit: %w", err)
	}
	node := builder.Build()

	if node.Kind() != datamodel.Kind_Map {
		return false, fmt.Errorf("commit must be a Map, got %s instead", node.Kind())
	}

	m, err := parseMap(node)
	if err != nil {
		return false, err
	}

	commit := Commit{}

	if n, found := m["version"]; !found {
		return false, fmt.Errorf("missing \"version\"")
	} else {
		v, err := n.AsInt()
		if err != nil {
			return false, fmt.Errorf("failed to parse \"version\": %w", err)
		}
		commit.Version = int(v)
	}

	if n, found := m["did"]; !found {
		return false, fmt.Errorf("missing \"did\"")
	} else {
		v, err := n.AsString()
		if err != nil {
			return false, fmt.Errorf("failed to parse \"did\": %w", err)
		}
		commit.DID = v
	}

	if n, found := m["data"]; !found {
		return false, fmt.Errorf("missing \"data\"")
	} else {
		v, err := n.AsLink()
		if err != nil {
			return false, fmt.Errorf("failed to parse \"data\": %w", err)
		}
		c, err := cid.Parse([]byte(v.Binary()))
		if err != nil {
			return false, fmt.Errorf("failed to convert \"data\" to CID: %w", err)
		}
		commit.Data = c
	}

	if n, found := m["rev"]; !found {
		return false, fmt.Errorf("missing \"rev\"")
	} else {
		v, err := n.AsString()
		if err != nil {
			return false, fmt.Errorf("failed to parse \"rev\": %w", err)
		}
		commit.Rev = v
	}

	if n, found := m["prev"]; !found {
		return false, fmt.Errorf("missing \"prev\"")
	} else {
		if !n.IsNull() {
			v, err := n.AsLink()
			if err != nil {
				return false, fmt.Errorf("failed to parse \"prev\": %w", err)
			}
			c, err := cid.Parse([]byte(v.Binary()))
			if err != nil {
				return false, fmt.Errorf("failed to convert \"prev\" to CID: %w", err)
			}
			commit.Prev = &c
		}
	}

	if n, found := m["sig"]; !found {
		return false, fmt.Errorf("missing \"sig\"")
	} else {
		v, err := n.AsBytes()
		if err != nil {
			return false, fmt.Errorf("failed to parse \"sig\": %w", err)
		}
		commit.Sig = v
	}

	if commit.Version != 3 {
		return false, fmt.Errorf("unknown commit version %d", commit.Version)
	}

	unsignedBuilder := basicnode.Prototype.Map.NewBuilder()
	mb, err := unsignedBuilder.BeginMap(int64(len(m) - 1))
	if err != nil {
		return false, fmt.Errorf("initializing a map for unsigned commit: %w", err)
	}
	// XXX: signature validation depends on this specific order of keys in the map.
	for _, k := range []string{"did", "rev", "data", "prev", "version"} {
		if k == "sig" {
			continue
		}
		if err := mb.AssembleKey().AssignString(k); err != nil {
			return false, fmt.Errorf("failed to assemble key %q: %w", k, err)
		}
		if err := mb.AssembleValue().AssignNode(m[k]); err != nil {
			return false, fmt.Errorf("failed to assemble value for key %q: %w", k, err)
		}
	}
	if err := mb.Finish(); err != nil {
		return false, fmt.Errorf("failed to finalize the map: %w", err)
	}
	unsignedNode := unsignedBuilder.Build()

	buf := bytes.NewBuffer(nil)
	if err := (&dagcbor.EncodeOptions{AllowLinks: true}).Encode(unsignedNode, buf); err != nil {
		return false, fmt.Errorf("failed to serialize unsigned commit: %w", err)
	}
	unsignedBytes := buf.Bytes()
	unsignedHash := sha256.Sum256(unsignedBytes)
	return validateSignature(unsignedHash[:], commit.Sig)
}
