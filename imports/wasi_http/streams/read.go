package streams

import (
	"context"
	"encoding/binary"
	"log"

	"github.com/stealthrocket/wasi-go/imports/wasi_http/common"
	"github.com/tetratelabs/wazero/api"
)

func streamReadFn(ctx context.Context, mod api.Module, stream_handle uint32, length uint64, out_ptr uint32) {
	data := make([]byte, length)
	_, _, err := Streams.Read(stream_handle, data)

	//	data, err := types.ResponseBody()
	if err != nil {
		log.Fatalf(err.Error())
	}

	ptr_len := uint32(len(data)) + 1
	data = append(data, 0)
	ptr, err := common.Malloc(ctx, mod, ptr_len)
	if err != nil {
		log.Fatalf(err.Error())
	}
	mod.Memory().Write(ptr, data)

	data = []byte{}
	// 0 == is_ok, 1 == is_err
	le := binary.LittleEndian
	data = le.AppendUint32(data, 0)
	data = le.AppendUint32(data, ptr)
	data = le.AppendUint32(data, ptr_len)
	// No more data to read.
	data = le.AppendUint32(data, 0)
	mod.Memory().Write(out_ptr, data)
}

func dropInputStreamFn(_ context.Context, mod api.Module, stream uint32) {
	Streams.DeleteStream(stream)
}
