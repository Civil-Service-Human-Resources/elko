// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tav/golly/log"
	"golang.org/x/tools/go/vcs"
)

var infoMap = map[string]*Info{}

type bySource []*Import

func (l bySource) Len() int {
	return len(l)
}

func (l bySource) Less(i, j int) bool {
	return l[i].Source < l[j].Source
}

func (l bySource) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

type Import struct {
	VCS      string   `json:"vcs"`
	Source   string   `json:"source"`
	Revision string   `json:"revision"`
	Packages []string `json:"packages"`
}

type Info struct {
	Deps       []string
	ImportPath string
	Standard   bool
}

func Install(path string) {
	out, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatal(err)
	}
	if len(out) == 0 {
		return
	}
	imports := []*Import{}
	err = json.Unmarshal(out, &imports)
	if err != nil {
		log.Fatal(err)
	}
	var (
		args []string
		cmd  *exec.Cmd
		pkg  string
		repo *vcs.RepoRoot
	)
	buf := &bytes.Buffer{}
	for _, imp := range imports {
		if imp.Revision == "" {
			continue
		}
		for _, pkg = range imp.Packages {
			log.Infof(">> Running: go get -v %s", pkg)
			Run(GO, []string{"go", "get", "-v", pkg})
		}
		repo, err = vcs.RepoRootForImportPath(imp.Packages[0], false)
		if err != nil {
			log.Fatal(err)
		}
		err = os.Chdir(filepath.Join(srcPath, repo.Root))
		if err != nil {
			log.Fatal(err)
		}
		switch imp.VCS {
		case "git":
			args = []string{"rev-parse", "HEAD"}
		case "hg":
			args = []string{"identify", "--id", "--debug"}
		}
		buf.Reset()
		cmd = exec.Command(imp.VCS, args...)
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
		if strings.TrimSpace(buf.String()) != imp.Revision {
			log.Infof(">> Updating to revision %q", imp.Revision)
			switch imp.VCS {
			case "git":
				args = []string{"checkout", imp.Revision}
			case "hg":
				args = []string{"update", imp.Revision}
			}
			buf.Reset()
			cmd = exec.Command(imp.VCS, args...)
			cmd.Stderr = buf
			err = cmd.Run()
			if err != nil {
				log.Errorf("Running %s returned: \n\n%s", cmd.Args, buf.String())
				log.Fatal(err)
			}
		}
	}
}

func GetInfo(imports []string) {
	args := []string{"go", "list", "-json"}
	args = append(args, imports...)
	buf := &bytes.Buffer{}
	cmd := exec.Cmd{
		Path:   GO,
		Args:   args,
		Stdout: buf,
		Stderr: os.Stderr,
	}
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	dec := json.NewDecoder(buf)
	for {
		info := &Info{}
		err = dec.Decode(info)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		infoMap[info.ImportPath] = info
	}
}

func SaveImports(imports []string, path string) {

	skip := map[string]bool{}

	out, err := ioutil.ReadFile(path)
	if err == nil {
		prevImports := []*Import{}
		err = json.Unmarshal(out, &prevImports)
		if err == nil {
			for _, imp := range prevImports {
				if imp.Revision == "" {
					skip[imp.Source] = true
				}
			}
		}
	}

	if len(imports) == 0 {
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	log.Info(">> Generating .goimports")

	GetInfo(imports)

	imports = []string{}
	seen := map[string]bool{}

	for _, info := range infoMap {
		for _, dep := range info.Deps {
			if seen[dep] {
				continue
			}
			imports = append(imports, dep)
			seen[dep] = true
		}
	}

	GetInfo(imports)

	buf := &bytes.Buffer{}
	imports = []string{}
	repos := map[string]*Import{}

	for pkg, info := range infoMap {
		if info.Standard {
			continue
		}
		repo, err := vcs.RepoRootForImportPath(pkg, false)
		if err != nil {
			log.Fatal(err)
		}
		if imp, exists := repos[repo.Root]; exists {
			imp.Packages = append(imp.Packages, pkg)
		} else {
			var args []string
			switch repo.VCS.Cmd {
			case "git":
				args = []string{"rev-parse", "HEAD"}
			case "hg":
				args = []string{"identify", "--id", "--debug"}
			}
			os.Chdir(filepath.Join(srcPath, repo.Root))
			buf.Reset()
			cmd := exec.Command(repo.VCS.Cmd, args...)
			cmd.Stdout = buf
			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
			imports = append(imports, repo.Root)
			repos[repo.Root] = &Import{
				Packages: []string{pkg},
				VCS:      repo.VCS.Cmd,
				Revision: strings.TrimSpace(buf.String()),
				Source:   repo.Repo,
			}
		}
	}

	final := []*Import{}
	for _, root := range imports {
		imp := repos[root]
		if skip[imp.Source] {
			imp.Revision = ""
		}
		final = append(final, imp)
		sort.Strings(imp.Packages)
	}

	os.Chdir(cwd)

	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}

	if len(final) != 0 {
		sort.Sort(bySource(final))
		out, err := json.MarshalIndent(final, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		f.Write(out)
	}

	f.Close()

	err = os.Chdir(cwd)
	if err != nil {
		log.Fatal(err)
	}

}
