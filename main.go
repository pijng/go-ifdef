package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dave/dst"
	"github.com/pijng/moonject"
)

const IFDEF_GOOS = "#ifdef GOOS:"
const ENDIF = "#endif"
const ELSE = "#else"
const CUSTOM_DIRECTIVE_PREFIX = "#define "

func main() {
	goos := os.Getenv("GOOS")
	ifdefModifier := IfdefModifier{GOOS: goos}

	moonject.Process(ifdefModifier)
}

type directive struct {
	name      string
	evaluated bool
}

type IfdefModifier struct {
	GOOS string
}

type ifdefStmt struct {
	goos    string
	stmtIdx int
}

type stmt struct {
	ifdefIdx int
	stmtIdx  int
}

func (ifm IfdefModifier) Modify(f *dst.File) *dst.File {
	f = processGoosDirectives(f, ifm.GOOS)
	// res := decorator.NewRestorerWithImports("root", guess.New())
	// res.Print(f)
	f = processCustomDirectives(f)
	// res = decorator.NewRestorerWithImports("root", guess.New())
	// res.Print(f)

	return f
}

func processGoosDirectives(f *dst.File, goos string) *dst.File {
	for _, decl := range f.Decls {
		fDecl, isFunc := decl.(*dst.FuncDecl)
		if !isFunc {
			continue
		}

		ifdefStmts := make([]ifdefStmt, 0)
		elseStmts := make([]stmt, 0)
		endifStmts := make([]stmt, 0)

		for stmtIdx, fStmt := range fDecl.Body.List {
			ifdefGOOS, found := extractIfdefGOOS(fStmt.Decorations().Start.All())
			if !found {
				continue
			}

			ifdefStmts = append(ifdefStmts, ifdefStmt{goos: ifdefGOOS, stmtIdx: stmtIdx})
		}

		idx := 0
		for stmtIdx, fStmt := range fDecl.Body.List {
			if idx >= len(ifdefStmts) {
				continue
			}

			ifdef := ifdefStmts[idx]
			found := false
			if hasElse(fStmt.Decorations().Start.All()) {
				elseStmts = append(elseStmts, stmt{ifdefIdx: ifdef.stmtIdx, stmtIdx: stmtIdx})
				found = true
			}

			if hasEndif(fStmt.Decorations().End.All()) {
				endifStmts = append(endifStmts, stmt{ifdefIdx: ifdef.stmtIdx, stmtIdx: stmtIdx})
				found = true
			}

			if found {
				idx++
			}
		}

		for idx := len(ifdefStmts) - 1; idx >= 0; idx-- {
			ifdef := ifdefStmts[idx]
			elseStmt, elseFound := correspondingStmt(elseStmts, ifdef.stmtIdx)
			endifStmt, endifFound := correspondingStmt(endifStmts, ifdef.stmtIdx)
			if !endifFound {
				panic(fmt.Sprintf("#endif not found for %+v", ifdef))
			}

			fDecl.Body.List[ifdef.stmtIdx].Decorations().Start.Clear()
			fDecl.Body.List[ifdef.stmtIdx].Decorations().End.Clear()
			fDecl.Body.List[endifStmt.stmtIdx].Decorations().Start.Clear()
			fDecl.Body.List[endifStmt.stmtIdx].Decorations().End.Clear()

			if ifdef.goos == goos && !elseFound {
				continue
			}

			var modifiedBodyList []dst.Stmt

			startIdx := ifdef.stmtIdx
			// endIdx := endifStmts[idx] + 1

			if elseFound {
				if ifdef.goos == goos {
					modifiedBodyList = append(fDecl.Body.List[:elseStmt.stmtIdx], fDecl.Body.List[endifStmt.stmtIdx+1:]...)
				} else {
					modifiedBodyList = append(fDecl.Body.List[:startIdx], fDecl.Body.List[elseStmt.stmtIdx:]...)
					// modifiedBodyList = append(modifiedBodyList, fDecl.Body.List[endifStmt.stmtIdx:]...)
				}
			} else {
				modifiedBodyList = append(fDecl.Body.List[:startIdx], fDecl.Body.List[endifStmt.stmtIdx+1:]...)
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
		ifdefGOOS, found = extractDirectiveByPattern(commentLine, IFDEF_GOOS)
	}

	return ifdefGOOS, found
}

func hasEndif(comments []string) bool {
	var found bool

	for _, commentLine := range comments {
		_, ok := extractDirectiveByPattern(commentLine, ENDIF)
		found = ok
	}

	return found
}

func hasElse(comments []string) bool {
	var found bool

	for _, commentLine := range comments {
		_, ok := extractDirectiveByPattern(commentLine, ELSE)
		found = ok
	}

	return found
}

func processCustomDirectives(f *dst.File) *dst.File {
	allComments := make([]string, 0)

	for _, decl := range f.Decls {
		fDecl, isFunc := decl.(*dst.FuncDecl)
		if !isFunc {
			continue
		}

		allComments = append(allComments, fDecl.Decorations().Start.All()...)
	}

	customDirectives := extractCustomDirectives(allComments)

	for _, decl := range f.Decls {
		fDecl, isFunc := decl.(*dst.FuncDecl)
		if !isFunc {
			continue
		}

		for name, directive := range customDirectives {
			ifdefStmts := make([]ifdefStmt, 0)
			elseStmts := make([]stmt, 0)
			endifStmts := make([]stmt, 0)

			for stmtIdx, fStmt := range fDecl.Body.List {
				found := hasCustomDirective(fStmt.Decorations().Start.All(), name)
				if !found {
					continue
				}

				ifdefStmts = append(ifdefStmts, ifdefStmt{stmtIdx: stmtIdx})
			}

			idx := 0
			for stmtIdx, fStmt := range fDecl.Body.List {
				if idx >= len(ifdefStmts) {
					continue
				}

				ifdef := ifdefStmts[idx]
				found := false
				if hasElse(fStmt.Decorations().Start.All()) {
					elseStmts = append(elseStmts, stmt{ifdefIdx: ifdef.stmtIdx, stmtIdx: stmtIdx})
					found = true
				}

				if hasEndif(fStmt.Decorations().End.All()) {
					endifStmts = append(endifStmts, stmt{ifdefIdx: ifdef.stmtIdx, stmtIdx: stmtIdx})
					found = true
				}

				if found {
					idx++
				}
			}

			for idx := len(ifdefStmts) - 1; idx >= 0; idx-- {
				ifdef := ifdefStmts[idx]
				elseStmt, elseFound := correspondingStmt(elseStmts, ifdef.stmtIdx)
				endifStmt, endifFound := correspondingStmt(endifStmts, ifdef.stmtIdx)
				if !endifFound {
					panic(fmt.Sprintf("#endif not found for %+v", ifdef))
				}

				fDecl.Body.List[ifdef.stmtIdx].Decorations().Start.Clear()
				fDecl.Body.List[ifdef.stmtIdx].Decorations().End.Clear()
				fDecl.Body.List[endifStmt.stmtIdx].Decorations().Start.Clear()
				fDecl.Body.List[endifStmt.stmtIdx].Decorations().End.Clear()

				if directive.evaluated && !elseFound {
					continue
				}

				var modifiedBodyList []dst.Stmt

				startIdx := ifdef.stmtIdx

				if elseFound {
					if directive.evaluated {
						modifiedBodyList = append(fDecl.Body.List[:elseStmt.stmtIdx], fDecl.Body.List[endifStmt.stmtIdx+1:]...)
					} else {
						modifiedBodyList = append(fDecl.Body.List[:startIdx], fDecl.Body.List[elseStmt.stmtIdx:]...)
						// modifiedBodyList = append(modifiedBodyList, fDecl.Body.List[endifStmt.stmtIdx:]...)
					}
				} else {
					modifiedBodyList = append(fDecl.Body.List[:startIdx], fDecl.Body.List[endifStmt.stmtIdx+1:]...)
				}

				fDecl.Body.List = modifiedBodyList
			}

		}

	}

	return f
}

func hasCustomDirective(comments []string, dirName string) bool {
	for _, commentLine := range comments {
		_, found := extractDirectiveByPattern(commentLine, fmt.Sprintf("#ifdef %s", dirName))
		if found {
			return true
		}
	}

	return false
}

func extractCustomDirectives(comments []string) map[string]directive {
	directives := make(map[string]directive)
	pattern := `^(\w+)\s+(true|false)$`
	re := regexp.MustCompile(pattern)

	for _, commentLine := range comments {
		customDirective, ok := extractDirectiveByPattern(commentLine, CUSTOM_DIRECTIVE_PREFIX)
		if !ok {
			continue
		}

		if !re.MatchString(customDirective) {
			continue
		}

		splittedDir := re.FindStringSubmatch(customDirective)
		name, val := splittedDir[1], splittedDir[2]
		dir := directive{name: name}

		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			panic(fmt.Sprintf("invalid bool value '%s': %s", val, err))
		}

		dir.evaluated = boolVal

		directives[name] = dir
	}

	return directives
}

func extractDirectiveByPattern(commentLine string, pattern string) (string, bool) {
	_, comment, ok := strings.Cut(commentLine, "// ")
	if !ok {
		return "", false
	}

	_, rawDirective, ok := strings.Cut(comment, pattern)
	if !ok {
		return "", false
	}

	return strings.TrimSpace(rawDirective), true
}

func correspondingStmt(stmts []stmt, ifdefStmtIdx int) (stmt, bool) {
	stmtIdx := slices.IndexFunc(stmts, func(e stmt) bool { return e.ifdefIdx == ifdefStmtIdx })
	if stmtIdx == -1 {
		return stmt{}, false
	}

	return stmts[stmtIdx], true
}
