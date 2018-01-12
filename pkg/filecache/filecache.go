// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package filecache

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tav/golly/log"
)

var ErrNothingChanged = errors.New("filecache: no files changed")

var (
	Changed         = map[string]int{}
	Digest          = []byte{}
	Files           = map[string]*File{}
	Index           = 0
	ParentDirectory = ""
	PathMap         = map[string]FileList{}
	SymlinkScope    = ""
	Tracked         = map[string]int{}
)

type File struct {
	Data    []byte    `json:"data"`
	Digest  [64]byte  `json:"digest"`
	Mode    int64     `json:"mode"`
	Name    string    `json:"name"`
	Index   int       `json:"-"`
	ModTime time.Time `json:"mtime"`
}

type FileList []*File

func (l FileList) Digest() []byte {
	sort.Sort(l)
	s := sha512.New()
	for _, f := range l {
		s.Write([]byte(f.Name))
		s.Write([]byte{'\n'})
		s.Write(f.Digest[:])
		s.Write([]byte{'\n'})
	}
	return s.Sum(nil)
}

func (l FileList) Len() int           { return len(l) }
func (l FileList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l FileList) Less(i, j int) bool { return l[i].Name < l[j].Name }

func ChangedSince(index int, paths ...string) bool {
	for _, path := range paths {
		mindex, exists := Changed[path]
		if !exists {
			return true
		}
		if mindex > index {
			return true
		}
	}
	return false
}

func Exists(path string) bool {
	_, exists := Files[path]
	return exists
}

func Read(path string) []byte {
	Tracked[path] = Changed[path]
	if f, exists := Files[path]; exists {
		return f.Data
	}
	log.Fatalf("Couldn't read: %s", path)
	return nil
}

func ResetTracking() {
	Tracked = map[string]int{}
}

func Track(path string) {
	Tracked[path] = Changed[path]
}

func TrackingData() (int, []string) {
	max := 0
	paths := []string{}
	for path, index := range Tracked {
		paths = append(paths, path)
		if index > max {
			max = index
		}
	}
	ResetTracking()
	return max, paths
}

func Update(paths ...string) error {
	var (
		changed bool
		ichange bool
		file    *File
		files   []*File
	)
	Digest = nil
	PathMap = map[string]FileList{}
	Index += 1
	l := FileList{}
	idx := Index
	for _, path := range paths {
		i, err := os.Stat(path)
		if err != nil {
			return err
		}
		if i.IsDir() {
			ichange, files, err = updateDir(path)
			if err != nil {
				return err
			}
			l = append(l, files...)
		} else {
			ichange, file, err = updateFile(path, i)
			if err != nil {
				return err
			}
			l = append(l, file)
		}
		if ichange {
			changed = true
			Changed[path] = idx
		}
	}
	index := Index
	for path, file := range Files {
		if file.Index != index {
			delete(Changed, path)
			delete(Files, path)
			changed = true
			path = filepath.Dir(path)
			for path != "." {
				Changed[path] = idx
				path = filepath.Dir(path)
			}
		}
	}
	Digest = l.Digest()
	if !changed {
		return ErrNothingChanged
	}
	return nil
}

func updateDir(path string) (bool, []*File, error) {
	var (
		changed  bool
		file     *File
		files    []*File
		ichanged bool
		name     string
	)
	listing, err := ioutil.ReadDir(path)
	if err != nil {
		return changed, nil, err
	}
	l := FileList{}
	for _, i := range listing {
		name = i.Name()
		if i.Mode()&os.ModeSymlink != 0 {
			i, err = os.Stat(filepath.Join(path, name))
			if err != nil {
				return changed, nil, err
			}
		}
		if i.IsDir() {
			ichanged, files, err = updateDir(filepath.Join(path, name))
			if err != nil {
				return changed, nil, err
			}
			l = append(l, files...)
		} else {
			ichanged, file, err = updateFile(filepath.Join(path, name), i)
			if err != nil {
				return changed, nil, err
			}
			l = append(l, file)
		}
		if ichanged && !changed {
			Changed[path] = Index
			changed = true
		}
	}
	PathMap[path] = l
	return changed, l, nil
}

func updateFile(path string, i os.FileInfo) (bool, *File, error) {
	f, exists := Files[path]
	if exists {
		if i.ModTime() == f.ModTime {
			f.Index = Index
			return false, f, nil
		}
	} else {
		f = &File{}
		Files[path] = f
	}
	Changed[path] = Index
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return true, nil, err
	}
	f.Index = Index
	f.Data = data
	f.Digest = sha512.Sum512(data)
	f.Mode = int64(i.Mode().Perm())
	f.ModTime = i.ModTime()
	f.Name = path
	real, err := filepath.EvalSymlinks(filepath.Join(ParentDirectory, path))
	if err != nil {
		return true, nil, err
	}
	if !strings.HasPrefix(real, SymlinkScope) {
		return true, nil, fmt.Errorf(
			"elko: path %s is symlinked to outside the ELKO_SYMLINK_SCOPE (%s): %s",
			path, SymlinkScope, real)
	}
	return true, f, nil
}
