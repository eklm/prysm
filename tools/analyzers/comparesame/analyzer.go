package comparesame

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"

	"github.com/prysmaticlabs/prysm/tools/analyzers"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// Doc explaining the tool.
const Doc = "Tool to detect comparison (==, !=, >=, <=, >, <) of identical boolean expressions."

const messageTemplate = "Boolean expression has identical expressions on both sides. The result is always %v."

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "comparesame",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector, err := analyzers.GetInspector(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{
		(*ast.BinaryExpr)(nil),
	}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		expr, ok := node.(*ast.BinaryExpr)
		if !ok {
			return
		}

		switch expr.Op {
		case token.EQL, token.NEQ, token.GEQ, token.LEQ, token.GTR, token.LSS:
			var xBuf, yBuf bytes.Buffer
			if err := printer.Fprint(&xBuf, pass.Fset, expr.X); err != nil {
				pass.Reportf(expr.X.Pos(), err.Error())
			}
			if err := printer.Fprint(&yBuf, pass.Fset, expr.Y); err != nil {
				pass.Reportf(expr.Y.Pos(), err.Error())
			}
			if xBuf.String() == yBuf.String() {
				switch expr.Op {
				case token.EQL, token.NEQ, token.GEQ, token.LEQ:
					pass.Reportf(expr.OpPos, messageTemplate, true)
				case token.GTR, token.LSS:
					pass.Reportf(expr.OpPos, messageTemplate, false)
				}
			}
		}
	})

	return nil, nil
}
