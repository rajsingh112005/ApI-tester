package discovery

import (
	"strings"
	"api-tester/backend/helpers"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

func (e *Extractor) ResolveHandlers(routes []Route, table SymbolTable, baseDir string) []Route {
	for i, route := range routes {
		if route.HandlerCode != "" {
			continue
		}
		if route.HandlerName == "" {
			continue
		}

		name := route.HandlerName

		if strings.Contains(name, ".") {
			parts := strings.SplitN(name, ".", 2)
			name = parts[1]
		}

		sym, found := table[name]
		if !found {
			continue
		}

		routes[i].HandlerCode = sym.Code
		routes[i].HandlerFile = sym.File

		tree, src, err := e.ParseFile(sym.File)
		if err != nil {
			continue
		}

		fnNode := e.findFunctionNode(tree.RootNode(), src, name)
		if fnNode != nil {
			routes[i].Schema = e.ExtractSchema(fnNode, src)
		}
		tree.Close()
	}

	return routes
}

func (e *Extractor) findFunctionNode(root *tree_sitter.Node, src []byte, name string) *tree_sitter.Node {
	queries := e.plugin.Queries()

	for _, queryStr := range []string{
		queries.FuncDecl,
		queries.ArrowFunc,
		queries.ObjectMethod,
		queries.ObjectPropFunc,
	} {
		if queryStr == "" {
			continue
		}

		q, err := tree_sitter.NewQuery(e.lang, queryStr)
		if err != nil {
			continue
		}

		cursor := tree_sitter.NewQueryCursor()
		matches := cursor.Matches(q, root, src)

		for m := matches.Next(); m != nil; m = matches.Next() {
			captures := helpers.GetCaptureMap(m, q, src)
			if captures["fn_name"] == name {
				node := helpers.GetFirstCaptureNode(m, q, "body")
				cursor.Close()
				q.Close()
				return node
			}
		}

		cursor.Close()
		q.Close()
	}

	return nil
}
