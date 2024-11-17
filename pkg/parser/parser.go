package parser

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"go.uber.org/zap"
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

//go:embed default_templates/handlerTemplate.txt
var handlerTemplate string

type TemplateReplace struct {
	PackageName        string
	InterfaceName      string
	Methods            []Method
	ImportName         string
	DirPackageName     string
	Imports            map[string]bool
	NeedsContextImport bool
	NeedsStrconvImport bool
	NeedsJSONImport    bool
	NeedsNetHTTPImport bool
	NeedsFmtImport     bool
}

func GenerateHTTPHandlers(ctx context.Context, i FileInterface, packageName, outputDir, newTemplate string) error {
	if newTemplate == "" {
		newTemplate = handlerTemplate
	}

	// Create a new FileSet
	fset := token.NewFileSet()

	// Determine the module root directory
	moduleRootDir, err := getModuleRootDir()
	if err != nil {
		return fmt.Errorf("failed to get module root directory: %w", err)
	}

	ctxLogger.Info(ctx, "Import path", zap.String("importPath", i.ImportName))
	// Load the package using go/packages
	cfg := &packages.Config{
		Context: ctx,
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps,
		Fset:    fset,
		Dir:     moduleRootDir,
		Env:     os.Environ(),
	}

	pkgs, err := packages.Load(cfg, i.ImportName)
	if err != nil {
		ctxLogger.Error(ctx, "Failed to load packages", zap.Error(err))
		return fmt.Errorf("failed to load packages: %w", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		ctxLogger.Error(ctx, "Errors occurred while loading packages")
		return fmt.Errorf("errors occurred while loading packages")
	}
	// Find the package that contains your interface
	var pkg *packages.Package
	for _, p := range pkgs {
		for _, file := range p.Syntax {
			ctxLogger.Info(ctx, "File", zap.String("name", p.Fset.Position(file.Package).Filename), zap.String("path", i.FilePath))
			if strings.HasSuffix(p.Fset.Position(file.Package).Filename, i.FilePath) {
				pkg = p
				break
			}
		}
		if pkg != nil {
			break
		}
	}

	if pkg == nil {
		ctxLogger.Info(ctx, "pkgs", zap.Any("pkgs", pkgs))
		return fmt.Errorf("could not find package containing %s", i.FilePath)
	}

	for _, name := range i.Interfaces {
		interfaceSource := getInterfaceSourceFromPackage(pkg, name)
		if interfaceSource == nil {
			ctxLogger.Error(ctx, "Failed to parse interface source", zap.String("interface", name))
			continue
		}

		methods := getMethods(ctx, "/"+name, interfaceSource, pkg.TypesInfo)
		ctxLogger.Info(ctx, "Found methods", zap.Int("count", len(methods)))
		ctxLogger.Info(ctx, "Methods", zap.Any("methods", methods))
		// Proceed to generate handlers using the methods
		// ...
	}
	// Proceed as before...
	return nil
}

func getModuleRootDir() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getImportPath(filePath string) (string, error) {
	modulePath, moduleDir, err := getModuleInfo()
	if err != nil {
		return "", err
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	relPath, err := filepath.Rel(moduleDir, absFilePath)
	if err != nil {
		return "", err
	}

	importPath := path.Join(modulePath, filepath.ToSlash(relPath))
	importPath = strings.TrimSuffix(importPath, ".go") // Remove .go if necessary

	return importPath, nil
}

func getModuleInfo() (modulePath string, moduleDir string, err error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Path}}|{{.Dir}}")
	output, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(strings.TrimSpace(string(output)), "|")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected output from go list")
	}
	return parts[0], parts[1], nil
}

func getInterfaceSourceFromPackage(pkg *packages.Package, interfaceName string) *ast.InterfaceType {
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != interfaceName {
					continue
				}
				if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
					return interfaceType
				}
			}
		}
	}
	return nil
}

