package discovery

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

type JavaScriptPlugin struct{}

func (p *JavaScriptPlugin) Language() *tree_sitter.Language {
	return tree_sitter.NewLanguage(tree_sitter_javascript.Language())
}

func (p *JavaScriptPlugin) IsHTTPMethod(method string) bool {
	methods := map[string]bool{
		"get": true, "post": true, "put": true,
		"patch": true, "delete": true, "use": true, "all": true,
	}
	return methods[strings.ToLower(method)]
}
func (p *JavaScriptPlugin) CleanRoutePath(raw string) string {
	return strings.Trim(raw, "\"'`")
}

func (p *JavaScriptPlugin) Queries() LanguageQueries {
	return LanguageQueries{

		Route: `
(call_expression
  function: (member_expression
    object: (_) @app
    property: (property_identifier) @method)
  arguments: (arguments
    . [(string) (template_string)] @route_path
    (_)+ @handlers))
`,

		FuncDecl: `
(function_declaration
  name: (identifier) @fn_name
  parameters: (formal_parameters) @params
  body: (statement_block) @body)
`,

		ArrowFunc: `
(variable_declarator
  name: (identifier) @fn_name
  value: [(arrow_function
             parameters: (formal_parameters) @params
             body: (_) @body)
          (function_expression
             parameters: (formal_parameters) @params
             body: (statement_block) @body)])
`,

		ObjectMethod: `
(method_definition
  name: (property_identifier) @fn_name
  parameters: (formal_parameters) @params
  body: (statement_block) @body)
`,

		ObjectPropFunc: `
(pair
  key: [(identifier) (string) (property_identifier)] @fn_name
  value: [(arrow_function
             parameters: (formal_parameters) @params
             body: (_) @body)
          (function_expression
             parameters: (formal_parameters) @params
             body: (statement_block) @body)])
`,

		MountPoint: `
(call_expression
  function: (member_expression
    object: (_) @app
    property: (property_identifier) @use)
  arguments: (arguments
    (string) @prefix
    (_) @router_ref))
`,

		RouterCreation: `
(call_expression
  function: (member_expression
    object: (call_expression
      function: (member_expression
        object: (_) @router
        property: (property_identifier) @route_fn)
      arguments: (arguments
        (string) @route_path))
    property: (property_identifier) @method)
  arguments: (arguments
    (_) @handler))
`,

		CJSExports: `
(assignment_expression
  left: (member_expression
    object: (identifier) @mod
    property: (property_identifier) @exp)
  right: (_) @export_value)
`,

		CJSDirectExport: `
(assignment_expression
  left: (member_expression
    object: (member_expression
      object: (identifier) @mod
      property: (property_identifier) @exp)
    property: (property_identifier) @export_name)
  right: (_) @fn_value)
`,

		ESMExport: `
[(export_statement
    declaration: (function_declaration
      name: (identifier) @export_name))
 (export_statement
    declaration: (lexical_declaration
      (variable_declarator
        name: (identifier) @export_name
        value: [(arrow_function) (function_expression)] @fn_value)))]
`,

		RequireImport: `
(variable_declarator
  name: (_) @import_binding
  value: (call_expression
    function: (identifier) @req_fn
    arguments: (arguments (string) @import_path)))
`,

		ESMImport: `
(import_statement
  (import_clause
    (named_imports
      (import_specifier
        name: (identifier) @named_import
        alias: (identifier)? @alias)))
  source: (string) @import_path)
`,

		ReqBody: `
(member_expression
  object: (member_expression
    object: (identifier) @req
    property: (property_identifier) @kw)
  property: (property_identifier) @field)
`,

		ReqBodyDestructure: `
(variable_declarator
  name: (object_pattern
    (shorthand_property_identifier_pattern) @field)
  value: (member_expression
    object: (identifier) @req
    property: (property_identifier) @kw))
`,

		ReqParams: `
(member_expression
  object: (member_expression
    object: (identifier) @req
    property: (property_identifier) @kw)
  property: (property_identifier) @param)
`,

		ReqQuery: `
(member_expression
  object: (member_expression
    object: (identifier) @req
    property: (property_identifier) @kw)
  property: (property_identifier) @field)
`,

		ZodObject: `
(call_expression
  function: (member_expression
    object: (identifier) @z
    property: (property_identifier) @method)
  arguments: (arguments (object) @schema_obj))
`,

		ZodField: `
(pair
  key: [(identifier) (string) (property_identifier)] @field_name
  value: (call_expression
    function: (member_expression
      object: (identifier) @z2
      property: (property_identifier) @zod_type)))
`,
	}
}

func (p *JavaScriptPlugin) PostProcessRoute(route Route) Route {
	return route
}
