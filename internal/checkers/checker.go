package checkers

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"golang.org/x/tools/go/analysis"
	"bytes"
	"go/printer"
)

type Config struct {
	RequireStringKey bool
	NoPrintfLike     bool
}

type CallContext struct {
	Expr      *ast.CallExpr
	Func      *types.Func
	Signature *types.Signature
	File      *ast.File  
}

type Checker interface {
	FilterKeyAndValues(pass *analysis.Pass, keyAndValues []ast.Expr) []ast.Expr
	CheckLoggingKey(pass *analysis.Pass, keyAndValues []ast.Expr)
	CheckPrintfLikeSpecifier(pass *analysis.Pass, args []ast.Expr)
}

func ExecuteChecker(c Checker, pass *analysis.Pass, call CallContext, cfg Config) {
	params := call.Signature.Params()
	nparams := params.Len() // variadic => nonzero
	startIndex := nparams - 1

	lastArg := params.At(nparams - 1)
	iface, ok := lastArg.Type().(*types.Slice).Elem().(*types.Interface)
	if !ok || !iface.Empty() {
		return // final (args) param is not ...interface{}
	}

	keyValuesArgs := c.FilterKeyAndValues(pass, call.Expr.Args[startIndex:])
	
	if len(keyValuesArgs)%2 != 0 {
		firstArg := keyValuesArgs[0]
		lastArg := keyValuesArgs[len(keyValuesArgs)-1]
		pass.Report(analysis.Diagnostic{
			Pos:      firstArg.Pos(),
			End:      lastArg.End(),
			Category: DiagnosticCategory,
			Message:  "odd number of arguments passed as key-value pairs for logging",
		})
	}

	// // Find the enclosing statement that contains the call
	// enclosingStmt := findEnclosingStmt(call.File, call.Expr)
	// if enclosingStmt == nil {
	// 	fmt.Println("enclosingStmt == nil")
	// 	return // Exit if no enclosing statement is found
	// }

	// // Convert the enclosing statement to string
	// var buf bytes.Buffer
	// fset := token.NewFileSet()
	// if err := printer.Fprint(&buf, fset, enclosingStmt); err != nil {
	// 	fmt.Println("printer.Fprint(&buf, fset, enclosingStmt) == err")
	// 	return // Handle error, if needed
	// }
	// fmt.Printf("HELLLOOOOOOOOOO: %v", enclosingStmt)

	// // Check if the line contains "NewLogger"
	// if !strings.Contains(buf.String(), "NewLogger") {
	// 	return // Exit if "NewLogger" is not found
	// }

	// // Find the line of code containing the call
	// lineNumber := pass.Fset.Position(call.Expr.Pos()).Line

	// // Retrieve the line content using the provided file set and position
	// lineContent := getLineContent(pass.Fset.File(call.Expr.Pos()), lineNumber)
	// if lineContent == "" || !strings.Contains(lineContent, "NewLogger") {
	// 	return // Exit if "NewLogger" is not found
	// }

	result := isWithValuesCallOnNewLogger(call.File, call.Expr.Pos())
	if result == false {
		return
	}


	hasTraceId := false
	for i := 0; i < len(keyValuesArgs); i += 2 {
		arg := keyValuesArgs[i]
		basic, ok := arg.(*ast.BasicLit)
		if !ok {
			// Just ignore it if its not of type BasicLiteral
			continue
		}

		// We use traceId not traceID based on spanId in Google stackdriver stuctured logging
		// https://cloud.google.com/logging/docs/structured-logging
		// In the opentelemetry docs it is "TraceId"
		// https://opentelemetry.io/docs/specs/otel/trace/api/#retrieving-the-traceid-and-spanid
		// It looks like in the wire format of the w3c spec it might be trace-id
		// https://www.w3.org/TR/trace-context/#trace-id
		// This is also how its defined in the OpenTelemetry spec for jsonLogs
		// https://opentelemetry.io/docs/specs/otel/protocol/file-exporter/#examples
		// https://opentelemetry.io/docs/specs/otel/logs/
		if basic.Value == "\"traceId\"" {
			hasTraceId = true
			break
		}
	}

	if !hasTraceId {
		d := &analysis.Diagnostic{
			Category: DiagnosticCategory,
			Message:  "missing traceId in logging keys",
			// Here's where we set the position at which to report this.
			// We use the position of call argument
			Pos: call.Expr.Pos(),
			// N.B we don't set end because it should just apply to the entire line pointed at by pos.
		}

		// Parse the existing arguments to the log function
		existingArgs, err := getArgs(call.Expr)
		if err != nil {
			// Handle error here. For example:
			pass.Report(analysis.Diagnostic{
					Pos:      call.Expr.Pos(),
					Category: DiagnosticCategory,
					Message:  fmt.Sprintf("Failed to get arguments: %v", err),
			})
   	 return
		}

		// Add span declaration at the start of the function
		spanDeclaration := "span := trace.SpanFromContext(ctx)"
		
		// Add traceId and spanId to the logging call
		traceIdAddition := "\"traceId\", span.SpanContext().TraceID().String()"
		spanIdAddition := "\"spanId\", span.SpanContext().SpanID().String()"

		// Create a new slice to hold the modified arguments
		newArgs := make([]string, len(existingArgs)+2)

		// Copy the existing arguments before startIndex to the new slice
		if len(existingArgs) > 0 {
			copy(newArgs, existingArgs[:startIndex])
		}

		newArgs[startIndex] = traceIdAddition
		newArgs[startIndex+1] = spanIdAddition

		// Copy the remaining existing arguments to the new slice
		if len(existingArgs) > startIndex {
			copy(newArgs[startIndex+2:], existingArgs[startIndex:])
		}

		// Construct the new arguments string
		newArgsStr := strings.Join(newArgs, ", ")

		// Replace the existing logging call with the new one including traceId and spanId
		newLogCall := fmt.Sprintf(newArgsStr)

		spanInsertPos := findPosOfFuncBody(call.File, call.Expr)

		textEdits := []analysis.TextEdit{
			// Add the span declaration
			{
				Pos: spanInsertPos,
				End: spanInsertPos,
				NewText: []byte(spanDeclaration + "\n"),
			},
			// Replace the logging call
			{
				Pos: findPosOfArgs(call.Expr),
				End: findEndPosOfArgs(call.Expr),
				NewText: []byte(newLogCall),
			},
		}

		lib := "go.opentelemetry.io/otel/trace"
		pos, err := getImportPos(call.File, lib)
		if err == nil && pos != token.NoPos {
			// Create an edit map to add the trace lib
			edit := analysis.TextEdit{
				Pos:     pos,
				End:     pos,
				NewText: []byte("\"" + lib + "\"" + "\n"),
			}
			// Append edit to textEdits
			textEdits = append(textEdits, edit)
		}

		d.SuggestedFixes = []analysis.SuggestedFix{
			{
				Message: "Add traceId and spanId to logging keys",
				// Edit the code to make the fix.
				TextEdits: textEdits,
			},
		}
		pass.Report(*d)
	}

	if cfg.RequireStringKey {
		c.CheckLoggingKey(pass, keyValuesArgs)
	}

	if cfg.NoPrintfLike {
		// Check all args
		c.CheckPrintfLikeSpecifier(pass, call.Expr.Args)
	}
}

