package discovery
import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

type GolangPlugin struct{}

// Language returns the Go grammar for tree-sitter to parse with
func (p *GolangPlugin) Language() *tree_sitter.Language {
	return tree_sitter.NewLanguage(tree_sitter_go.Language())
}

// IsHTTPMethod filters field names that represent HTTP methods
// Gin uses uppercase:  r.GET  r.POST  r.PUT  r.PATCH  r.DELETE
// Echo uses uppercase: e.GET  e.POST  e.PUT  e.PATCH  e.DELETE
// Chi uses title case: r.Get  r.Post  r.Put  r.Patch  r.Delete
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

// CleanRoutePath strips surrounding double quotes from Go string literals
func (p *GolangPlugin) CleanRoutePath(raw string) string {
	return strings.Trim(raw, `"`)
}

func (p *GolangPlugin) Queries() LanguageQueries {
	return LanguageQueries{

		// ── ROUTE ──────────────────────────────────────────────────────────
		// Handles all these patterns:
		//
		// Gin:
		//   r.GET("/users", GetUsers)
		//   r.POST("/users", authMiddleware, CreateUser)
		//   r.PUT("/users/:id", UpdateUser)
		//
		// Echo:
		//   e.GET("/users", GetUsers)
		//   e.POST("/users", CreateUser)
		//
		// Chi:
		//   r.Get("/users", GetUsers)
		//   r.Post("/users", CreateUser)
		//
		// The router variable (r, e, router) is captured as @router.
		// Method name (GET, POST etc) captured as @method.
		// Route path string captured as @route_path.
		// All handler arguments captured as @handlers.
		Route: `
(call_expression
  function: (selector_expression
    operand: (identifier) @router
    field: (field_identifier) @method)
  arguments: (argument_list
    . (interpreted_string_literal) @route_path
    (_)+ @handlers))
`,

		// ── FUNCTION DECLARATIONS ──────────────────────────────────────────
		// Handles top-level handler functions:
		//   func GetUsers(c *gin.Context) { ... }
		//   func CreateUser(c *gin.Context) { ... }
		//   func GetUsers(w http.ResponseWriter, r *http.Request) { ... }
		//
		// Go uses function_declaration for all top-level functions.
		// No async keyword — Go uses goroutines instead.
		FuncDecl: `
(function_declaration
  name: (identifier) @fn_name
  parameters: (parameter_list) @params
  body: (block) @body)
`,

		// ── NO ARROW FUNCTIONS ─────────────────────────────────────────────
		// Go has no arrow functions.
		// Anonymous functions exist but are rarely used as route handlers.
		ArrowFunc: "",

		// ── METHOD DECLARATIONS (struct receivers) ─────────────────────────
		// Handles handler methods on controller structs:
		//   func (h *UserHandler) GetUsers(c *gin.Context) { ... }
		//   func (h *UserHandler) CreateUser(c *gin.Context) { ... }
		//
		// @receiver captures the struct type
		// @fn_name captures the method name
		// @body captures the method body
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
        // r.Group('/api', func(r) { ... })
MountPoint: `
(call_expression
  function: (selector_expression
    operand: (identifier) @router
    field: (field_identifier) @group_fn)
  arguments: (argument_list
    (interpreted_string_literal) @prefix
    (_) @group_body))
`,
		// ── NO OBJECT PROP FUNCTIONS ───────────────────────────────────────
		// Go has no object literal functions like JS.
		ObjectPropFunc: "",

		// ── NO CJS EXPORTS ─────────────────────────────────────────────────
		// Go has no module.exports pattern.
		// Functions are exported by capitalizing their name.
		CJSExports:      "",
		CJSDirectExport: "",

		// ── NO ESM EXPORTS ─────────────────────────────────────────────────
		// Go has no explicit export statements.
		// Capitalized identifiers are automatically exported package-wide.
		ESMExport: "",

		// ── NO REQUIRE IMPORTS ─────────────────────────────────────────────
		// Go has no require() function.
		RequireImport: "",

		// ── IMPORTS ────────────────────────────────────────────────────────
		// Handles:
		//   import "github.com/user/project/controllers"
		//
		//   import (
		//       "github.com/user/project/controllers"
		//       userCtrl "github.com/user/project/controllers/user"
		//   )
		//
		// @import_path captures the full import path string.
		// @alias captures the optional local alias.
		ESMImport: `
(import_declaration
  (import_spec_list
    (import_spec
      name: (package_identifier)? @alias
      path: (interpreted_string_literal) @import_path)))
`,


		// ── SCHEMA: struct type declarations ──────────────────────────────
		// Handles request body struct definitions:
		//   type CreateUserRequest struct {
		//       Name  string `json:"name"`
		//       Email string `json:"email"`
		//   }
		//
		// @struct_name captures the type name (e.g. "CreateUserRequest")
		// @field captures each field name
		// @field_type captures the Go type
		// @tag captures the json struct tag if present
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

		// ── NO ZOD SCHEMA ──────────────────────────────────────────────────
		// Go uses struct tags and libraries like validator for validation,
		// not Zod. Struct fields are captured via ReqBodyDestructure above.
		ZodObject: "",
		ZodField:  "",
	}
}

func (p *GolangPlugin) PostProcessRoute(route Route) Route {
	schema := route.Schema
	var realPathParams []string
	for _, param := range schema.PathParams {
		// param is valid if it was captured from c.Param() call
		// we check by looking at handler code
		if strings.Contains(route.HandlerCode, `c.Param("`+param+`"`) ||
			strings.Contains(route.HandlerCode, `ctx.Param("`+param+`"`) {
			realPathParams = append(realPathParams, param)
		}
	}
	schema.PathParams = realPathParams

	// same for query params — keep only c.Query() results
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