func getMethods(ctx context.Context, name string, interfaceSource *ast.InterfaceType, info *types.Info) []*Method {
	var methods []*Method

	for _, m := range interfaceSource.Methods.List {
		if m.Names == nil || len(m.Names) == 0 {
			continue
		}
		methodName := m.Names[0].Name

		// Type assertion to get the function type
		funcType, ok := m.Type.(*ast.FuncType)
		if !ok {
			// Handle embedded interfaces or non-function types if necessary
			continue
		}

		// Initialize the Method struct
		method := Method{
			Name:         methodName,
			HTTPMethod:   "",
			HandlerName:  "",
			URLPath:      "",
			Params:       nil,
			Returns:      nil,
			QueryParams:  nil,
			ResponseType: "",
			RequestType:  "",
			HasContext:   false,
		}

		// Extract parameters
		for _, field := range funcType.Params.List {
			paramType := exprToString(field.Type, info)
			for _, name := range field.Names {
				param := Param{
					Name: name.Name,
					Type: paramType,
				}

				if paramType == "context.Context" {
					method.HasContext = true
				} else {
					method.Params = append(method.Params, param)
				}
			}

			// Handle unnamed parameters (e.g., context.Context without a variable name)
			if len(field.Names) == 0 && paramType == "context.Context" {
				method.HasContext = true
			}
		}

		// Extract return types
		if funcType.Results != nil {
			for _, field := range funcType.Results.List {
				returnType := exprToString(field.Type, info)
				ret := Return{
					Type: returnType,
				}
				method.Returns = append(method.Returns, ret)
			}
		}

		// Infer ResponseType and RequestType from parameters and returns
		if len(method.Params) > 0 {
			method.RequestType = method.Params[len(method.Params)-1].Type
		}
		if len(method.Returns) > 0 {
			method.ResponseType = method.Returns[0].Type
		}

		// Infer HTTPMethod and URLPath based on method name or custom tags
		method.HTTPMethod = inferHTTPMethod(methodName)
		method.URLPath = inferURLPath(name, methodName)

		methods = append(methods, &method)
	}
	return methods
}

// Infer HTTP method from the method name (simple heuristic)
func inferHTTPMethod(methodName string) string {
	if strings.HasPrefix(methodName, "Get") {
		return "GET"
	} else if strings.HasPrefix(methodName, "Post") {
		return "POST"
	} else if strings.HasPrefix(methodName, "Put") {
		return "PUT"
	} else if strings.HasPrefix(methodName, "Delete") {
		return "DELETE"
	}
	return "GET" // Default to GET
}

// Function to split CamelCase words, keeping abbreviations and mixed-case words together
func splitCamelCase(s string) []string {
	var words []string
	var lastPos int
	runes := []rune(s)

	for i := 1; i < len(runes); i++ {
		prev := runes[i-1]
		curr := runes[i]

		// Word boundary rules
		if unicode.IsLower(prev) && unicode.IsUpper(curr) { // Lower -> Upper (e.g., "userId")
			words = append(words, string(runes[lastPos:i]))
			lastPos = i
		} else if unicode.IsUpper(prev) && unicode.IsUpper(curr) { // Upper -> Upper
			if i+1 < len(runes) && unicode.IsLower(runes[i+1]) { // "IDentifier"
				words = append(words, string(runes[lastPos:i]))
				lastPos = i
			}
		}
	}

	// Add the final word
	words = append(words, string(runes[lastPos:]))
	return words
}

// Infer URL path from the method name using an enhanced heuristic
func inferURLPath(prefix string, methodName string, params ...Param) string {
	// Split the method name into words
	words := splitCamelCase(methodName)

	// Initialize variables
	action := ""
	resources := []string{}
	pathParams := []string{}

	// Define a set of common actions
	actionsSet := map[string]bool{
		"Get":    true,
		"Create": true,
		"Update": true,
		"Delete": true,
		"List":   true,
		"Add":    true,
		"Remove": true,
	}

	// Process words to extract action and resources
	i := 0
	if i < len(words) && actionsSet[words[i]] {
		action = words[i]
		i++
	}

	// Collect resources until 'By' keyword
	for i < len(words) {
		if words[i] == "By" {
			i++
			break
		}
		resources = append(resources, words[i])
		i++
	}

	// Map function parameter names to a lookup map
	paramNames := make(map[string]bool)
	for _, param := range params {
		paramNames[strings.ToLower(param.Name)] = true
	}

	// Collect and combine path parameters from the method name after 'By'
	var tempParam []string
	for i < len(words) {
		tempParam = append(tempParam, strings.ToLower(words[i]))
		for i := 0; i < len(tempParam); i++ {
			combinedParam := strings.Join(tempParam[i:], "")
			if paramNames[combinedParam] {
				pathParams = append(pathParams, combinedParam)
				tempParam = nil // Reset tempParam for the next parameter

				break
			}
		}
		i++
	}

	// If no path parameters from method name, infer from parameters for certain actions
	if len(pathParams) == 0 && (action == "Get" || action == "Update" || action == "Delete") {
		for _, param := range params {
			paramName := strings.ToLower(param.Name)
			// Common identifier names
			if paramName == "id" || strings.HasSuffix(paramName, "id") {
				pathParams = append(pathParams, paramName)

			}
		}
	}

	// Build the path
	var pathBuilder strings.Builder
	for _, res := range resources {
		// Pluralize resource names for collection endpoints
		resourceName := strings.ToLower(res)
		if (action == "List" || action == "Create") && !strings.HasSuffix(resourceName, "s") {
			resourceName = resourceName + "s"
		}
		pathBuilder.WriteString("/")
		pathBuilder.WriteString(resourceName)
	}

	for _, param := range pathParams {
		pathBuilder.WriteString("/{")
		pathBuilder.WriteString(param)
		pathBuilder.WriteString("}")
	}

	return path.Join(strings.ToLower(prefix), pathBuilder.String())
}
