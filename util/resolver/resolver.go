package resolver

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/bluesky-social/indigo/api"
	"github.com/bluesky-social/indigo/did"
	"github.com/rs/zerolog"
)

var Resolver did.Resolver

func init() {
	resolver := did.NewMultiResolver()
	plcAddr := os.Getenv("ATP_PLC_ADDR")
	if plcAddr == "" {
		plcAddr = "https://plc.directory"
	}
	resolver.AddHandler("plc", &fallbackResolver{
		resolvers: []did.Resolver{
			&api.PLCServer{Host: plcAddr},
			&api.PLCServer{Host: "https://plc.directory"},
		}})
	resolver.AddHandler("web", &did.WebResolver{})

	Resolver = resolver
}

func GetDocument(ctx context.Context, didstr string) (*did.Document, error) {
	return Resolver.GetDocument(ctx, didstr)
}

type fallbackResolver struct {
	resolvers []did.Resolver
}

func (r *fallbackResolver) GetDocument(ctx context.Context, didstr string) (*did.Document, error) {
	log := zerolog.Ctx(ctx)
	errs := []error{}
	for _, res := range r.resolvers {
		if d, err := res.GetDocument(ctx, didstr); err == nil {
			return d, nil
		} else {
			log.Trace().Err(err).Str("plc", res.(*api.PLCServer).Host).
				Msgf("Failed to resolve %q using %q: %s", didstr, res.(*api.PLCServer).Host, err)
			errs = append(errs, err)
		}
	}
	return nil, errors.Join(errs...)
}

func (r *fallbackResolver) FlushCacheFor(did string) {
	for _, res := range r.resolvers {
		res.FlushCacheFor(did)
	}
}

func GetPDSEndpointAndPublicKey(ctx context.Context, did string) (*url.URL, string, error) {
	doc, err := GetDocument(ctx, did)
	if err != nil {
		return nil, "", fmt.Errorf("resolving did %q: %w", did, err)
	}

	pdsHost := ""
	for _, srv := range doc.Service {
		if srv.Type != "AtprotoPersonalDataServer" {
			continue
		}
		pdsHost = srv.ServiceEndpoint
	}
	if pdsHost == "" {
		return nil, "", fmt.Errorf("did not find any PDS in DID Document")
	}
	u, err := url.Parse(pdsHost)
	if err != nil {
		return nil, "", fmt.Errorf("PDS endpoint (%q) is an invalid URL: %w", pdsHost, err)
	}
	if u.Host == "" {
		return nil, "", fmt.Errorf("PDS endpoint (%q) doesn't have a host part", pdsHost)
	}

	key := ""
	for _, m := range doc.VerificationMethod {
		if m.ID != fmt.Sprintf("%s#atproto", did) {
			continue
		}
		if m.PublicKeyMultibase == nil {
			continue
		}
		key = *m.PublicKeyMultibase
	}
	if key == "" {
		return nil, "", fmt.Errorf("didn't find public key")
	}
	return u, key, nil
}
