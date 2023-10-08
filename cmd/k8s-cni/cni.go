// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build go1.21

// The tailscaled program is the Tailscale client daemon. It's configured
// and controlled via the tailscale CLI program.
//
// It primarily supports Linux, though other systems will likely be
// supported in the future.
package main // import "tailscale.com/cmd/tailscaled"

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"tailscale.com/control/controlclient"
	"tailscale.com/envknob"
	"tailscale.com/ipn/ipnlocal"
	"tailscale.com/ipn/ipnserver"
	"tailscale.com/ipn/store"
	"tailscale.com/logpolicy"
	"tailscale.com/logtail"
	"tailscale.com/net/dns"
	"tailscale.com/net/dnsfallback"
	"tailscale.com/net/netmon"
	"tailscale.com/net/netns"
	"tailscale.com/net/tsdial"
	"tailscale.com/net/tstun"
	"tailscale.com/paths"
	"tailscale.com/safesocket"
	"tailscale.com/syncs"
	"tailscale.com/tsd"
	"tailscale.com/tsweb/varz"
	"tailscale.com/types/flagtype"
	"tailscale.com/types/logger"
	"tailscale.com/types/logid"
	"tailscale.com/util/clientmetric"
	"tailscale.com/util/multierr"
	"tailscale.com/util/osshare"
	"tailscale.com/version"
	"tailscale.com/wgengine"
	"tailscale.com/wgengine/router"
)

// defaultPort returns the default UDP port to listen on for disco+wireguard.
// By default it returns 0, to pick one randomly from the kernel.
// If the environment variable PORT is set, that's used instead.
// The PORT environment variable is chosen to match what the Linux systemd
// unit uses, to make documentation more consistent.
func defaultPort() uint16 {
	if s := envknob.String("PORT"); s != "" {
		if p, err := strconv.ParseUint(s, 10, 16); err == nil {
			return uint16(p)
		}
	}
	if envknob.GOOS() == "windows" {
		return 41641
	}
	return 0
}

var args struct {
	// tunname is a /dev/net/tun tunnel name ("tailscale0"), the
	// string "userspace-networking", "tap:TAPNAME[:BRIDGENAME]"
	// or comma-separated list thereof.
	tunname string

	cleanup        bool
	debug          string
	port           uint16
	statepath      string
	statedir       string
	socketpath     string
	birdSocketPath string
	verbose        int
	socksAddr      string // listen address for SOCKS5 server
	httpProxyAddr  string // listen address for HTTP proxy server
	disableLogs    bool
}

var (
	installSystemDaemon   func([]string) error                      // non-nil on some platforms
	uninstallSystemDaemon func([]string) error                      // non-nil on some platforms
	createBIRDClient      func(string) (wgengine.BIRDClient, error) // non-nil on some platforms
)

var subCommands = map[string]*func([]string) error{
	"install-system-daemon":   &installSystemDaemon,
	"uninstall-system-daemon": &uninstallSystemDaemon,
}

var beCLI func() // non-nil if CLI is linked in

