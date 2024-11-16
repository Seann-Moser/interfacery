package parser

import (
	"go/ast"
	"testing"
)

// Mock data for testing
var (
	mockReturns1 = []Return{
		{Type: "int"},
		{Type: "error"},
	}
	mockReturns2 = []Return{
		{Type: "error"},
	}
	mockReturns3 = []Return{
		{Type: "string"},
		{Type: "int"},
	}
)

func TestExprToString(t *testing.T) {
	tests := []struct {
		name     string
		input    ast.Expr
		expected string
	}{
		{"Ident", &ast.Ident{Name: "foo"}, "foo"},
		{"SelectorExpr", &ast.SelectorExpr{
			X:   &ast.Ident{Name: "pkg"},
			Sel: &ast.Ident{Name: "Bar"},
		}, "pkg.Bar"},
		{"StarExpr", &ast.StarExpr{X: &ast.Ident{Name: "ptr"}}, "*ptr"},
		{"ArrayType", &ast.ArrayType{Elt: &ast.Ident{Name: "int"}}, "[]int"},
		{"MapType", &ast.MapType{
			Key:   &ast.Ident{Name: "string"},
			Value: &ast.Ident{Name: "int"},
		}, "map[string]int"},
		{"UnknownExpr", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exprToString(tt.input)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetermineHTTPMethod(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		expected string
	}{
		{"GetMethod", "GetUser", "GET"},
		{"ListMethod", "ListUsers", "GET"},
		{"CreateMethod", "CreateUser", "POST"},
		{"NewMethod", "NewSession", "POST"},
		{"UpdateMethod", "UpdateUser", "PUT"},
		{"DeleteMethod", "DeleteUser", "DELETE"},
		{"RemoveMethod", "RemoveItem", "DELETE"},
		{"DefaultMethod", "FetchData", "POST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineHTTPMethod(tt.funcName)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"CamelCase", "CamelCaseString", "camel_case_string"},
		{"SingleWord", "Word", "word"},
		{"LowerCase", "lowercase", "lowercase"},
		{"EmptyString", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasError(t *testing.T) {
	tests := []struct {
		name     string
		returns  []Return
		expected bool
	}{
		{"HasError", mockReturns1, true},
		{"OnlyError", mockReturns2, true},
		{"NoError", mockReturns3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasError(tt.returns)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasOnlyError(t *testing.T) {
	tests := []struct {
		name     string
		returns  []Return
		expected bool
	}{
		{"OnlyError", mockReturns2, true},
		{"HasMultiple", mockReturns1, false},
		{"NoError", mockReturns3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasOnlyError(tt.returns)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasMultiple(t *testing.T) {
	tests := []struct {
		name     string
		returns  []Return
		expected bool
	}{
		{"HasMultipleNonErrors", mockReturns3, true},
		{"SingleNonError", []Return{{Type: "string"}}, false},
		{"OnlyError", mockReturns2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasMultiple(tt.returns)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestOrFunc(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		defaultValue  string
		expectedValue string
	}{
		{"ValueProvided", "hello", "world", "hello"},
		{"DefaultUsed", "", "world", "world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := orFunc(tt.value, tt.defaultValue)
			if result != tt.expectedValue {
				t.Errorf("got %v, want %v", result, tt.expectedValue)
			}
		})
	}
}
