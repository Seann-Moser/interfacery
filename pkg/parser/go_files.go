package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileInterface represents a Go file, its package name, and the interfaces it contains.
type FileInterface struct {
	FilePath    string   // Relative file path
	PackageName string   // Package name
	Interfaces  []string // List of interface names
	ImportName  string
}

func getRelativePath(fullPath string) string {
	const prefix = "go/src/"
	if idx := strings.Index(fullPath, prefix); idx != -1 {
		return fullPath[idx+len(prefix):]
	}
	return fullPath // Return the original path if "go/src/" is not found
}

// FindGoFilesWithInterfaces recursively searches for .go files containing interfaces.
// If interfaceName is non-empty, it filters only files containing the specified interface.
func FindGoFilesWithInterfaces(rootDir, interfaceName string, ignoreDirs ...string) ([]FileInterface, error) {
	var result []FileInterface
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	currentDir = getRelativePath(currentDir)
	err = filepath.WalkDir(rootDir, func(filePath string, d fs.DirEntry, err error) error {
		if strings.Contains(filePath, "vendor") {
			return filepath.SkipDir
		}
		for _, ignoreDir := range ignoreDirs {
			if strings.Contains(filePath, ignoreDir) {
				return filepath.SkipDir
			}
		}
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		// Get interfaces and package name
		interfaces, pkgName, err := getInterfacesAndPackage(filePath, interfaceName)
		if err != nil {
			return err
		}

		// Add to result if interfaces are found
		if len(interfaces) > 0 {
			relativePath, err := filepath.Rel(rootDir, filePath)
			if err != nil {
				return err
			}
			result = append(result, FileInterface{
				FilePath:    relativePath,
				PackageName: pkgName,
				Interfaces:  interfaces,
				ImportName:  path.Join(currentDir, strings.TrimSuffix(strings.TrimSuffix(relativePath, "/"+pkgName+".go"), ".go")),
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// getInterfacesAndPackage retrieves all interfaces and the package name in a Go file.
func getInterfacesAndPackage(filePath, interfaceName string) ([]string, string, error) {
	fset := token.NewFileSet()

	// Parse the file
	node, err := parser.ParseFile(fset, filePath, nil, parser.AllErrors)
	if err != nil {
		return nil, "", err
	}

	var interfaces []string
	packageName := node.Name.Name // Extract package name

	// Traverse the AST to find interfaces
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			// Check if it's an interface type
			if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				if interfaceName == "" || typeSpec.Name.Name == interfaceName {
					interfaces = append(interfaces, typeSpec.Name.Name)
				}
			}
		}
	}

	return interfaces, packageName, nil
}
