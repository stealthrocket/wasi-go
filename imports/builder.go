package imports

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/stealthrocket/wasi-go"
	"github.com/stealthrocket/wasi-go/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero"
)

// Builder is used to setup and instantiate the WASI host module.
type Builder struct {
	name               string
	args               []string
	env                []string
	mounts             []mount
	listens            []string
	dials              []string
	customStdio        bool
	stdin              int
	stdout             int
	stderr             int
	realtime           func(context.Context) (uint64, error)
	realtimePrecision  time.Duration
	monotonic          func(context.Context) (uint64, error)
	monotonicPrecision time.Duration
	yield              func(context.Context) error
	exit               func(context.Context, int) error
	raise              func(context.Context, int) error
	rand               io.Reader
	socketsExtension   *wasi_snapshot_preview1.Extension
	pathOpenSockets    bool
	nonBlockingStdio   bool
	tracer             io.Writer
	decorators         []wasi_snapshot_preview1.Decorator
	wrappers           []func(wasi.System) wasi.System
	errors             []error
	maxOpenFiles       int
	maxOpenDirs        int
}

// NewBuilder creates a Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

type mount struct {
	dir  string
	mode int
}

// WithName sets the name of the module, which is exposed to the module
// as argv[0].
func (b *Builder) WithName(name string) *Builder {
	b.name = name
	return b
}

// WithArgs sets command line arguments.
func (b *Builder) WithArgs(args ...string) *Builder {
	b.args = args
	return b
}

// WithEnv sets environment variables.
func (b *Builder) WithEnv(env ...string) *Builder {
	b.env = env
	return b
}

// WithDirs specifies a set of directories to preopen.
//
// The directory can either be a path, or a string of the form "path:path[:ro]"
// for compatibility with wazero's WASI preview 1 host module. Since virtual
// file systems are not supported by this implementation, the two paths must
// be the same when using this syntax. The optional ":ro" prefix means that
// this directory is read-only.
func (b *Builder) WithDirs(dirs ...string) *Builder {
	for _, dir := range dirs {
		mode := int('r' + 'w')
		prefix, readOnly := strings.CutSuffix(dir, ":ro")
		if readOnly {
			mode = 'r'
		}
		parts := strings.Split(prefix, ":")
		switch {
		case len(parts) == 1:
			dir = parts[0]
		case len(parts) == 2 && parts[0] == parts[1]:
			dir = parts[0]
		case len(parts) == 2:
			b.errors = append(b.errors, fmt.Errorf("virtual filesystems are not supported (cannot mount %q)", dir))
		default:
			b.errors = append(b.errors, fmt.Errorf("invalid directory %q", dir))
		}
		b.mounts = append(b.mounts, mount{dir: dir, mode: mode})
	}
	return b
}

// WithListens specifies a list of addresses to listen on before starting
// the module. The listener sockets are added to the set of preopens.
func (b *Builder) WithListens(listens ...string) *Builder {
	b.listens = listens
	return b
}

// WithDials specifies a list of addresses to dial before starting
// the module. The connection sockets are added to the set of preopens.
func (b *Builder) WithDials(dials ...string) *Builder {
	b.dials = dials
	return b
}

// WithStdio sets stdio file descriptors.
//
// Note that the file descriptors will be duplicated before the module takes
// ownership. The caller is responsible for managing the specified
// descriptors.
func (b *Builder) WithStdio(stdin, stdout, stderr int) *Builder {
	b.customStdio = true
	b.stdin = stdin
	b.stdout = stdout
	b.stderr = stderr
	return b
}

// WithRealtimeClock sets the realtime clock and precision.
func (b *Builder) WithRealtimeClock(clock func(context.Context) (uint64, error), precision time.Duration) *Builder {
	b.realtime = clock
	b.realtimePrecision = precision
	return b
}

// WithMonotonicClock sets the monotonic clock and precision.
func (b *Builder) WithMonotonicClock(clock func(context.Context) (uint64, error), precision time.Duration) *Builder {
	b.monotonic = clock
	b.monotonicPrecision = precision
	return b
}

// WithYield sets the sched_yield function.
func (b *Builder) WithYield(fn func(context.Context) error) *Builder {
	b.yield = fn
	return b
}

// WithExit sets the proc_exit function.
func (b *Builder) WithExit(fn func(context.Context, int) error) *Builder {
	b.exit = fn
	return b
}

// WithRaise sets the proc_raise function.
func (b *Builder) WithRaise(fn func(context.Context, int) error) *Builder {
	b.raise = fn
	return b
}

// WithSocketsExtension enables a sockets extension.
//
// The name can be one of:
// - none: disable sockets extensions (use vanilla WASI preview 1)
// - wasmedgev1: use WasmEdge sockets extension version 1
// - wasmedgev2: use WasmEdge sockets extension version 2
// - wasix: use the WASIX sockets extension
// - path_open: use the extension to the path_open system call (unix.PathOpenSockets)
// - auto: attempt to detect one of the extensions above
func (b *Builder) WithSocketsExtension(name string, module wazero.CompiledModule) *Builder {
	switch strings.ToLower(name) {
	case "none", "":
		// no sockets extension
	case "wasmedgev1":
		b.socketsExtension = &wasi_snapshot_preview1.WasmEdgeV1
	case "wasmedgev2":
		b.socketsExtension = &wasi_snapshot_preview1.WasmEdgeV2
	case "wasix":
		b.socketsExtension = &wasi_snapshot_preview1.Wasix
	case "path_open":
		b.socketsExtension = nil
		b.pathOpenSockets = true
	case "auto":
		b.socketsExtension = DetectSocketsExtension(module)
	default:
		b.errors = append(b.errors, fmt.Errorf("invalid socket extension %q", name))
	}
	return b
}

// WithNonBlockingStdio enables or disables non-blocking stdio.
// When enabled, stdio file descriptors will have the O_NONBLOCK flag set
// before the module is started.
func (b *Builder) WithNonBlockingStdio(enable bool) *Builder {
	b.nonBlockingStdio = enable
	return b
}

// WithTracer enables the Tracer, and instructs it to write to the
// specified io.Writer.
func (b *Builder) WithTracer(enable bool, w io.Writer) *Builder {
	if !enable {
		w = nil
	}
	b.tracer = w
	return b
}

// WithDecorators sets the host module decorators.
func (b *Builder) WithDecorators(decorators ...wasi_snapshot_preview1.Decorator) *Builder {
	b.decorators = decorators
	return b
}

// WithWrappers sets the wasi.System wrappers.
func (b *Builder) WithWrappers(wrappers ...func(wasi.System) wasi.System) *Builder {
	b.wrappers = wrappers
	return b
}

// WithMaxOpenFiles sets the limit on the maximum number of files that may be
// opened by the guest module.
func (b *Builder) WithMaxOpenFiles(n int) *Builder {
	b.maxOpenFiles = n
	return b
}

// WithMaxOpenFiles sets the limit on the maximum number of directories that may
// be opened by the guest module.
func (b *Builder) WithMaxOpenDirs(n int) *Builder {
	b.maxOpenDirs = n
	return b
}
