package streams

import (
	"context"
	"encoding/binary"
	"log"

	"github.com/stealthrocket/wasi-go/imports/wasi_http/common"
	"github.com/tetratelabs/wazero/api"
)

func streamReadFn(ctx context.Context, mod api.Module, stream_handle uint32, length uint64, out_ptr uint32) {
	rawData := make([]byte, length)
	n, done, err := Streams.Read(stream_handle, rawData)

	//	data, err := types.ResponseBody()
	if err != nil {
		log.Fatalf(err.Error())
	}

	data := rawData[0:n]
	ptr_len := uint32(len(data))
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
	if done {
		// No more data to read.
		data = le.AppendUint32(data, 0)
	} else {
		data = le.AppendUint32(data, 1)
	}
	mod.Memory().Write(out_ptr, data)
}

func dropInputStreamFn(_ context.Context, mod api.Module, stream uint32) {
	Streams.DeleteStream(stream)
}
