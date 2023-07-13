package streams

import (
	"context"
	"encoding/binary"
	"log"

	"github.com/tetratelabs/wazero/api"
)

func writeStreamFn(_ context.Context, mod api.Module, stream, ptr, l, result_ptr uint32) {
	data, ok := mod.Memory().Read(ptr, l)
	if !ok {
		log.Printf("Body read failed!\n")
		return
	}
	n, err := Streams.Write(stream, data)
	if err != nil {
		log.Printf("Failed to read: %v\n", err.Error())
	}

	data = []byte{}
	// 0 == is_ok, 1 == is_err
	le := binary.LittleEndian
	data = le.AppendUint32(data, 0)
	// write the number of bytes written
	data = le.AppendUint32(data, uint32(n))
	mod.Memory().Write(result_ptr, data)
}
