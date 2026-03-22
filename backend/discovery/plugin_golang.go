package discovery

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

type GolangPlugin struct{}

func (p *GolangPlugin) Language() *tree_sitter.Language {
	return tree_sitter.NewLanguage(tree_sitter_go.Language())
}

func (p *GolangPlugin) IsHTTPMethod(method string) bool {
	methods := map[string]bool{
		"GET":    true,
		"POST":   true,
		"PUT":    true,
		"PATCH":  true,
		"DELETE": true,
		"Get":    true,
		"Post":   true,
		"Put":    true,
		"Patch":  true,
		"Delete": true,
	}
	return methods[method]
}

func (p *GolangPlugin) CleanRoutePath(raw string) string {
	return strings.Trim(raw, `"`)
}

func (p *GolangPlugin) Queries() LanguageQueries {
	return LanguageQueries{

		Route: `
(call_expression
  function: (selector_expression
    operand: (identifier) @router
    field: (field_identifier) @method)
  arguments: (argument_list
    . (interpreted_string_literal) @route_path
    (_)+ @handlers))
`,

		FuncDecl: `
(function_declaration
  name: (identifier) @fn_name
  parameters: (parameter_list) @params
  body: (block) @body)
`,

		ArrowFunc: "",

		ObjectMethod: `
(method_declaration
  receiver: (parameter_list
    (parameter_declaration
      name: (identifier) @receiver_name
      type: (_) @receiver_type))
  name: (field_identifier) @fn_name
  parameters: (parameter_list) @params
  body: (block) @body)
`,
		MountPoint: `
(call_expression
  function: (selector_expression
    operand: (identifier) @router
    field: (field_identifier) @group_fn)
  arguments: (argument_list
    (interpreted_string_literal) @prefix
    (_) @group_body))
`,

		ObjectPropFunc:  "",
		CJSExports:      "",
		CJSDirectExport: "",

		ESMExport:     "",
		RequireImport: "",

		ESMImport: `
(import_declaration
  (import_spec_list
    (import_spec
      name: (package_identifier)? @alias
      path: (interpreted_string_literal) @import_path)))
`,

		ReqBodyDestructure: `
(type_declaration
  (type_spec
    name: (type_identifier) @struct_name
    type: (struct_type
      (field_declaration_list
        (field_declaration
          name: (field_identifier) @field
          type: (_) @field_type
          tag: (raw_string_literal)? @tag)))))
`,

		ReqParams: `
(call_expression
  function: (selector_expression
    operand: (identifier) @ctx
    field: (field_identifier) @method)
  arguments: (argument_list
    (interpreted_string_literal) @param))
`,

		ReqQuery: `
(call_expression
  function: (selector_expression
    operand: (identifier) @ctx
    field: (field_identifier) @method)
  arguments: (argument_list
    (interpreted_string_literal) @field))
`,

		ReqBody: `
(call_expression
  function: (selector_expression
    operand: (identifier) @ctx
    field: (field_identifier) @method)
  arguments: (argument_list
    (unary_expression
      operand: (identifier) @body_var)))
`,

		ZodObject: "",
		ZodField:  "",
	}
}

func (p *GolangPlugin) PostProcessRoute(route Route) Route {
	schema := route.Schema
	var realPathParams []string
	for _, param := range schema.PathParams {
		if strings.Contains(route.HandlerCode, `c.Param("`+param+`"`) ||
			strings.Contains(route.HandlerCode, `ctx.Param("`+param+`"`) {
			realPathParams = append(realPathParams, param)
		}
	}
	schema.PathParams = realPathParams

	var realQueryParams []string
	for _, field := range schema.QueryParams {
		if strings.Contains(route.HandlerCode, `c.Query("`+field+`"`) ||
			strings.Contains(route.HandlerCode, `c.DefaultQuery("`+field+`"`) ||
			strings.Contains(route.HandlerCode, `ctx.Query("`+field+`"`) {
			realQueryParams = append(realQueryParams, field)
		}
	}
	schema.QueryParams = realQueryParams

	route.Schema = schema
	return route
}