func main() {
	envknob.PanicIfAnyEnvCheckedInInit()
	envknob.ApplyDiskConfig()

	printVersion := false
	flag.IntVar(&args.verbose, "verbose", 0, "log verbosity level; 0 is default, 1 or higher are increasingly verbose")
	flag.BoolVar(&args.cleanup, "cleanup", false, "clean up system state and exit")
	flag.StringVar(&args.debug, "debug", "", "listen address ([ip]:port) of optional debug server")
	flag.StringVar(&args.socksAddr, "socks5-server", "", `optional [ip]:port to run a SOCK5 server (e.g. "localhost:1080")`)
	flag.StringVar(&args.httpProxyAddr, "outbound-http-proxy-listen", "", `optional [ip]:port to run an outbound HTTP proxy (e.g. "localhost:8080")`)
	flag.StringVar(&args.tunname, "tun", "tailscale0", `tunnel interface name; use "userspace-networking" (beta) to not use TUN`)
	flag.Var(flagtype.PortValue(&args.port, defaultPort()), "port", "UDP port to listen on for WireGuard and peer-to-peer traffic; 0 means automatically select")
	flag.StringVar(&args.statepath, "state", "", "absolute path of state file; use 'kube:<secret-name>' to use Kubernetes secrets or 'arn:aws:ssm:...' to store in AWS SSM; use 'mem:' to not store state and register as an ephemeral node. If empty and --statedir is provided, the default is <statedir>/tailscaled.state. Default: "+paths.DefaultTailscaledStateFile())
	flag.StringVar(&args.statedir, "statedir", "", "path to directory for storage of config state, TLS certs, temporary incoming Taildrop files, etc. If empty, it's derived from --state when possible.")
	flag.StringVar(&args.socketpath, "socket", paths.DefaultTailscaledSocket(), "path of the service unix socket")
	flag.StringVar(&args.birdSocketPath, "bird-socket", "", "path of the bird unix socket")
	flag.BoolVar(&printVersion, "version", false, "print version information and exit")
	flag.BoolVar(&args.disableLogs, "no-logs-no-support", false, "disable log uploads; this also disables any technical support")

	if len(os.Args) > 0 && filepath.Base(os.Args[0]) == "tailscale" && beCLI != nil {
		beCLI()
		return
	}

	if len(os.Args) > 1 {
		sub := os.Args[1]
		if fp, ok := subCommands[sub]; ok {
			if *fp == nil {
				log.SetFlags(0)
				log.Fatalf("%s not available on %v", sub, runtime.GOOS)
			}
			if err := (*fp)(os.Args[2:]); err != nil {
				log.SetFlags(0)
				log.Fatal(err)
			}
			return
		}
	}

	flag.Parse()
	if flag.NArg() > 0 {
		// Windows subprocess is spawned with /subprocess, so we need to avoid this check there.
		if runtime.GOOS != "windows" || (flag.Arg(0) != "/subproc" && flag.Arg(0) != "/firewall") {
			log.Fatalf("tailscaled does not take non-flag arguments: %q", flag.Args())
		}
	}

	if fd, ok := envknob.LookupInt("TS_PARENT_DEATH_FD"); ok && fd > 2 {
		go dieOnPipeReadErrorOfFD(fd)
	}

	if printVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	if runtime.GOOS == "darwin" && os.Getuid() != 0 && !strings.Contains(args.tunname, "userspace-networking") && !args.cleanup {
		log.SetFlags(0)
		log.Fatalf("tailscaled requires root; use sudo tailscaled (or use --tun=userspace-networking)")
	}

	if args.socketpath == "" && runtime.GOOS != "windows" {
		log.SetFlags(0)
		log.Fatalf("--socket is required")
	}

	if args.birdSocketPath != "" && createBIRDClient == nil {
		log.SetFlags(0)
		log.Fatalf("--bird-socket is not supported on %s", runtime.GOOS)
	}

	// Only apply a default statepath when neither have been provided, so that a
	// user may specify only --statedir if they wish.
	if args.statepath == "" && args.statedir == "" {
		args.statepath = paths.DefaultTailscaledStateFile()
	}

	if args.disableLogs {
		envknob.SetNoLogsNoSupport()
	}

	err := run()

	// Remove file sharing from Windows shell (noop in non-windows)
	osshare.SetFileSharingEnabled(false, logger.Discard)

	if err != nil {
		log.Fatal(err)
	}
}

func statePathOrDefault() string {
	if args.statepath != "" {
		return args.statepath
	}
	if args.statedir != "" {
		return filepath.Join(args.statedir, "tailscaled.state")
	}
	return ""
}

// serverOptions is the configuration of the Tailscale node agent.
type serverOptions struct {
	// VarRoot is the Tailscale daemon's private writable
	// directory (usually "/var/lib/tailscale" on Linux) that
	// contains the "tailscaled.state" file, the "certs" directory
	// for TLS certs, and the "files" directory for incoming
	// Taildrop files before they're moved to a user directory.
	// If empty, Taildrop and TLS certs don't function.
	VarRoot string

	// LoginFlags specifies the LoginFlags to pass to the client.
	LoginFlags controlclient.LoginFlags
}

func ipnServerOpts() (o serverOptions) {
	o.VarRoot = args.statedir

	// If an absolute --state is provided but not --statedir, try to derive
	// a state directory.
	if o.VarRoot == "" && filepath.IsAbs(args.statepath) {
		if dir := filepath.Dir(args.statepath); strings.EqualFold(filepath.Base(dir), "tailscale") {
			o.VarRoot = dir
		}
	}
	if strings.HasPrefix(statePathOrDefault(), "mem:") {
		// Register as an ephemeral node.
		o.LoginFlags = controlclient.LoginEphemeral
	}

	return o
}

var logPol *logpolicy.Policy
var debugMux *http.ServeMux

