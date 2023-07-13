cc := /usr/local/lib/wasi-sdk-20.0/bin/clang

default: http.wasm

http.wasm: http.c
	${cc} proxy.c proxy_component_type.o http.c -o http.wasm
