package parser

import (
	"go/ast"
	"strings"
)

// Helper function to convert AST expressions to string
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	default:
		return ""
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
