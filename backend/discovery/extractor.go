package discovery

import (
	"api-tester/backend/helpers"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type Extractor struct {
	plugin LanguagePlugin
	parser *tree_sitter.Parser
	lang   *tree_sitter.Language
}

func NewExtractor(plugin LanguagePlugin) *Extractor {
	p := tree_sitter.NewParser()
	lang := plugin.Language()
	if err := p.SetLanguage(lang); err != nil {
		panic(fmt.Sprintf("failed to set language: %v", err))
	}
	return &Extractor{
		plugin: plugin,
		parser: p,
		lang:   lang,
	}
}

func (e *Extractor) ParseFile(path string) (*tree_sitter.Tree, []byte, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	tree := e.parser.Parse(src, nil)
	return tree, src, nil
}

func (e *Extractor) ExtractRoutes(filePath string) ([]Route, error) {
	tree, src, err := e.ParseFile(filePath)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	queries := e.plugin.Queries()
	if queries.Route == "" {
		return nil, nil
	}

	// NewQuery now returns *QueryError not error
	// access .Message .Row .Column directly for real error info
	q, qErr := tree_sitter.NewQuery(e.lang, queries.Route)
	if qErr != nil {
		return nil, fmt.Errorf("route query failed at row %d col %d: %s",
			qErr.Row, qErr.Column, qErr.Message)
	}
	defer q.Close()

	cursor := tree_sitter.NewQueryCursor()
	defer cursor.Close()

	var routes []Route
	matches := cursor.Matches(q, tree.RootNode(), src)

	for match := matches.Next(); match != nil; match = matches.Next() {
		captures := helpers.GetCaptureMap(match, q, src)

		method := captures["method"]
		if !e.plugin.IsHTTPMethod(method) {
			continue
		}

		route := Route{
			Method: strings.ToUpper(method),
			Path:   e.plugin.CleanRoutePath(captures["route_path"]),
			File:   filePath,
		}

		handlerNode := helpers.GetLastCapture(match, q, "handlers")
		if handlerNode != nil {
			route.HandlerCode, route.HandlerName = e.resolveHandlerNode(handlerNode, src)
			if route.HandlerCode != "" {
				route.Schema = e.ExtractSchema(handlerNode, src)
			}
		}

		if route.HandlerName == "" {
			if fnName, ok := captures["fn_name"]; ok && fnName != "" {
				route.HandlerName = fnName
			}
		}

		route = e.plugin.PostProcessRoute(route)

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes, nil
}

func (e *Extractor) resolveHandlerNode(node *tree_sitter.Node, src []byte) (string, string) {
	switch node.Kind() {
	case "arrow_function", "function_expression":
		return node.Utf8Text(src), ""
	case "function_declaration":
		return node.Utf8Text(src), ""
	case "identifier":
		return "", node.Utf8Text(src)
	case "member_expression":
		return "", node.Utf8Text(src)
	case "selector_expression":
		return "", node.Utf8Text(src)
	default:
		return node.Utf8Text(src), ""
	}
}

// ─── SCHEMA EXTRACTION ───────────────────────────────────────────────────────

func (e *Extractor) ExtractSchema(node *tree_sitter.Node, src []byte) Schema {
	var schema Schema
	seen := map[string]bool{}
	queries := e.plugin.Queries()

	// kwFilter applied in Go since #eq? predicate is unreliable in go-tree-sitter
	// kwCapture is the capture name to check, kwFilter is the required value
	collectFiltered := func(queryStr, captureName, kwCapture, kwFilter string, target *[]string) {
		if queryStr == "" {
			return
		}
		q, qErr := tree_sitter.NewQuery(e.lang, queryStr)
		if qErr != nil {
			fmt.Fprintf(os.Stderr, "schema query failed at row %d col %d: %s\n",
				qErr.Row, qErr.Column, qErr.Message)
			return
		}
		defer q.Close()

		cursor := tree_sitter.NewQueryCursor()
		defer cursor.Close()

		matches := cursor.Matches(q, node, src)
		for m := matches.Next(); m != nil; m = matches.Next() {
			captures := helpers.GetCaptureMap(m, q, src)

			// Go-side keyword filter replaces unreliable #eq? predicate
			if kwFilter != "" && captures[kwCapture] != kwFilter {
				continue
			}

			if val, ok := captures[captureName]; ok {
				key := captureName + ":" + val
				if !seen[key] {
					seen[key] = true
					*target = append(*target, val)
				}
			}
		}
	}

	// filter @kw == "body" / "params" / "query" in Go
	collectFiltered(queries.ReqBody, "field", "kw", "body", &schema.BodyFields)
	collectFiltered(queries.ReqBodyDestructure, "field", "kw", "body", &schema.BodyFields)
	collectFiltered(queries.ReqParams, "param", "kw", "params", &schema.PathParams)
	collectFiltered(queries.ReqQuery, "field", "kw", "query", &schema.QueryParams)

	if queries.ZodObject != "" && queries.ZodField != "" {
		schema.ZodFields = e.extractZodFields(node, src)
	}

	return schema
}

func (e *Extractor) extractZodFields(node *tree_sitter.Node, src []byte) []ZodField {
	queries := e.plugin.Queries()
	var zodFields []ZodField

	q, qErr := tree_sitter.NewQuery(e.lang, queries.ZodObject)
	if qErr != nil {
		fmt.Fprintf(os.Stderr, "zod object query failed at row %d col %d: %s\n",
			qErr.Row, qErr.Column, qErr.Message)
		return nil
	}
	defer q.Close()

	cursor := tree_sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(q, node, src)
	for m := matches.Next(); m != nil; m = matches.Next() {
		captures := helpers.GetCaptureMap(m, q, src)

		if captures["z"] != "z" || captures["method"] != "object" {
			continue
		}

		schemaObjNode := helpers.GetFirstCaptureNode(m, q, "schema_obj")
		if schemaObjNode == nil {
			continue
		}

		fq, qErr := tree_sitter.NewQuery(e.lang, queries.ZodField)
		if qErr != nil {
			fmt.Fprintf(os.Stderr, "zod field query failed at row %d col %d: %s\n",
				qErr.Row, qErr.Column, qErr.Message)
			continue
		}

		fcursor := tree_sitter.NewQueryCursor()
		fmatches := fcursor.Matches(fq, schemaObjNode, src)

		for fm := fmatches.Next(); fm != nil; fm = fmatches.Next() {
			fc := helpers.GetCaptureMap(fm, fq, src)
			if fc["field_name"] != "" && fc["zod_type"] != "" {
				zodFields = append(zodFields, ZodField{
					Name:    fc["field_name"],
					ZodType: fc["zod_type"],
				})
			}
		}
		fcursor.Close()
		fq.Close()
	}

	return zodFields
}

func (e *Extractor) BuildSymbolTable(files []string) SymbolTable {
	table := make(SymbolTable)
	queries := e.plugin.Queries()

	for _, file := range files {
		tree, src, err := e.ParseFile(file)
		if err != nil {
			continue
		}

		for _, queryStr := range []string{
			queries.FuncDecl,
			queries.ArrowFunc,
			queries.ObjectMethod,
			queries.ObjectPropFunc,
		} {
			if queryStr == "" {
				continue
			}

			q, qErr := tree_sitter.NewQuery(e.lang, queryStr)
			if qErr != nil {
				fmt.Fprintf(os.Stderr, "symbol query failed at row %d col %d: %s\n",
					qErr.Row, qErr.Column, qErr.Message)
				continue
			}

			cursor := tree_sitter.NewQueryCursor()
			matches := cursor.Matches(q, tree.RootNode(), src)

			for m := matches.Next(); m != nil; m = matches.Next() {
				captures := helpers.GetCaptureMap(m, q, src)
				name := captures["fn_name"]
				if name == "" {
					continue
				}
				code := captures["body"]
				if code == "" {
					code = helpers.GetFullFunctionText(m, src)
				}
				table[name] = Symbol{Name: name, Code: code, File: file}
			}

			cursor.Close()
			q.Close()
		}

		tree.Close()
	}

	return table
}
func (e *Extractor) ExtractMountPoints(filePath string) ([]MountPoint, error) {
	queries := e.plugin.Queries()
	if queries.MountPoint == "" {
		return nil, nil
	}

	tree, src, err := e.ParseFile(filePath)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	q, qErr := tree_sitter.NewQuery(e.lang, queries.MountPoint)
	if qErr != nil {
		return nil, fmt.Errorf("mount query failed: %s", qErr.Message)
	}
	defer q.Close()

	cursor := tree_sitter.NewQueryCursor()
	defer cursor.Close()

	var mounts []MountPoint
	matches := cursor.Matches(q, tree.RootNode(), src)

	for match := matches.Next(); match != nil; match = matches.Next() {
		captures := helpers.GetCaptureMap(match, q, src)

		// only collect app.use() calls
		if captures["use"] != "use" {
			continue
		}

		prefix := e.plugin.CleanRoutePath(captures["prefix"])
		routerRef := captures["router_ref"]

		// handle require('./routes/users') — extract file base name
		if strings.HasPrefix(routerRef, "require(") {
			routerRef = strings.TrimPrefix(routerRef, "require('")
			routerRef = strings.TrimPrefix(routerRef, `require("`)
			routerRef = strings.TrimSuffix(routerRef, "')")
			routerRef = strings.TrimSuffix(routerRef, `")`)
			routerRef = filepath.Base(routerRef)
			routerRef = strings.TrimSuffix(routerRef, filepath.Ext(routerRef))
		}

		if prefix != "" && routerRef != "" {
			mounts = append(mounts, MountPoint{
				Prefix:     prefix,
				RouterName: routerRef,
				File:       filePath,
			})
		}
		if captures["group_fn"] == "Group" {
			prefix := e.plugin.CleanRoutePath(captures["prefix"])
			if prefix != "" {
				mounts = append(mounts, MountPoint{
					Prefix:     prefix,
					RouterName: captures["router"],
					MountFn:    "Group",
					File:       filePath,
				})
			}
		}
	}

	return mounts, nil
}
func (e *Extractor) ExtractRouterPrefixes(filePath string) (map[string]string, error) {
	queries := e.plugin.Queries()
	if queries.RouterCreation == "" {
		return nil, nil
	}

	tree, src, err := e.ParseFile(filePath)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	q, qErr := tree_sitter.NewQuery(e.lang, queries.RouterCreation)
	if qErr != nil {
		return nil, fmt.Errorf("router creation query failed: %s", qErr.Message)
	}
	defer q.Close()

	cursor := tree_sitter.NewQueryCursor()
	defer cursor.Close()

	prefixes := map[string]string{}
	matches := cursor.Matches(q, tree.RootNode(), src)

	for match := matches.Next(); match != nil; match = matches.Next() {
		captures := helpers.GetCaptureMap(match, q, src)

		routerType := captures["router_type"]
		prefixKw := captures["prefix_kw"]
		routerName := captures["router_name"]
		prefix := e.plugin.CleanRoutePath(captures["prefix"])

		// only APIRouter and Blueprint create route groups
		if routerType != "APIRouter" && routerType != "Blueprint" {
			continue
		}

		// FastAPI uses "prefix", Flask uses "url_prefix"
		if prefixKw != "prefix" && prefixKw != "url_prefix" {
			continue
		}

		if routerName != "" && prefix != "" {
			prefixes[routerName] = prefix
		}
	}

	return prefixes, nil
}