func run() error {
	var logf logger.Logf = log.Printf

	sys := new(tsd.System)

	netMon, err := netmon.New(func(format string, args ...any) {
		logf(format, args...)
	})
	if err != nil {
		return fmt.Errorf("netmon.New: %w", err)
	}
	sys.Set(netMon)

	pol := logpolicy.New(logtail.CollectionNode, netMon, nil /* use log.Printf */)
	pol.SetVerbosityLevel(args.verbose)
	logPol = pol
	defer func() {
		// Finish uploading logs after closing everything else.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		pol.Shutdown(ctx)
	}()

	if err := envknob.ApplyDiskConfigError(); err != nil {
		log.Printf("Error reading environment config: %v", err)
	}

	if envknob.Bool("TS_DEBUG_MEMORY") {
		logf = logger.RusagePrefixLog(logf)
	}
	logf = logger.RateLimitedFn(logf, 5*time.Second, 5, 100)

	if args.cleanup {
		if envknob.Bool("TS_PLEASE_PANIC") {
			panic("TS_PLEASE_PANIC asked us to panic")
		}
		dns.Cleanup(logf, args.tunname)
		router.Cleanup(logf, args.tunname)
		return nil
	}

	if args.statepath == "" && args.statedir == "" {
		log.Fatalf("--statedir (or at least --state) is required")
	}

	if args.debug != "" {
		debugMux = newDebugMux()
	}

	return startIPNServer(context.Background(), logf, pol.PublicID, sys)
}

var sigPipe os.Signal // set by sigpipe.go

