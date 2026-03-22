package discovery

import (
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type LanguagePlugin interface {
	Language() *tree_sitter.Language
	Queries() LanguageQueries
	IsHTTPMethod(method string) bool
	CleanRoutePath(raw string) string
	PostProcessRoute(route Route) Route
}

type LanguageQueries struct {
	Route              string
	FuncDecl           string
	ArrowFunc          string
	MountPoint         string
	RouterCreation     string
	ObjectMethod       string
	ObjectPropFunc     string
	CJSExports         string
	CJSDirectExport    string
	ESMExport          string
	RequireImport      string
	ESMImport          string
	ReqBody            string
	ReqBodyDestructure string
	ReqParams          string
	ReqQuery           string
	ZodObject          string
	ZodField           string
}

func GetPlugin(lang string) LanguagePlugin {
	switch lang {
	case "javascript", "js":
		return &JavaScriptPlugin{}
	case "python", "py":
		return &PythonPlugin{}
	case "go", "golang":
		return &GolangPlugin{}
	default:
		return nil
	}
}
func DetectLanguage(filePath string) string {
	ext := strings.TrimPrefix(filepath.Ext(filePath), ".")
	switch ext {
	case "js":
		return "javascript"
	case "ts":
		return "typescript"
	case "py":
		return "python"
	case "go":
		return "go"
	default:
		return ""
	}
}
