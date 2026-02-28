package discovery

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Route struct {
	ID         string   `json:"id"`
	Method     string   `json:"method"`
	Path       string   `json:"path"`
	File       string   `json:"file"`
	Handler    string   `json:"handler"`
	Framework  string   `json:"framework"`
	Middleware []string `json:"middleware,omitempty"`
}

var (
	reExpress = regexp.MustCompile(`\b(app|router)\.(get|post|put|patch|delete)\s*\(\s*['"]([^'"]+)['"]`)
	reGoHTTP  = regexp.MustCompile(`\bHandle(Func)?\s*\(\s*["']([^"']+)["']`)
	reDjango  = regexp.MustCompile(`\bpath\s*\(\s*['"]([^'"]+)['"]\s*,\s*([a-zA-Z0-9_\.]+)`)
)

func Discover(projectRoot string) ([]Route, error) {
	var routes []Route
	err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".ccc" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".js", ".ts", ".go", ".py":
			found, ferr := scanFile(path)
			if ferr != nil {
				return ferr
			}
			routes = append(routes, found...)
		}
		return nil
	})
	return routes, err
}

func scanFile(path string) ([]Route, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var routes []Route
	s := bufio.NewScanner(f)
	lineNo := 0
	for s.Scan() {
		lineNo++
		line := s.Text()
		for _, m := range reExpress.FindAllStringSubmatch(line, -1) {
			routes = append(routes, Route{
				ID:        id(path, lineNo, strings.ToUpper(m[2]), m[3]),
				Method:    strings.ToUpper(m[2]),
				Path:      m[3],
				File:      path,
				Handler:   "inline_handler",
				Framework: "node",
			})
		}
		for _, m := range reGoHTTP.FindAllStringSubmatch(line, -1) {
			routes = append(routes, Route{
				ID:        id(path, lineNo, "ANY", m[2]),
				Method:    "ANY",
				Path:      m[2],
				File:      path,
				Handler:   "handler",
				Framework: "go",
			})
		}
		for _, m := range reDjango.FindAllStringSubmatch(line, -1) {
			routes = append(routes, Route{
				ID:        id(path, lineNo, "ANY", "/"+strings.TrimLeft(m[1], "/")),
				Method:    "ANY",
				Path:      "/" + strings.TrimLeft(m[1], "/"),
				File:      path,
				Handler:   m[2],
				Framework: "django",
			})
		}
	}
	return routes, s.Err()
}

func id(file string, line int, method, path string) string {
	return filepath.Base(file) + ":" + strings.ToLower(method) + ":" + path + ":" + strconv.Itoa(line)
}
