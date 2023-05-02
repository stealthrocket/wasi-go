package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	_ "unsafe" // for go:linktime

	"github.com/stealthrocket/wasi"
	"github.com/stealthrocket/wasi/wasiunix"
	"github.com/stealthrocket/wasi/wasizero"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
)

const Version = "devel"

var (
	envs    strings
	dirs    strings
	version bool
	help    bool
	h       bool
)

func main() {
	flag.Var(&envs, "env", "Environment variables to pass to the WASM module.")
	flag.Var(&dirs, "dir", "Directories to pre-open.")
	flag.BoolVar(&version, "version", false, "Print the version and exit.")
	flag.BoolVar(&help, "help", false, "Print usage information.")
	flag.BoolVar(&h, "h", false, "Print usage information.")
	flag.Parse()

	if version {
		fmt.Println("wasiunix", Version)
		os.Exit(0)
	} else if h || help {
		showUsage()
		os.Exit(0)
	}

	if err := run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Printf(`wasiunix - Run a WebAssembly module

USAGE:
   wasiunix [OPTIONS]... <MODULE> [--] [ARGS]...

ARGS:
   <MODULE>
      The path of the WebAssembly module to run

   [ARGS]...
      Arguments to pass to the module

OPTIONS:
   --dir <DIR>
      Grant access to the specified host directory		

   --env <NAME=VAL>
      Pass an environment variable to the module

   --version
      Print the version and exit

   -h, --help
      Show this usage information
`)
}

func run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: wasiunix [OPTIONS]... <MODULE> [--] [ARGS]...")
	}

	wasmFile := args[0]
	wasmName := filepath.Base(wasmFile)
	wasmCode, err := os.ReadFile(wasmFile)
	if err != nil {
		return fmt.Errorf("could not read WASM file '%s': %w", wasmFile, err)
	}

	args = args[1:]
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	provider := &wasiunix.Provider{
		Args:    append([]string{wasmName}, args...),
		Environ: envs,
		Realtime: func() uint64 {
			now := time.Now()
			sec, nsec := uint64(now.Second()), uint64(now.Nanosecond())
			return sec*1e9 + nsec
		},
		RealtimePrecision:  time.Microsecond,
		Monotonic:          nanotime,
		MonotonicPrecision: time.Nanosecond,
		Rand:               rand.Reader,
		Exit:               os.Exit,
	}

	stdioStat := wasi.FDStat{
		FileType:         wasi.CharacterDeviceType,
		RightsBase:       wasi.AllRights,
		RightsInheriting: wasi.AllRights,
	}
	provider.RegisterFD(syscall.Stdin, "/dev/stdin", stdioStat)
	provider.RegisterFD(syscall.Stdout, "/dev/stdout", stdioStat)
	provider.RegisterFD(syscall.Stderr, "/dev/stderr", stdioStat)

	for _, dir := range dirs {
		fd, err := syscall.Open(dir, syscall.O_DIRECTORY, 0)
		if err != nil {
			return err
		}
		provider.RegisterFD(fd, dir, wasi.FDStat{
			FileType:         wasi.DirectoryType,
			RightsBase:       wasi.AllRights,
			RightsInheriting: wasi.AllRights,
		})
	}

	module := wazergo.MustInstantiate(ctx, runtime, wasizero.HostModule, wasizero.WithWASI(provider))
	ctx = wazergo.WithModuleInstance(ctx, module)

	instance, err := runtime.Instantiate(ctx, wasmCode)
	if err != nil {
		return err
	}
	return instance.Close(ctx)
}

//go:linkname nanotime runtime.nanotime
func nanotime() uint64

type strings []string

func (s strings) String() string {
	return fmt.Sprintf("%v", []string(s))
}

func (s *strings) Set(value string) error {
	*s = append(*s, value)
	return nil
}
