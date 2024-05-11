package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/pijng/moonject"
)

const GOOS = "GOOS:"
const IFDEF = "#ifdef"
const ENDIF = "#endif"
const ELSE = "#else"
const DEFINE = "#define "

func main() {
	goos := os.Getenv("GOOS")
	ifdefModifier := IfdefModifier{GOOS: goos}

	moonject.Process(ifdefModifier)
}

type IfdefModifier struct {
	GOOS string
}

type ifdefStmt struct {
	evaluated bool
	line      int
}

type stmt struct {
	ifdefIdx int
	line     int
}

func (ifm IfdefModifier) Modify(f *dst.File, dec *decorator.Decorator, res *decorator.Restorer) *dst.File {
	var buf bytes.Buffer
	err := res.Fprint(&buf, f)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(buf.Bytes()))
	scanner.Split(bufio.ScanLines)

	lineIdx := 0
	customDirectives := make(map[string]bool)
	ifdefStmts := make([]ifdefStmt, 0)
	elseStmts := make([]stmt, 0)
	endifStmts := make([]stmt, 0)

	for scanner.Scan() {
		line := scanner.Text()

		name, evaluated, directiveFound := customDirective(line)
		if directiveFound {
			customDirectives[name] = evaluated
		}

		ifdef, ifdefFound := extractDirectiveByPattern(line, IFDEF)
		if ifdefFound {
			var evaluated bool

			_, ifdefGOOS, isGOOS := strings.Cut(ifdef, GOOS)
			if isGOOS {
				evaluated = ifm.GOOS == ifdefGOOS
			} else {
				evaluated = customDirectives[ifdef]
			}

			ifdefStmts = append(ifdefStmts, ifdefStmt{evaluated: evaluated, line: lineIdx})
		}

		els, elseFound := extractDirectiveByPattern(line, ELSE)
		if elseFound {
			ifdefIdx, ok := latestNonClosedIfdef(ifdefStmts, endifStmts)
			if !ok {
				panic(fmt.Sprintf("#ifdef not found for %+v", els))
			}
			elseStmts = append(elseStmts, stmt{ifdefIdx: ifdefIdx, line: lineIdx})
		}

		endif, endifFound := extractDirectiveByPattern(line, ENDIF)
		if endifFound {
			ifdefIdx, ok := latestNonClosedIfdef(ifdefStmts, endifStmts)
			if !ok {
				panic(fmt.Sprintf("#ifdef not found for %+v", endif))
			}
			endifStmts = append(endifStmts, stmt{ifdefIdx: ifdefIdx, line: lineIdx})
		}

		lineIdx++
	}

	contentLines := strings.Split(buf.String(), "\n")

	linesWere := len(contentLines)
	linesRemoved := 0
	latestIfdefLine := 0
	for ifdefIdx, ifdef := range ifdefStmts {
		if ifdef.line < latestIfdefLine {
			continue
		}

		elseStmt, elseFound := correspondingStmt(elseStmts, ifdefIdx)
		endifStmt, endifFound := correspondingStmt(endifStmts, ifdefIdx)
		if !endifFound {
			panic(fmt.Sprintf("#endif not found for %+v", ifdef))
		}

		start := ifdef.line - linesRemoved
		end := endifStmt.line - linesRemoved + 1

		if ifdef.evaluated {
			if elseFound {
				contentLines = slices.Concat(contentLines[:start], contentLines[start+1:elseStmt.line-linesRemoved], contentLines[end:])
			} else {
				contentLines = slices.Concat(contentLines[:start], contentLines[start+1:endifStmt.line-linesRemoved], contentLines[end:])
			}
		}

		if !ifdef.evaluated {
			if elseFound {
				contentLines = slices.Concat(contentLines[:start], contentLines[elseStmt.line+1-linesRemoved:endifStmt.line-linesRemoved], contentLines[end:])
			} else {
				contentLines = slices.Concat(contentLines[:start], contentLines[end:])
			}
		}

		latestIfdefLine = end
		linesRemoved = linesWere - len(contentLines)
	}

	code := strings.Join(contentLines, "\n")
	f, err = dec.Parse(code)
	if err != nil {
		panic(err)
	}

	return f
}

func latestNonClosedIfdef(ifdefStmts []ifdefStmt, endifStmts []stmt) (int, bool) {
	for n := len(ifdefStmts) - 1; n >= 0; n-- {

		_, hasEndif := correspondingStmt(endifStmts, n)
		if hasEndif {
			continue
		}

		return n, true
	}

	return 0, false
}

func customDirective(line string) (string, bool, bool) {
	customDirective, ok := extractDirectiveByPattern(line, DEFINE)
	if !ok {
		return "", false, false
	}

	parts := strings.Split(customDirective, " ")
	name := parts[0]

	var strVal string
	if len(parts) == 1 {
		strVal = os.Getenv(name)
	} else {
		strVal = parts[1]
	}

	boolVal, err := strconv.ParseBool(strVal)
	if err != nil {
		boolVal = false
	}

	return name, boolVal, true
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
	rawDirective = strings.TrimSpace(rawDirective)

	return rawDirective, true
}

func correspondingStmt(stmts []stmt, ifdefStmtIdx int) (stmt, bool) {
	stmtIdx := slices.IndexFunc(stmts, func(e stmt) bool { return e.ifdefIdx == ifdefStmtIdx })
	if stmtIdx == -1 {
		return stmt{}, false
	}

	return stmts[stmtIdx], true
}
