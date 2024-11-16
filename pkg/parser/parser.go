package parser

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed default_templates/handlerTemplate.txt
var handlerTemplate string

type TemplateReplace struct {
	PackageName    string
	InterfaceName  string
	Methods        []Method
	ImportName     string
	DirPackageName string
}

func GenerateHTTPHandlers(i FileInterface, interfaceSrc, packageName, outputDir, newTemplate string) error {
	if newTemplate == "" {
		newTemplate = handlerTemplate
	}
	// Parse the interface source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "interface.go", interfaceSrc, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse interface source: %v", err)
	}

	var iface *ast.InterfaceType
	var ifaceName string

	// Find the interface in the AST
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Type == nil {
				continue
			}
			if astType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				iface = astType
				ifaceName = typeSpec.Name.Name
				break
			}
		}
		if iface != nil {
			break
		}
	}

	if iface == nil {
		return fmt.Errorf("no interface found in the provided source")
	}

	var methods []Method

	for _, method := range iface.Methods.List {
		// Each method can have multiple names (unlikely in interfaces, but possible)
		for _, methodName := range method.Names {
			funcType, ok := method.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			// Extract parameters
			var params []Param
			var queryParams []string
			var muxVar []string
			if funcType.Params != nil {
				for i, param := range funcType.Params.List {
					var paramName string
					if len(param.Names) > 0 {
						paramName = param.Names[0].Name
					} else {
						// Generate a parameter name if not provided
						paramName = fmt.Sprintf("param%d", i)
					}
					if strings.HasSuffix(strings.ToLower(paramName), "id") {
						muxVar = append(muxVar, paramName)
					}
					paramType := exprToString(param.Type)
					// Skip context.Context parameter
					if paramType == "context.Context" && paramName == "ctx" {
						continue
					}
					params = append(params, Param{
						Name: paramName,
						Type: paramType,
					})
					queryParams = append(queryParams, paramName)
				}
			}

			// Extract return types
			var returns []Return
			if funcType.Results != nil {
				for _, result := range funcType.Results.List {
					resultType := exprToString(result.Type)
					returns = append(returns, Return{
						Type: resultType,
					})
				}
			}

			// Determine HTTP method based on the function name
			httpMethod := determineHTTPMethod(methodName.Name)

			// Determine URL path
			urlPath := "/" + strings.ToLower(ifaceName) + "/" + toSnakeCase(methodName.Name)

			// Determine ResponseTypeMap and RequestTypeMap
			var responseType string
			var requestType string

			// Handle ResponseType
			if len(returns) > 1 {
				// Assuming the last return type is error
				responseType = returns[0].Type
			} else if len(returns) == 1 {
				if returns[0].Type != "error" {
					responseType = returns[0].Type
				} else {
					responseType = ""
				}
			} else {
				responseType = ""
			}

			// Handle RequestType
			if len(params) > 0 {
				// If multiple params, define a struct type
				if len(params) == 1 {
					requestType = params[0].Type
				} else {
					requestType = fmt.Sprintf("%sRequest", methodName.Name)
				}
			} else {
				requestType = ""
			}

			methods = append(methods, Method{
				Name:         methodName.Name,
				HTTPMethod:   httpMethod,
				HandlerName:  fmt.Sprintf("%sHandler", methodName.Name),
				URLPath:      urlPath,
				Params:       params,
				Returns:      returns,
				QueryParams:  queryParams,
				ResponseType: responseType,
				RequestType:  requestType,
			})
		}
	}

	// Prepare the template for the handler file

	// Create a FuncMap for template functions
	funcMap := template.FuncMap{
		"toSnakeCase":  toSnakeCase,
		"or":           orFunc,
		"title":        strings.Title,
		"toLower":      strings.ToLower,
		"hasError":     hasError,
		"hasOnlyError": hasOnlyError,
		"hasMultiple":  hasMultiple,
		"toPascalCase": toPascalCase,
	}

	// Parse the template
	tmpl, err := template.New("handler").Funcs(funcMap).Parse(newTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Prepare data for the template
	data := TemplateReplace{
		PackageName:    packageName,
		InterfaceName:  ifaceName,
		Methods:        methods,
		ImportName:     i.ImportName,
		DirPackageName: getPathSegment(outputDir, -1),
	}

	// Execute the template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	// Ensure the output directory exists
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Write to a Go file
	outputFile := fmt.Sprintf("%s/%s.go", outputDir, packageName)
	err = os.WriteFile(outputFile, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write handler file: %v", err)
	}

	log.Printf("HTTP handlers generated successfully at %s", outputFile)
	return nil
}

// getPathSegment extracts the segment of the path at the specified position.
func getPathSegment(path string, position int) string {
	// Normalize the path to use forward slashes consistently
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")

	// Validate the position
	if position < 0 || position >= len(parts) {
		position = len(parts) - 1
	}

	return parts[position]
}
