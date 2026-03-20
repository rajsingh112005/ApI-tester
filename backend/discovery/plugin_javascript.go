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

        // ── ROUTE ──────────────────────────────────────────────────────────
        // Handles all these patterns:
        //   app.get('/users', handler)
        //   app.post('/users', authMiddleware, handler)
        //   router.put('/users/:id', mw1, mw2, handler)
        // The . before route_path anchors it as the first argument
        // (_)+ captures one or more handlers after the path
        Route: `
(call_expression
  function: (member_expression
    object: (_) @app
    property: (property_identifier) @method)
  arguments: (arguments
    . [(string) (template_string)] @route_path
    (_)+ @handlers))
`,

        // ── FUNCTION DECLARATIONS ──────────────────────────────────────────
        // Handles:
        //   function getUser(req, res) { ... }
        //   async function getUser(req, res) { ... }
        FuncDecl: `
(function_declaration
  name: (identifier) @fn_name
  parameters: (formal_parameters) @params
  body: (statement_block) @body)
`,

        // ── ARROW FUNCTIONS / FUNCTION EXPRESSIONS ─────────────────────────
        // Handles:
        //   const getUser = (req, res) => { ... }
        //   const getUser = async (req, res) => { ... }
        //   const getUser = function(req, res) { ... }
        //   export const getUser = async (req, res) => { ... }
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

        // ── OBJECT METHOD SHORTHAND ────────────────────────────────────────
        // Handles:
        //   module.exports = { getUser(req, res) { ... } }
        ObjectMethod: `
(method_definition
  name: (property_identifier) @fn_name
  parameters: (formal_parameters) @params
  body: (statement_block) @body)
`,

        // ── OBJECT PROPERTY WITH FUNCTION VALUE ───────────────────────────
        // Handles:
        //   module.exports = { getUser: async (req, res) => { ... } }
        //   module.exports = { getUser: function(req, res) { ... } }
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
          // app.use('/api/users', usersRouter)
// app.use('/api/users', require('./routes/users'))
MountPoint: `
(call_expression
  function: (member_expression
    object: (_) @app
    property: (property_identifier) @use)
  arguments: (arguments
    (string) @prefix
    (_) @router_ref))
`,

// router.route('/users').get(handler).post(handler)
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
        // ── CJS EXPORTS ────────────────────────────────────────────────────
        // Handles:
        //   module.exports = { getUser, createUser }
        //   module.exports = { getUser: fn }
        // Filter @mod == "module" && @exp == "exports" in Go
        CJSExports: `
(assignment_expression
  left: (member_expression
    object: (identifier) @mod
    property: (property_identifier) @exp)
  right: (_) @export_value)
`,

        // ── CJS DIRECT EXPORT ──────────────────────────────────────────────
        // Handles:
        //   module.exports.getUser = function(req, res) { ... }
        CJSDirectExport: `
(assignment_expression
  left: (member_expression
    object: (member_expression
      object: (identifier) @mod
      property: (property_identifier) @exp)
    property: (property_identifier) @export_name)
  right: (_) @fn_value)
`,

        // ── ESM EXPORTS ────────────────────────────────────────────────────
        // Handles:
        //   export function getUser(...) { ... }
        //   export const getUser = (req, res) => { ... }
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

        // ── REQUIRE IMPORTS ────────────────────────────────────────────────
        // Handles:
        //   const getUser = require('./controllers/users')
        //   const { getUser, createUser } = require('./controllers/users')
        // Filter @req_fn == "require" in Go
        RequireImport: `
(variable_declarator
  name: (_) @import_binding
  value: (call_expression
    function: (identifier) @req_fn
    arguments: (arguments (string) @import_path)))
`,

        // ── ESM IMPORTS ────────────────────────────────────────────────────
        // Handles:
        //   import { getUser } from './controllers/users'
        //   import { getUser as fetchUser } from './controllers/users'
        ESMImport: `
(import_statement
  (import_clause
    (named_imports
      (import_specifier
        name: (identifier) @named_import
        alias: (identifier)? @alias)))
  source: (string) @import_path)
`,

        // ── SCHEMA: req.body.field ─────────────────────────────────────────
        // Handles:
        //   req.body.name
        //   req.body.email
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