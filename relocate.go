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
		return "", nil
	}
	return fmt.Sprintf("%s[`%s`^]", fileURL, path.Base(file)), nil
}

func (r *relocator) DoFunction(file string, function string) (replace string, err error) {
	fileURL, err := r.fileURL(file)
	if err != nil {
		return "", err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path.Join(r.goDir, file), nil, 0)
	if err != nil {
		return "", err
	}

	for _, decl := range f.Decls {
		switch decl.(type) {
		case *ast.FuncDecl:
			name := decl.(*ast.FuncDecl).Name.Name
			if name != function {
				continue
			}
			pos := fset.Position(decl.Pos())
			return fmt.Sprintf("%s#L%d[`%s`^]", fileURL, pos.Line, function), nil
		}
	}

	return "", fmt.Errorf("couldn't find function %s in file %s", function, file)
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
