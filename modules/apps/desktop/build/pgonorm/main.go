// Command pgonorm rewrites the file names of a pprof profile so that none of them names the
// machine that collected it. Usage: pgonorm <root> <in> <out>, where root is the repository root.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/pprof/profile"
)

var (
	goroot   = regexp.MustCompile(`^.*/(?:go|goroot|golang\.org/toolchain@[^/]+)/src/(.*)$`)
	modCache = regexp.MustCompile(`^.*/pkg/mod/(.*)$`)
	buildDir = regexp.MustCompile(`^.*/go-build[0-9]*/(?:b[0-9]+/)?`)
	storeLib = regexp.MustCompile(`^/nix/store/[a-z0-9]+-[^/]*/(.*)$`)
	homeLike = regexp.MustCompile(`/home/|/Users/|/tmp/|/var/folders/|[A-Za-z]:\\`)
)

func normalize(root, name string) string {
	name = filepath.ToSlash(name)
	if rest, ok := strings.CutPrefix(name, root+"/"); ok {
		return rest
	}
	if m := goroot.FindStringSubmatch(name); m != nil {
		return "$GOROOT/src/" + m[1]
	}
	if m := modCache.FindStringSubmatch(name); m != nil {
		return "$GOMODCACHE/" + m[1]
	}
	if i := strings.Index(name, "/modules/"); i >= 0 && homeLike.MatchString(name) {
		return name[i+1:]
	}
	if buildDir.MatchString(name) {
		return buildDir.ReplaceAllString(name, "")
	}
	if m := storeLib.FindStringSubmatch(name); m != nil {
		return m[1]
	}
	return name
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: pgonorm <root> <in> <out>")
		os.Exit(2)
	}
	root := strings.TrimSuffix(filepath.ToSlash(os.Args[1]), "/")
	in, err := os.Open(os.Args[2])
	if err != nil {
		fail(err)
	}
	p, err := profile.Parse(in)
	in.Close()
	if err != nil {
		fail(err)
	}
	for _, m := range p.Mapping {
		m.File = normalize(root, m.File)
	}
	for _, f := range p.Function {
		f.Filename = normalize(root, f.Filename)
	}
	p.Comments = nil
	p.DropFrames = normalize(root, p.DropFrames)
	p.KeepFrames = normalize(root, p.KeepFrames)
	out, err := os.Create(os.Args[3])
	if err != nil {
		fail(err)
	}
	if err := p.Write(out); err != nil {
		fail(err)
	}
	if err := out.Close(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "pgonorm:", err)
	os.Exit(1)
}
