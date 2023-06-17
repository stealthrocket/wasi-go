package wasi_http

import (
	"context"

	"github.com/stealthrocket/wasi-go/imports/wasi_http/default_http"
	"github.com/stealthrocket/wasi-go/imports/wasi_http/types"
	"github.com/stealthrocket/wasi-go/imports/wasi_http/wasi_streams"
	"github.com/tetratelabs/wazero"
)

func Instantiate(ctx context.Context, rt wazero.Runtime) error {
	if err := types.Instantiate(ctx, rt); err != nil {
		return err
	}
	if err := wasi_streams.Instantiate(ctx, rt); err != nil {
		return err
	}
	if err := default_http.Instantiate(ctx, rt); err != nil {
		return err
	}
	return nil
}
