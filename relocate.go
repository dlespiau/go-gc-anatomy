package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

// relocate is a preprocessor replacing special markers pointing to files and
// file-level declarations (const, var and func declarations) by github links.
//
// To do the latter, relocate needs to parse the .go file, find the symbol,
// derive the line number and construct the appropriate link. Because those
// lines can defer differ between revisions, links are tied to a specific git
// hash.
//
// Examples:
//
//     @@src/cmd/compile/internal/gc/inl.go@@ ->
//     https://github.com/golang/go/tree/$commit_hash/src/cmd/compile/internal/gc/inl.go
//
//     @@src/cmd/compile/internal/gc/inl.go:ishairy@@ ->
//     https://github.com/golang/go/tree/$commit_hash/src/cmd/compile/internal/gc/inl.go#L196
//
// To work, relocate needs a local go checkout placed in ./go.

func findHeadCommit(path string) (hash string, err error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type relocator struct {
	goDir   string
	gitHash string
	baseURL string
}

func (r *relocator) fileURL(file string) (url string, err error) {
	if _, err := os.Stat(path.Join(r.goDir, file)); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/tree/%s/%s", r.baseURL, r.gitHash, file), nil
}

func (r *relocator) DoFile(file string) (replace string, err error) {
	fileURL, err := r.fileURL(file)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s[`%s`^]", fileURL, path.Base(file)), nil
}

func (r *relocator) DoFunction(file string, symbol string) (replace string, err error) {
	fileURL, err := r.fileURL(file)
	if err != nil {
		return "", err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path.Join(r.goDir, file), nil, 0)
	if err != nil {
		return "", err
	}

	var node ast.Node
	for _, decl := range f.Decls {
		switch decl.(type) {
		case *ast.FuncDecl:
			name := decl.(*ast.FuncDecl).Name.Name
			if name != symbol {
				continue
			}
			node = decl
			goto out
		case *ast.GenDecl:
			gen := decl.(*ast.GenDecl)
			if gen.Tok != token.CONST && gen.Tok != token.VAR {
				continue
			}

			for _, spec := range gen.Specs {
				for _, ident := range spec.(*ast.ValueSpec).Names {
					if ident.Name != symbol {
						continue
					}
					node = ident
					goto out
				}
			}
		}
	}

out:
	if node != nil {
		pos := fset.Position(node.Pos())
		return fmt.Sprintf("%s#L%d[`%s`^]", fileURL, pos.Line, symbol), nil
	}

	return "", fmt.Errorf("couldn't find symbol %s in file %s", symbol, file)
}

func main() {
	hash, err := findHeadCommit("go/")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(os.Stderr, "relocate: using go commit", hash)

	relocator := &relocator{
		goDir:   "go/",
		gitHash: hash,
		baseURL: "https://github.com/golang/go",
	}

	re := regexp.MustCompile(`@@.*@@`)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		rep := re.ReplaceAllFunc([]byte(line), func(src []byte) []byte {
			var (
				err     error
				replace string
			)

			ref := string(src[2 : len(src)-2])
			index := strings.Index(ref, ":")
			if index == -1 {
				/* @@src/cmd/compile/internal/gc/inl.go@@ */
				replace, err = relocator.DoFile(ref)
			} else {
				/* @@src/cmd/compile/internal/gc/inl.go:caninl@@ */
				replace, err = relocator.DoFunction(ref[:index], ref[index+1:])
			}
			if err != nil {
				log.Fatal(err)
			}
			return []byte(replace)
		})

		fmt.Println(string(rep))
	}
}
