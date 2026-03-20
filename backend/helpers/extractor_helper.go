package helpers

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

func GetFullFunctionText(match *tree_sitter.QueryMatch, src []byte) string {
	if len(match.Captures) == 0 {
		return ""
	}
	return match.Captures[0].Node.Utf8Text(src)
}

func GetCaptureMap(match *tree_sitter.QueryMatch, q *tree_sitter.Query, src []byte) map[string]string {
	result := make(map[string]string)
	for _, cap := range match.Captures {
		name := q.CaptureNames()[cap.Index]
		if _, exists := result[name]; !exists {
			result[name] = cap.Node.Utf8Text(src)
		}
	}
	return result
}

func GetLastCapture(match *tree_sitter.QueryMatch, q *tree_sitter.Query, captureName string) *tree_sitter.Node {
	var last *tree_sitter.Node
	for _, cap := range match.Captures {
		if q.CaptureNames()[cap.Index] == captureName {
			last = &cap.Node
		}
	}
	return last
}

func GetFirstCaptureNode(match *tree_sitter.QueryMatch, q *tree_sitter.Query, captureName string) *tree_sitter.Node {
	for _, cap := range match.Captures {
		if q.CaptureNames()[cap.Index] == captureName {
			return &cap.Node
		}
	}
	return nil
}
