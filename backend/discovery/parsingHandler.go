package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Response struct {
	Routes []Route `json:"routes"`
	Total  int     `json:"total"`
	Error  string  `json:"error,omitempty"`
}

func Handler(path string) Response {
	var res Response
	filesByLang := map[string][]string{}

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// skip node_modules and hidden directories
			base := filepath.Base(filePath)
			if base == "node_modules" || base == "vendor" || (len(base) > 0 && base[0] == '.') {
				return filepath.SkipDir
			}
			return nil
		}

		lang := DetectLanguage(filePath)
		if lang != "" {
			filesByLang[lang] = append(filesByLang[lang], filePath)
		}
		return nil
	})

	if err != nil {
		res.Error = fmt.Sprintf("error walking directory: %v", err)
		return res
	}

	var allRoutes []Route

	// step 2 — for each language group, run its plugin
	for lang, files := range filesByLang {
		plugin := GetPlugin(lang)
		if plugin == nil {
			fmt.Fprintf(os.Stderr, "no plugin for language: %s — skipping\n", lang)
			continue
		}

		extractor := NewExtractor(plugin)

		fmt.Fprintf(os.Stderr, "indexing %d %s files...\n", len(files), lang)
		table := extractor.BuildSymbolTable(files)
		fmt.Fprintf(os.Stderr, "found %d symbols\n", len(table))
		routerPrefixes := map[string]string{}
		for _, file := range files {
			prefixes, err := extractor.ExtractRouterPrefixes(file)
			if err != nil {
				continue
			}
			for name, prefix := range prefixes {
				routerPrefixes[name] = prefix
			}
		}
		var allMounts []MountPoint
		for _, file := range files {
			mounts, err := extractor.ExtractMountPoints(file)
			if err != nil {
				continue
			}
			allMounts = append(allMounts, mounts...)
		}

		// STEP 2 — extract routes then apply prefixes
		for _, file := range files {
			routes, err := extractor.ExtractRoutes(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error parsing %s: %v\n", file, err)
				continue
			}

			routes = extractor.ResolveHandlers(routes, table, path)

			// apply mount prefix to each route
			for i := range routes {
				routes[i].Path = resolveMountPrefix(routes[i], allMounts)
			}

			allRoutes = append(allRoutes, routes...)
		}
	}

	res.Routes = allRoutes
	res.Total = len(allRoutes)
	return res
}

func resolveMountPrefix(route Route, mounts []MountPoint) string {
	routeBase := strings.TrimSuffix(
		filepath.Base(route.File),
		filepath.Ext(route.File),
	)

	for _, mount := range mounts {
		if strings.EqualFold(mount.RouterName, routeBase) {
			return mount.Prefix + route.Path
		}
	}

	return route.Path
}