// EnclosingFunc finds the function that encloses the given position.
// TODO: Refactor the code to avoid revisiting files 
func enclosingFunc(file *ast.File, pos token.Pos) (fun *ast.FuncDecl, funLit *ast.FuncLit) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.FuncDecl:
			if v.Body != nil && pos >= v.Body.Pos() && pos <= v.Body.End() {
				fun = v
			}
		case *ast.FuncLit:
			if pos >= v.Body.Pos() && pos <= v.Body.End() {
				funLit = v
			}
		}
		return true
	})
	return
}

// FindPosOfFuncBody returns the position of the beginning of the function's body
// that contains the given call expression.
func findPosOfFuncBody(file *ast.File, call *ast.CallExpr) token.Pos {
	fun, _ := enclosingFunc(file, call.Pos())

	if fun != nil && fun.Body != nil {
		insertPos := fun.Body.Rbrace
		if len(fun.Body.List) > 0 {
			insertPos = fun.Body.List[0].Pos()
		}
		return insertPos
	}
	// If fun or fun.Body is nil, return a valid position or handle the error appropriately
	// For now, we just return token.NoPos
	return token.NoPos
}


func getArgs(call *ast.CallExpr) ([]string, error) {
	fset := token.NewFileSet()
	args := make([]string, len(call.Args))
	
	for i, arg := range call.Args {
			// Use the Fprint function from go/printer to convert each expression to a string
			var buf bytes.Buffer
			if err := printer.Fprint(&buf, fset, arg); err != nil {
					return nil, fmt.Errorf("error formatting argument: %w", err)
			}
			args[i] = buf.String()
	}

	return args, nil
}


func findPosOfArgs(call *ast.CallExpr) token.Pos {
    if len(call.Args) > 0 {
        return call.Args[0].Pos()
    }
    // If there are no arguments, return the position of the opening parenthesis
    return call.Lparen
}

func findEndPosOfArgs(call *ast.CallExpr) token.Pos {
    if len(call.Args) > 0 {
        return call.Args[len(call.Args)-1].End()
    }
    // If there are no arguments, return the position of the opening parenthesis
    return call.Lparen
}


// findImportStmt finds all import statements in the given AST file.
// Returns an error if the provided file is nil.
func findImportStmt(file *ast.File) (importSpecs []*ast.ImportSpec, err error) {
    if file == nil {
        return nil, fmt.Errorf("provided file is nil")
    }
    
    for _, decl := range file.Decls {
        genDecl, ok := decl.(*ast.GenDecl)
        if !ok {
            continue
        }

        // If the declaration is an import declaration
        if genDecl.Tok == token.IMPORT {
            for _, spec := range genDecl.Specs {
                importSpec, ok := spec.(*ast.ImportSpec)
                if ok {
                    importSpecs = append(importSpecs, importSpec)
                }
            }
        }
    }
    return importSpecs, nil
}



func getImportPos(file *ast.File, lib string) (token.Pos, error) {
	importSpecs, err := findImportStmt(file)
	if err != nil {
			return token.NoPos, err
	}

	for _, importSpec := range importSpecs {
			if strings.Trim(importSpec.Path.Value, `""`) == lib {
					return token.NoPos, nil
			}
	}

	if len(importSpecs) > 0 {
		// Return position after opening bracket of the first import declaration
		return importSpecs[0].Pos(), nil
	}

	return token.NoPos, nil
}

func isWithValuesCallOnNewLogger(file *ast.File, pos token.Pos) bool {
	// Function to find the node at the given position
	var result bool
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		// Check if the node corresponds to the given position
		if n.Pos() <= pos && pos <= n.End() {
			// Check if the node is a call expression
			if call, ok := n.(*ast.CallExpr); ok {
				// Check if the function being called is a selector expression (e.g., obj.Method())
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						// Check if the receiver is a call to zapr.NewLogger
					if call, ok := sel.X.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
							if sel.Sel.Name == "NewLogger" {
								fmt.Println("WORKS")
								result = true
								return true
							}
						}
					}
				}
			}
		}
		return true
	})
	return result
}





