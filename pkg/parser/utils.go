package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"reflect"
	"strings"
)

// Updated helper function to convert AST expressions to string with package info
func exprToString(expr ast.Expr, info *types.Info) string {
	qf := func(pkg *types.Package) string {
		// Customize how package names are printed.
		// You can return pkg.Path() or pkg.Name(), depending on your preference.
		return pkg.Path()
	}

	switch t := expr.(type) {
	case *ast.Ident:
		if obj, ok := info.Uses[t]; ok {
			if typ := obj.Type(); typ != nil && typ != types.Typ[types.Invalid] {
				return types.TypeString(typ, qf)
			}
		}
		return t.Name
	case *ast.SelectorExpr:
		if typ, ok := info.Types[t]; ok && typ.Type != nil && typ.Type != types.Typ[types.Invalid] {
			return types.TypeString(typ.Type, qf)
		}
		// Fallback to reconstructing the selector expression
		xStr := exprToString(t.X, info)
		return xStr + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X, info)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt, info)
	case *ast.MapType:
		return "map[" + exprToString(t.Key, info) + "]" + exprToString(t.Value, info)
	case *ast.Ellipsis:
		return "..." + exprToString(t.Elt, info)
	default:
		// For other types, use the printer as a last resort
		var buf bytes.Buffer
		err := printer.Fprint(&buf, token.NewFileSet(), expr)
		if err != nil {
			return ""
		}
		return buf.String()
	}
}

// determineHTTPMethod infers the HTTP method based on the function name.
func determineHTTPMethod(funcName string) string {
	// Simple heuristic: CRUD operations
	switch {
	case strings.HasPrefix(funcName, "Get") || strings.HasPrefix(funcName, "List"):
		return "GET"
	case strings.HasPrefix(funcName, "Create") || strings.HasPrefix(funcName, "New"):
		return "POST"
	case strings.HasPrefix(funcName, "Update"):
		return "PUT"
	case strings.HasPrefix(funcName, "Delete") || strings.HasPrefix(funcName, "Remove"):
		return "DELETE"
	default:
		return "POST" // default to POST
	}
}

// toSnakeCase converts a CamelCase string to snake_case.
func toSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && isUpper(r) && (i+1 < len(str) && isLower(rune(str[i+1])) || isLower(rune(str[i-1]))) {
			result = append(result, '_')
		}
		result = append(result, toLowerRune(r))
	}
	return string(result)
}

func isUpper(r rune) bool {
	return 'A' <= r && r <= 'Z'
}

func isLower(r rune) bool {
	return 'a' <= r && r <= 'z'
}

func toLowerRune(r rune) rune {
	if isUpper(r) {
		return r + ('a' - 'A')
	}
	return r
}

// hasError checks if any of the return types is an error
func hasError(returns []Return) bool {
	for _, r := range returns {
		if r.Type == "error" {
			return true
		}
	}
	return false
}

// hasOnlyError checks if the only return type is error
func hasOnlyError(returns []Return) bool {
	if len(returns) != 1 {
		return false
	}
	return returns[0].Type == "error"
}

// hasMultiple checks if there are multiple non-error return types
func hasMultiple(returns []Return) bool {
	count := 0
	for _, r := range returns {
		if r.Type != "error" {
			count++
		}
	}
	return count > 1
}

// Template function to handle optional fields
func orFunc(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// toPascalCase converts a string to PascalCase.
func toPascalCase(input string) string {
	// Split the input string by common delimiters
	words := strings.FieldsFunc(input, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})

	// Capitalize the first letter of each word
	var pascalCase string
	for _, word := range words {
		if len(word) > 0 {
			pascalCase += strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return pascalCase
}

type DataType struct {
	Name       string
	Type       string
	PkgName    string
	IsPointer  bool
	IsEllipsis bool
	IsArray    bool
	IsSlice    bool
}

// parseType extracts the type information from an AST expression
func parseType(expr ast.Expr) DataType {
	switch t := expr.(type) {
	case *ast.StarExpr:
		// Pointer Type
		innerDataType := parseType(t.X)
		innerDataType.IsPointer = true
		innerDataType.Type = "*" + innerDataType.Type
		return innerDataType

	case *ast.SelectorExpr:
		// Qualified identifier, e.g., pkg.Type
		pkgIdent, ok := t.X.(*ast.Ident)
		if ok {
			return DataType{
				Type:    pkgIdent.Name + "." + t.Sel.Name,
				PkgName: pkgIdent.Name,
			}
		}

	case *ast.Ident:
		// Simple identifier
		return DataType{
			Type: t.Name,
		}

	case *ast.ArrayType:
		// Slice or Array Type
		elemDataType := parseType(t.Elt)
		if t.Len == nil {
			// It's a slice
			elemDataType.IsSlice = true
			elemDataType.Type = "[]" + elemDataType.Type
		} else {
			// It's an array
			elemDataType.IsArray = true
			lengthExpr := t.Len
			var lengthStr string
			if basicLit, ok := lengthExpr.(*ast.BasicLit); ok && basicLit.Kind == token.INT {
				lengthStr = basicLit.Value // Array length as string
			} else {
				lengthStr = "N" // Placeholder if length is not a basic literal
			}
			elemDataType.Type = "[" + lengthStr + "]" + elemDataType.Type
		}
		return elemDataType

	case *ast.Ellipsis:
		// Variadic parameter
		elemDataType := parseType(t.Elt)
		elemDataType.IsEllipsis = true
		elemDataType.Type = "..." + elemDataType.Type
		return elemDataType

	case *ast.MapType:
		// Map Type
		keyDataType := parseType(t.Key)
		valueDataType := parseType(t.Value)
		return DataType{
			Type: "map[" + keyDataType.Type + "]" + valueDataType.Type,
			// You might need to handle PkgName if key or value types are from different packages
		}

	case *ast.InterfaceType:
		// Interface Type
		if len(t.Methods.List) == 0 {
			// Empty interface
			return DataType{
				Type: "interface{}",
			}
		} else {
			// Interface with methods
			return DataType{
				Type: "interface{ /* methods */ }",
			}
		}

	default:
		// Handle other types or unknown types
		b, _ := json.Marshal(expr)
		fmt.Println(string(b))
		fmt.Println(reflect.TypeOf(expr))
		panic("unknown type")
	}
	return DataType{}
}

func getUniqueVarName(baseName string) string {
	existingNames := map[string]bool{
		"r":   true,
		"w":   true,
		"ctx": true,
	}
	name := baseName
	i := 1
	for existingNames[name] {
		name = fmt.Sprintf("%s%d", baseName, i)
		i++
	}
	existingNames[name] = true
	return name
}
