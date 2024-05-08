package main

import (
	"os"
	"slices"
	"strings"

	"github.com/dave/dst"
	"github.com/pijng/moonject"
)

const IFDEF_GOOS = "#ifdef GOOS:"
const ENDIF = "#endif"
const ELSE = "#else"

func main() {
	goos := os.Getenv("GOOS")
	ifdefModifier := IfdefModifier{GOOS: goos}

	moonject.Process(ifdefModifier)
}

type IfdefModifier struct {
	GOOS string
}

type ifdefStmt struct {
	goos    string
	stmtIdx int
}

type elseStmt struct {
	ifdefIdx int
	stmtIdx  int
}

func (ifm IfdefModifier) Modify(f *dst.File) *dst.File {
	for _, decl := range f.Decls {
		fDecl, isFunc := decl.(*dst.FuncDecl)
		if !isFunc {
			continue
		}

		ifdefStmts := make([]ifdefStmt, 0)
		elseStmts := make([]elseStmt, 0)
		endifStmts := make([]int, 0)
		var latestIfdef int

		for stmtIdx, fStmt := range fDecl.Body.List {
			ifdefGOOS, found := extractIfdefGOOS(fStmt.Decorations().Start.All())
			if found {
				ifdefStmts = append(ifdefStmts, ifdefStmt{goos: ifdefGOOS, stmtIdx: stmtIdx})
				latestIfdef = stmtIdx
			}

			if hasElse(fStmt.Decorations().Start.All()) {
				elseStmts = append(elseStmts, elseStmt{ifdefIdx: latestIfdef, stmtIdx: stmtIdx})
			}

			if hasEndif(fStmt.Decorations().End.All()) {
				endifStmts = append(endifStmts, stmtIdx)
			}

		}

		for idx := len(ifdefStmts) - 1; idx >= 0; idx-- {
			ifdef := ifdefStmts[idx]
			elseStmt, elseFound := correspondingElseStmt(elseStmts, ifdef.stmtIdx)

			if ifdef.goos == ifm.GOOS && !elseFound {
				continue
			}

			var modifiedBodyList []dst.Stmt

			startIdx := ifdef.stmtIdx
			endIdx := endifStmts[idx] + 1

			if elseFound {
				if ifdef.goos == ifm.GOOS {
					modifiedBodyList = append(fDecl.Body.List[:elseStmt.stmtIdx], fDecl.Body.List[endIdx:]...)
				} else {
					modifiedBodyList = append(fDecl.Body.List[:startIdx], fDecl.Body.List[startIdx+1:elseStmt.stmtIdx+1]...)
					modifiedBodyList = append(modifiedBodyList, fDecl.Body.List[endIdx:]...)
				}
			} else {
				modifiedBodyList = append(fDecl.Body.List[:startIdx], fDecl.Body.List[endIdx:]...)
			}

			fDecl.Body.List = modifiedBodyList
		}
	}

	return f
}

func extractIfdefGOOS(comments []string) (string, bool) {
	var ifdefGOOS string
	var found bool

	for _, commentLine := range comments {
		_, comment, ok := strings.Cut(commentLine, "// ")
		if !ok {
			continue
		}

		_, rawIfdefGOOS, ok := strings.Cut(comment, IFDEF_GOOS)
		if !ok {
			continue
		}

		ifdefGOOS = strings.TrimSpace(rawIfdefGOOS)
		if len(ifdefGOOS) == 0 {
			continue
		}

		found = true
	}

	return ifdefGOOS, found
}

func hasEndif(comments []string) bool {
	var found bool

	for _, commentLine := range comments {
		_, comment, ok := strings.Cut(commentLine, "// ")
		if !ok {
			continue
		}

		_, _, ok = strings.Cut(comment, ENDIF)
		found = ok
	}

	return found
}

func hasElse(comments []string) bool {
	var found bool

	for _, commentLine := range comments {
		_, comment, ok := strings.Cut(commentLine, "// ")
		if !ok {
			continue
		}

		_, _, ok = strings.Cut(comment, ELSE)
		found = ok
	}

	return found
}

func correspondingElseStmt(elseStmts []elseStmt, ifdefStmtIdx int) (elseStmt, bool) {
	elseIdx := slices.IndexFunc(elseStmts, func(e elseStmt) bool { return e.ifdefIdx == ifdefStmtIdx })
	if elseIdx == -1 {
		return elseStmt{}, false
	}

	return elseStmts[elseIdx], true
}
