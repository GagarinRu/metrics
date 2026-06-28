// Package osexitanalyzer forbids os.Exit in package main's main function.
package osexitanalyzer

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "osexitanalyzer",
	Doc:  "forbids os.Exit in package main's main function",	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		fname := pass.Fset.File(file.Pos()).Name()
		if strings.Contains(fname, "testdata") || strings.Contains(fname, "go-build") {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" || fn.Body == nil {
				return true
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				ident, ok := sel.X.(*ast.Ident)
				if !ok || ident.Name != "os" || sel.Sel.Name != "Exit" {
					return true
				}

				pass.Reportf(call.Pos(), "do not use os.Exit in main; move the call to a separate function")
				return true
			})

			return false
		})
	}

	return nil, nil
}
