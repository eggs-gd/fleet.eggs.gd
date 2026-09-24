package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/appconfig"
	"github.com/eggs-gd/fleet.eggs.gd/internal/projectscan"
	"github.com/eggs-gd/fleet.eggs.gd/internal/runtimedb"
	"github.com/eggs-gd/fleet.eggs.gd/internal/server"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
)

const version = "0.0.1"

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "rebuild-index":
		rebuildIndex(os.Args[2:])
	case "scan":
		scanProjects(os.Args[2:])
	case "workspaces":
		writeWorkspaces(os.Args[2:])
	case "serve":
		serve(os.Args[2:])
	case "version":
		fmt.Println(version)
	default:
		log.Printf("unknown command: %s", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func scanProjects(args []string) {
	flags := flag.NewFlagSet("scan", flag.ExitOnError)
	root := flags.String("root", "", "Data root (default: saved app config, else ~/.fleet/workspace)")
	var projects stringList
	flags.Var(&projects, "projects", "absolute directory to scan; repeat for several roots")
	flags.Parse(args)

	coreRoot, _, err := appconfig.ResolveDataRoot(*root)
	if err != nil {
		log.Fatal(err)
	}
	if len(projects) == 0 {
		log.Fatal("scan root is required: pass --projects")
	}
	result, err := projectscan.Scan(projectscan.ScanOptions{
		ScanRoots:   projects,
		RegistryDir: filepath.Join(coreRoot, "_registry"),
		WorkDir:     filepath.Join(coreRoot, "Work"),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Discovered %d repositories under %s\n", result.Repositories, strings.Join(result.Roots, ", "))
	fmt.Printf("Found %d deletion review candidates\n", result.DeletionCandidates)
	fmt.Printf("Wrote _registry to %s\n", result.RegistryDir)
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("scan root is empty")
	}
	*s = append(*s, value)
	return nil
}

func writeWorkspaces(args []string) {
	flags := flag.NewFlagSet("workspaces", flag.ExitOnError)
	root := flags.String("root", "", "Data root (default: saved app config, else ~/.fleet/workspace)")
	flags.Parse(args)

	coreRoot, _, err := appconfig.ResolveDataRoot(*root)
	if err != nil {
		log.Fatal(err)
	}
	projects, err := projectscan.WriteWorkspaces(filepath.Join(coreRoot, "_registry"), filepath.Join(coreRoot, "Work"))
	if err != nil {
		log.Fatal(err)
	}
	if err := markdown.RebuildWorkIndex(coreRoot); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Generated %d Work workspaces in %s\n", len(projects), filepath.Join(coreRoot, "Work"))
}

func rebuildIndex(args []string) {
	flags := flag.NewFlagSet("rebuild-index", flag.ExitOnError)
	root := flags.String("root", "", "Data root (default: saved app config, else ~/.fleet_data)")
	flags.Parse(args)

	coreRoot, _, err := appconfig.ResolveDataRoot(*root)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := appconfig.EnsureDataRoot(coreRoot); err != nil {
		log.Fatal(err)
	}
	if err := markdown.RebuildWorkIndex(coreRoot); err != nil {
		log.Fatal(err)
	}
}

func serve(args []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := flags.String("addr", "127.0.0.1:8787", "HTTP listen address")
	root := flags.String("root", "", "Data root (default: saved app config, else ~/.fleet_data)")
	backofficeDir := flags.String("backoffice-dir", "view/dist", "built backoffice static directory")
	dryRun := flags.Bool("dry-run", false, "plan agent launches without starting processes")
	live := flags.Bool("live", false, "deprecated no-op; serve is live by default")
	sessionTimeout := flags.Duration("session-timeout", 10*time.Minute, "idle attention threshold for controllable agent sessions; resets on activity and is not a total-session cap")
	flags.Parse(args)
	if *live {
		fmt.Fprintln(os.Stderr, "warning: --live is deprecated; core serve is live by default")
	}

	coreRoot, rootSource, err := appconfig.ResolveDataRoot(*root)
	if err != nil {
		log.Fatal(err)
	}
	created, err := appconfig.EnsureDataRoot(coreRoot)
	if err != nil {
		log.Fatal(err)
	}
	if created {
		log.Printf("bootstrapped new Data root at %s (source: %s)", coreRoot, rootSource)
	}

	runtimeRoot, err := appconfig.RuntimeRoot()
	if err != nil {
		log.Fatal(err)
	}
	if err := runtimedb.ImportLegacy(runtimeRoot); err != nil {
		log.Fatalf("import runtime data: %v", err)
	}

	// A relative --backoffice-dir resolves against the working directory,
	// like any other relative CLI path — not against --root (Data). App
	// (this binary's own source/install tree, where the built dist/ lives)
	// and Data (--root) are independent trees and are not assumed to be
	// co-located; callers that invoke this from elsewhere must pass an
	// absolute --backoffice-dir (see App/Makefile's `serve` target).
	staticDir, err := filepath.Abs(*backofficeDir)
	if err != nil {
		log.Fatal(err)
	}

	cfg := server.Config{
		Addr:           *addr,
		CoreRoot:       coreRoot,
		DataRootSource: rootSource,
		RuntimeRoot:    runtimeRoot,
		BackofficeDir:  staticDir,
		DryRun:         *dryRun,
		SessionTimeout: *sessionTimeout,
		Version:        version,
	}
	if err := server.Serve(cfg); err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  core rebuild-index [--root <data-root>]")
	fmt.Fprintln(os.Stderr, "  core scan [--root <data-root>] --projects <scan-root> [--projects <scan-root>...]")
	fmt.Fprintln(os.Stderr, "  core workspaces [--root <data-root>]")
	fmt.Fprintln(os.Stderr, "  core serve [--addr 127.0.0.1:8787] [--root <data-root>] [--backoffice-dir view/dist] [--dry-run] [--session-timeout 10m]")
	fmt.Fprintln(os.Stderr, "  core version")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "--root is optional: without it, Core uses the Data root saved in")
	fmt.Fprintln(os.Stderr, "~/.fleet/app.json, defaulting to and creating ~/.fleet/workspace on first run.")
}