func startIPNServer(ctx context.Context, logf logger.Logf, logID logid.PublicID, sys *tsd.System) error {
	ln, err := safesocket.Listen(args.socketpath)
	if err != nil {
		return fmt.Errorf("safesocket.Listen: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Exit gracefully by cancelling the ipnserver context in most common cases:
	// interrupted from the TTY or killed by a service manager.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	// SIGPIPE sometimes gets generated when CLIs disconnect from
	// tailscaled. The default action is to terminate the process, we
	// want to keep running.
	if sigPipe != nil {
		signal.Ignore(sigPipe)
	}
	go func() {
		select {
		case s := <-interrupt:
			logf("tailscaled got signal %v; shutting down", s)
			cancel()
		case <-ctx.Done():
			// continue
		}
	}()

	srv := ipnserver.New(logf, logID, sys.NetMon.Get())
	if debugMux != nil {
		debugMux.HandleFunc("/debug/ipn", srv.ServeHTMLStatus)
	}
	var lbErr syncs.AtomicValue[error]

	go func() {
		t0 := time.Now()
		if s, ok := envknob.LookupInt("TS_DEBUG_BACKEND_DELAY_SEC"); ok {
			d := time.Duration(s) * time.Second
			logf("sleeping %v before starting backend...", d)
			select {
			case <-time.After(d):
				logf("slept %v; starting backend...", d)
			case <-ctx.Done():
				return
			}
		}
		lb, err := getLocalBackend(ctx, logf, logID, sys)
		if err == nil {
			logf("got LocalBackend in %v", time.Since(t0).Round(time.Millisecond))
			srv.SetLocalBackend(lb)
			return
		}
		lbErr.Store(err) // before the following cancel
		cancel()         // make srv.Run below complete
	}()

	err = srv.Run(ctx, ln)

	if err != nil && lbErr.Load() != nil {
		return fmt.Errorf("getLocalBackend error: %v", lbErr.Load())
	}

	// Cancelation is not an error: it is the only way to stop ipnserver.
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("ipnserver.Run: %w", err)
	}

	return nil
}

func getLocalBackend(ctx context.Context, logf logger.Logf, logID logid.PublicID, sys *tsd.System) (_ *ipnlocal.LocalBackend, retErr error) {
	if logPol != nil {
		logPol.Logtail.SetNetMon(sys.NetMon.Get())
	}

	dialer := &tsdial.Dialer{Logf: logf} // mutated below (before used)
	sys.Set(dialer)

	_, err := createEngine(logf, sys)
	if err != nil {
		return nil, fmt.Errorf("createEngine: %w", err)
	}
	if debugMux != nil {
		if ms, ok := sys.MagicSock.GetOK(); ok {
			debugMux.HandleFunc("/debug/magicsock", ms.ServeHTTPDebug)
		}
		go runDebugServer(debugMux, args.debug)
	}

	opts := ipnServerOpts()

	store, err := store.New(logf, statePathOrDefault())
	if err != nil {
		return nil, fmt.Errorf("store.New: %w", err)
	}
	sys.Set(store)

	lb, err := ipnlocal.NewLocalBackend(logf, logID, sys, opts.LoginFlags)
	if err != nil {
		return nil, fmt.Errorf("ipnlocal.NewLocalBackend: %w", err)
	}
	lb.SetVarRoot(opts.VarRoot)
	if logPol != nil {
		lb.SetLogFlusher(logPol.Logtail.StartFlush)
	}
	if root := lb.TailscaleVarRoot(); root != "" {
		dnsfallback.SetCachePath(filepath.Join(root, "derpmap.cached.json"), logf)
	}
	return lb, nil
}

// createEngine tries to the wgengine.Engine based on the order of tunnels
// specified in the command line flags.
//
// onlyNetstack is true if the user has explicitly requested that we use netstack
// for all networking.
func createEngine(logf logger.Logf, sys *tsd.System) (onlyNetstack bool, err error) {
	if args.tunname == "" {
		return false, errors.New("no --tun value specified")
	}
	var errs []error
	for _, name := range strings.Split(args.tunname, ",") {
		logf("wgengine.NewUserspaceEngine(tun %q) ...", name)
		onlyNetstack, err = tryEngine(logf, sys, name)
		if err == nil {
			return onlyNetstack, nil
		}
		logf("wgengine.NewUserspaceEngine(tun %q) error: %v", name, err)
		errs = append(errs, err)
	}
	return false, multierr.New(errs...)
}

type cni_router struct {
	router.Router
	namsSpace string
}

func (r *cni_router) Up() error {
	defer func() {
		// change namespace back to the original one
	}()
	return r.Router.Up()
}
func (r *cni_router) Set(cfg *router.Config) error {
	defer func() {
		// change namespace back to the original one
	}()
	return r.Router.Set(cfg)
}

var tstunNew = tstun.New

func tryEngine(logf logger.Logf, sys *tsd.System, name string) (onlyNetstack bool, err error) {
	conf := wgengine.Config{
		ListenPort:   args.port,
		NetMon:       sys.NetMon.Get(),
		Dialer:       sys.Dialer.Get(),
		SetSubsystem: sys.Set,
		ControlKnobs: sys.ControlKnobs(),
	}

	onlyNetstack = false
	netstackSubnetRouter := onlyNetstack // but mutated later on some platforms
	netns.SetEnabled(!onlyNetstack)

	if args.birdSocketPath != "" && createBIRDClient != nil {
		log.Printf("Connecting to BIRD at %s ...", args.birdSocketPath)
		conf.BIRDClient, err = createBIRDClient(args.birdSocketPath)
		if err != nil {
			return false, fmt.Errorf("createBIRDClient: %w", err)
		}
	}

	dev, devName, err := tstunNew(logf, name)
	if err != nil {
		tstun.Diagnose(logf, name, err)
		return false, fmt.Errorf("tstun.New(%q): %w", name, err)
	}
	conf.Tun = dev
	if strings.HasPrefix(name, "tap:") {
		conf.IsTAP = true
		e, err := wgengine.NewUserspaceEngine(logf, conf)
		if err != nil {
			return false, err
		}
		sys.Set(e)
		return false, err
	}

	r, err := router.New(logf, dev, sys.NetMon.Get())
	if err != nil {
		dev.Close()
		return false, fmt.Errorf("creating router: %w", err)
	}

	d, err := dns.NewOSConfigurator(logf, devName)
	if err != nil {
		dev.Close()
		r.Close()
		return false, fmt.Errorf("dns.NewOSConfigurator: %w", err)
	}
	conf.DNS = d
	conf.Router = r
	sys.Set(conf.Router)

	e, err := wgengine.NewUserspaceEngine(logf, conf)
	if err != nil {
		return onlyNetstack, err
	}
	e = wgengine.NewWatchdog(e)
	sys.Set(e)
	sys.NetstackRouter.Set(netstackSubnetRouter)

	return onlyNetstack, nil
}

func newDebugMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/metrics", servePrometheusMetrics)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

func servePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	varz.Handler(w, r)
	clientmetric.WritePrometheusExpositionFormat(w)
}

func runDebugServer(mux *http.ServeMux, addr string) {
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// dieOnPipeReadErrorOfFD reads from the pipe named by fd and exit the process
// when the pipe becomes readable. We use this in tests as a somewhat more
// portable mechanism for the Linux PR_SET_PDEATHSIG, which we wish existed on
// macOS. This helps us clean up straggler tailscaled processes when the parent
// test driver dies unexpectedly.
func dieOnPipeReadErrorOfFD(fd int) {
	f := os.NewFile(uintptr(fd), "TS_PARENT_DEATH_FD")
	f.Read(make([]byte, 1))
	os.Exit(1)
}
