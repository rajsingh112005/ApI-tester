package discovery

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

type PythonPlugin struct{}

func (p *PythonPlugin) Language() *tree_sitter.Language {
	return tree_sitter.NewLanguage(tree_sitter_python.Language())
}
func (p *PythonPlugin) IsHTTPMethod(method string) bool {
	methods := map[string]bool{
		"get":    true,
		"post":   true,
		"put":    true,
		"patch":  true,
		"delete": true,
		"route":  true,
	}
	return methods[strings.ToLower(method)]
}

func (p *PythonPlugin) CleanRoutePath(raw string) string {
	return strings.Trim(raw, `"'`)
}

func (p *PythonPlugin) Queries() LanguageQueries {
	return LanguageQueries{

		Route: `
(decorated_definition
  (decorator
    (call
      function: (attribute
        object: (identifier) @app
        attribute: (identifier) @method)
      arguments: (argument_list
        (string) @route_path)))
  definition: (function_definition
    name: (identifier) @fn_name
    parameters: (parameters) @params
    body: (block) @body) @handlers)
`,

		FuncDecl: `
(function_definition
  name: (identifier) @fn_name
  parameters: (parameters) @params
  body: (block) @body)
`,
		MountPoint: `
(call
  function: (attribute
    object: (identifier) @app
    attribute: (identifier) @mount_fn)
  arguments: (argument_list
    (identifier) @router_ref
    (keyword_argument
      name: (identifier) @prefix_kw
      value: (string) @prefix)))
`,

		RouterCreation: `
(assignment
  left: (identifier) @router_name
  right: (call
    function: (identifier) @router_type
    arguments: (argument_list
      (keyword_argument
        name: (identifier) @prefix_kw
        value: (string) @prefix))))
`,
		ArrowFunc:       "",
		ObjectMethod:    "",
		CJSExports:      "",
		CJSDirectExport: "",
		ESMExport:       "",
		RequireImport:   "",

		ESMImport: `
(import_from_statement
  module_name: (dotted_name) @import_path
  name: [(dotted_name) @named_import
         (aliased_import
           name: (dotted_name) @named_import
           alias: (identifier) @alias)])
`,

		ReqBody: `
(call
  function: (attribute
    object: (identifier) @req
    attribute: (identifier) @method)
  arguments: (argument_list))
`,

		ReqBodyDestructure: `
(attribute
  object: (identifier) @obj
  attribute: (identifier) @field)
`,

		ReqParams: `
(function_definition
  parameters: (parameters
    (typed_parameter
      (identifier) @param)))
`,

		ReqQuery: `
(function_definition
  parameters: (parameters
    (typed_default_parameter
      (identifier) @field
      type: (_) @type)))
`,

		ZodObject: "",
		ZodField:  "",
	}
}
func (p *PythonPlugin) PostProcessRoute(route Route) Route {
	schema := route.Schema

	allParams := append(schema.PathParams, schema.QueryParams...)

	seen := map[string]bool{}
	var unique []string
	for _, param := range allParams {
		if !seen[param] {
			seen[param] = true
			unique = append(unique, param)
		}
	}

	schema.PathParams = nil
	schema.QueryParams = nil

	for _, param := range unique {
		if p.isDependsParam(param, route.HandlerCode) {
			continue
		}
		if strings.Contains(route.Path, "{"+param+"}") {
			schema.PathParams = append(schema.PathParams, param)
			continue
		}
		schema.QueryParams = append(schema.QueryParams, param)
	}

	route.Schema = schema
	return route
}

func (p *PythonPlugin) isDependsParam(param string, handlerCode string) bool {
	for _, line := range strings.Split(handlerCode, "\n") {
		if strings.Contains(line, param) && strings.Contains(line, "Depends(") {
			return true
		}
	}
	return false
}
