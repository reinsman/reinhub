// Command initmodule 重写当前项目中的 Go 模块导入路径。
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: initmodule <old-module> <new-module>")
		os.Exit(2)
	}

	oldModule, newModule := os.Args[1], os.Args[2]
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != "." && (entry.Name() == ".git" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		return rewriteImports(path, oldModule, newModule)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "修改 import 失败:", err)
		os.Exit(1)
	}
}

type replacement struct {
	start int
	end   int
	text  string
}

func rewriteImports(path, oldModule, newModule string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, path, source, parser.ImportsOnly)
	if err != nil {
		return fmt.Errorf("解析 %s: %w", path, err)
	}

	var replacements []replacement
	for _, spec := range parsed.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return fmt.Errorf("解析 %s 中的 import: %w", path, err)
		}
		if importPath != oldModule && !strings.HasPrefix(importPath, oldModule+"/") {
			continue
		}

		start := files.Position(spec.Path.Pos()).Offset
		end := files.Position(spec.Path.End()).Offset
		replacements = append(replacements, replacement{
			start: start,
			end:   end,
			text:  strconv.Quote(newModule + strings.TrimPrefix(importPath, oldModule)),
		})
	}

	if len(replacements) == 0 {
		return nil
	}
	for i := len(replacements) - 1; i >= 0; i-- {
		r := replacements[i]
		source = append(source[:r.start], append([]byte(r.text), source[r.end:]...)...)
	}
	return os.WriteFile(path, source, 0o644)
}
