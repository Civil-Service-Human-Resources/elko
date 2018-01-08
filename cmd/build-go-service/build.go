// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tav/golly/log"
	"github.com/tav/golly/process"
)

var packageMap = map[string]*Package{}

type Package struct {
	Files   []string
	Imports []string
	Name    string
}

func Build(dir string, dest string, goimports string) {

	if !strings.HasSuffix(dir, string(os.PathSeparator)) {
		dir += string(os.PathSeparator)
	}

	i, err := os.Stat(dir)
	if err != nil {
		log.Fatal(err)
	}

	if !i.IsDir() {
		log.Fatalf("The given path is not a directory: %s", dir)
	}

	log.Infof(">> Parsing go files in %s", dir)

	err = FindPackages(dir, dir)
	if err != nil {
		log.Fatal(err)
	}

	if len(packageMap) == 0 {
		log.Fatalf("No go packages found in: %s", dir)
	}

	compiled := map[string]bool{}
	notFound := map[string]struct{}{}
	packages := []*Package{}

	all := []string{}
	seen := map[string]bool{
		"github.com/tav/elko/pkg/elko": true,
	}

	for _, p := range packageMap {
		imports := []string{}
	nextImport:
		for _, imp := range p.Imports {
			if compiled[imp] {
				continue
			}
			for idx, path := range pkgPaths {
				path = filepath.Join(path, imp) + ".a"
				i, err = os.Stat(path)
				if err == nil && !i.IsDir() {
					compiled[imp] = true
					if !seen[imp] && idx == 1 {
						seen[imp] = true
						all = append(all, imp)
					}
					continue nextImport
				}
			}
			if !seen[imp] {
				seen[imp] = true
				all = append(all, imp)
			}
			if _, exists := packageMap[imp]; exists {
				imports = append(imports, imp)
			} else {
				notFound[imp] = struct{}{}
			}
		}
		p.Imports = imports
		packages = append(packages, p)
	}

	imports := []string{}
	if len(notFound) != 0 {
		log.Info(">> Downloading dependencies")
		for pkg := range notFound {
			log.Infof(">> Running: go get -v %s", pkg)
			Run(GO, []string{"go", "get", "-v", pkg})
			imports = append(imports, pkg)
		}
	}

	if goimports == "" {
		goimports = filepath.Join(dir, ".goimports")
	}

	SaveImports(all, goimports)

	moved := map[string]bool{}
	sorted := packages

	for len(packages) > 0 {
		j := 0
	nextPackage:
		for i, p := range packages {
			for _, imp := range p.Imports {
				if !moved[imp] {
					continue nextPackage
				}
			}
			moved[p.Name] = true
			if i != j {
				packages[i], packages[j] = packages[j], p
			}
			j += 1
		}
		if j == 0 {
			log.Error("Invalid and potentially cyclical imports found:")
			for _, p := range packages {
				log.Error("")
				log.Errorf("  Package %q imports:", p.Name)
				for _, imp := range p.Imports {
					log.Errorf("    %q", imp)
				}
			}
			log.Error("")
			process.Exit(1)
		}
		packages = packages[j:]
	}

	bin := &bytes.Buffer{}
	bin.Write([]byte(`package main

import "github.com/tav/elko/pkg/elko"

`))

	pkgPath := pkgPaths[1]
	for _, p := range sorted {
		log.Infof(">> Compiling package: %s", p.Name)
		archive := filepath.Join(pkgPath, p.Name) + ".a"
		dir, _ := filepath.Split(archive)
		err = os.MkdirAll(dir, 0750)
		if err != nil {
			log.Fatal(err)
		}
		args := []string{
			compilerName, "-I", pkgPath, "-o", archive, "-pack",
		}
		for _, f := range p.Files {
			args = append(args, f)
		}
		err = Run(compilerPath, args)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(bin, "import _ %q\n", p.Name)
	}

	bin.Write([]byte(`

func main() {
    elko.Run()
}
`))

	err = ioutil.WriteFile("/tmp/main.go", bin.Bytes(), 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof(">> Compiling binary: %s", dest)
	args := []string{
		compilerName, "-I", pkgPath, "-o", "/tmp/main.6", "/tmp/main.go",
	}

	err = Run(compilerPath, args)
	if err != nil {
		log.Fatal(err)
	}

	args = []string{
		linkerName, "-L", pkgPath, "-o", dest, "-s", "-w", "/tmp/main.6",
	}

	err = Run(linkerPath, args)
	if err != nil {
		log.Fatal(err)
	}

	os.Remove("/tmp/main.go")
	os.Remove("/tmp/main.6")

}

func FindPackages(root, path string) error {
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	files := []string{}
	for _, f := range dir {
		if f.IsDir() {
			err := FindPackages(root, filepath.Join(path, f.Name()))
			if err != nil {
				return err
			}
		} else {
			if strings.HasSuffix(f.Name(), ".go") {
				files = append(files, filepath.Join(path, f.Name()))
			}
		}
	}
	if len(files) == 0 {
		return nil
	}
	name := path[len(root):]
	fset := token.NewFileSet()
	seen := map[string]struct{}{}
	pkgName := ""
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		if name == "" {
			name = f.Name.Name
			pkgName = name
		} else if pkgName == "" {
			pkgName = f.Name.Name
		} else if f.Name.Name != pkgName {
			return fmt.Errorf("Conflicting package names in %s: %q and %q", path, pkgName, f.Name.Name)
		}
		if name == "main" {
			log.Infof("Skipping directory since main package found: %s", path)
			return nil
		}
		for _, imp := range f.Imports {
			impPath := imp.Path.Value
			if len(impPath) <= 2 || impPath[0] != '"' || impPath[len(impPath)-1] != '"' {
				return fmt.Errorf("Invalid import path in %s: %q", path, impPath)
			}
			seen[impPath[1:len(impPath)-1]] = struct{}{}
		}
	}
	p := &Package{
		Files:   files,
		Imports: []string{},
		Name:    name,
	}
	for imp := range seen {
		p.Imports = append(p.Imports, imp)
	}
	packageMap[name] = p
	return nil
}

func Run(path string, args []string) error {
	cmd := &exec.Cmd{
		Args:   args,
		Path:   path,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cmd.Run()
}
