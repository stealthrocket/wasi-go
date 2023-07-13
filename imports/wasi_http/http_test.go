package wasi_http

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stealthrocket/wasi-go"
	"github.com/stealthrocket/wasi-go/imports"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/sys"
)

type handler struct {
	urls   []string
	bodies []string
}

func deepEquals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	} else {
		for ix := range a {
			if a[ix] != b[ix] {
				return false
			}
		}
	}
	return true
}

func (h *handler) reset() {
	h.bodies = []string{}
	h.urls = []string{}
}

func (h *handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	body := ""

	if req.Body != nil {
		defer req.Body.Close()
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			panic(err.Error())
		}
		body = string(data)
	}

	res.WriteHeader(200)
	res.Write([]byte("Response"))

	h.urls = append(h.urls, req.URL.String())
	h.bodies = append(h.bodies, body)
}

func TestHttp(t *testing.T) {
	filePaths, _ := filepath.Glob("../../testdata/c/http/*.wasm")
	for _, file := range filePaths {
		fmt.Printf("%v\n", file)
	}
	if len(filePaths) == 0 {
		t.Log("nothing to test")
	}

	h := handler{}
	s := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: &h,
	}
	go s.ListenAndServe()
	defer s.Shutdown(context.TODO())

	expectedPaths := [][]string{
		{
			"/get?some=arg&goes=here",
			"/post",
		},
	}

	expectedBodies := [][]string{
		{
			"",
			"{\"foo\": \"bar\"}",
		},
	}

	for testIx, test := range filePaths {
		name := test
		for strings.HasPrefix(name, "../") {
			name = name[3:]
		}

		t.Run(name, func(t *testing.T) {
			bytecode, err := os.ReadFile(test)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()

			runtime := wazero.NewRuntime(ctx)
			defer runtime.Close(ctx)

			builder := imports.NewBuilder().
				WithName("http").
				WithArgs()
			var system wasi.System
			ctx, system, err = builder.Instantiate(ctx, runtime)
			if err != nil {
				t.Error("Failed to build WASI module: ", err)
			}
			defer system.Close(ctx)

			Instantiate(ctx, runtime)

			instance, err := runtime.Instantiate(ctx, bytecode)
			if err != nil {
				switch e := err.(type) {
				case *sys.ExitError:
					if exitCode := e.ExitCode(); exitCode != 0 {
						t.Error("exit code:", exitCode)
						t.FailNow()
					}
				default:
					t.Error("instantiating wasm module instance:", err)
				}
			}
			if instance != nil {
				if err := instance.Close(ctx); err != nil {
					t.Error("closing wasm module instance:", err)
				}
			}
			if !deepEquals(expectedPaths[testIx], h.urls) {
				t.Errorf("Unexpected paths: %v vs %v", h.urls, expectedPaths[testIx])
			}
			if !deepEquals(expectedBodies[testIx], h.bodies) {
				t.Errorf("Unexpected paths: %v vs %v", h.bodies, expectedBodies[testIx])
			}

			h.reset()
		})
	}
}
