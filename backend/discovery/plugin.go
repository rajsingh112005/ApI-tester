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
    Route              string // finds route registrations
    FuncDecl           string // function declarations
    ArrowFunc          string // arrow functions / function expressions
	MountPoint         string  
	RouterCreation     string 
    ObjectMethod       string // method shorthand in objects
    ObjectPropFunc     string // object property with function value
    CJSExports         string // module.exports = { ... }
    CJSDirectExport    string // module.exports.fn = ...
    ESMExport          string // export function / export const
    RequireImport      string // const x = require('./path')
    ESMImport          string // import { x } from './path'
    ReqBody            string // req.body.field
    ReqBodyDestructure string // const { field } = req.body
    ReqParams          string // req.params.field
    ReqQuery           string // req.query.field
    ZodObject          string // z.object({ ... })
    ZodField           string // individual field inside z.object
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