package resolver

import (
	"context"
	"errors"
	"os"

	"github.com/bluesky-social/indigo/api"
	"github.com/bluesky-social/indigo/did"
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
	errs := []error{}
	for _, res := range r.resolvers {
		if d, err := res.GetDocument(ctx, didstr); err == nil {
			return d, nil
		} else {
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
