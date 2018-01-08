// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tav/golly/log"
	"github.com/tav/golly/optparse"
	"github.com/tav/golly/process"
)

var (
	gobin    string
	pkgPath  string
	pkgPaths []string
	srcPath  string
)

func initGo() {

	goroot := runtime.GOROOT()
	gobin = filepath.Join(goroot, "bin", "go")

	gopath := os.Getenv("GOPATH")
	paths := strings.Split(gopath, string(os.PathListSeparator))
	if len(paths) == 0 {
		log.Fatalf("Invalid value for the GOPATH environment variable: %q", gopath)
	}

	gopath = paths[0]
	if gopath == "" {
		log.Fatalf("Invalid value for the GOPATH environment variable: %q", gopath)
	}

	osArch := runtime.GOOS + "_" + runtime.GOARCH
	pkgPath = filepath.Join(gopath, "pkg", osArch)
	pkgPaths = []string{
		filepath.Join(goroot, "pkg", osArch), pkgPath,
	}

	srcPath = filepath.Join(gopath, "src")

}

func main() {

	opts := optparse.New("Usage: build-go-service [OPTIONS] PATH\n")

	opts.SetVersion("0.0.1")

	goimports := opts.Flags("-g", "--goimports").Label("FILE").String(
		"Path to the .goimports file")

	installDeps := opts.Flags("-i", "--install-deps").Bool(
		"Install the dependencies specified by the .goimports file")

	output := opts.Flags("-o", "--output").Label("FILE").String(
		"Path to output the generated binary")

	os.Args[0] = "build-go-service"
	args := opts.Parse(os.Args)

	initGo()

	if *installDeps {
		Install(*goimports)
	} else if len(args) == 0 {
		opts.PrintUsage()
		process.Exit(1)
	} else {
		Build(args[0], *output, *goimports)
	}

	process.Exit(0)

}
