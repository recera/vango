package router

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// ===== Internal helpers for radix tree construction =====

type genNode struct {
	label     string
	kind      int // 0 static, 1 param, 2 catchAll
	paramName string
	paramType string
	handler   *Route
	statics   []*genNode
	param     *genNode
	catchAll  *genNode
}

func newGenNode() *genNode { return &genNode{} }

func (n *genNode) insertRoute(r Route) error {
	path := r.Path
	if path == "" || path[0] != '/' {
		return fmt.Errorf("route path must start with '/': %s", path)
	}
	i := 1
	cur := n
	for i <= len(path) {
		if i > len(path) {
			break
		}
		if path[i-1] == '[' {
			j := i - 1
			for j < len(path) && path[j] != ']' {
				j++
			}
			if j >= len(path) {
				return fmt.Errorf("unterminated param in path: %s", path)
			}
			raw := path[i-1+1 : j]
			if strings.HasPrefix(raw, "...") {
				name := raw[3:]
				if cur.catchAll != nil {
					return fmt.Errorf("duplicate catch-all at %s", path)
				}
				cur.catchAll = &genNode{kind: 2, paramName: name}
				cur = cur.catchAll
				i = j + 2 // skip "]/"
				break
			}
			name := raw
			ptype := "string"
			if k := strings.Index(raw, ":"); k != -1 {
				name = raw[:k]
				ptype = raw[k+1:]
			}
			if cur.param == nil {
				cur.param = &genNode{kind: 1, paramName: name, paramType: ptype}
			} else if cur.param.paramName != name || cur.param.paramType != ptype {
				return fmt.Errorf("param conflict at %s", path)
			}
			cur = cur.param
			i = j + 2
			continue
		}
		start := i - 1
		for i <= len(path) && path[i-1] != '[' {
			i++
		}
		label := strings.TrimSuffix(path[start:i-1], "/")
		if label != "" {
			var next *genNode
			for _, s := range cur.statics {
				if s.label == label {
					next = s
					break
				}
			}
			if next == nil {
				next = &genNode{kind: 0, label: label}
				cur.statics = append(cur.statics, next)
			}
			cur = next
		}
		if i > len(path) {
			break
		}
	}
	if cur.handler != nil {
		return fmt.Errorf("duplicate handler for path %s (already defined in %s)", r.Path, cur.handler.FilePath)
	}
	cur.handler = &r
	return nil
}

func (n *genNode) compressStatic() {
	for _, s := range n.statics {
		s.compressStatic()
	}
	if len(n.statics) == 1 && n.param == nil && n.catchAll == nil && n.handler == nil && n.kind == 0 {
		child := n.statics[0]
		if child.kind == 0 {
			n.label = n.label + child.label
			n.statics = child.statics
			n.param = child.param
			n.catchAll = child.catchAll
			n.handler = child.handler
			n.compressStatic()
		}
	}
}

// ===== Module/import helpers =====

func detectModulePath() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(string(data), "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(l, "module ")), nil
		}
	}
	return "", errors.New("module path not found in go.mod")
}

func packageAliasFromImport(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "pkg"
	}
	alias := parts[len(parts)-1]
	alias = strings.ReplaceAll(alias, "-", "_")
	if alias == "router" {
		return "routes"
	}
	return alias
}

// collectWrappersFor returns middleware and layout expression lists for a route file
func (g *CodeGenerator) collectWrappersFor(routeFile string) ([]string, []string) {
	var mws []string
	var layouts []string
	dir := filepath.Dir(routeFile)
	baseRoot, _ := filepath.Abs(g.routesDir)
	for {
		absDir, _ := filepath.Abs(dir)
		if !strings.HasPrefix(absDir, baseRoot) {
			break
		}
		mwPath := filepath.Join(dir, "_middleware.go")
		if _, err := os.Stat(mwPath); err == nil {
			importPath := filepath.ToSlash(filepath.Join(g.modulePath, dir))
			alias := packageAliasFromImport(importPath)
			// require exported Middleware() server.Middleware
			mws = append([]string{fmt.Sprintf("%s.Middleware()", alias)}, mws...)
		}
		layoutPath := filepath.Join(dir, "_layout.go")
		if _, err := os.Stat(layoutPath); err == nil {
			importPath := filepath.ToSlash(filepath.Join(g.modulePath, dir))
			alias := packageAliasFromImport(importPath)
			// require exported Layout(vdom.VNode) vdom.VNode
			layouts = append(layouts, fmt.Sprintf("%s.Layout", alias))
		}
		if absDir == baseRoot {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return mws, layouts
}

// ===== Emission =====

func (g *CodeGenerator) generateRadixTreeV2() error {
	// Build and validate tree
	root := newGenNode()
	for _, r := range g.routes {
		if err := root.insertRoute(r); err != nil {
			return err
		}
	}
	root.compressStatic()

	// Prepare imports and wrappers
	pkgImports := map[string]struct{}{
		fmt.Sprintf("%s/pkg/server", g.modulePath):        {},
		fmt.Sprintf("%s/pkg/vango/vdom", g.modulePath):    {},
		fmt.Sprintf("%s/pkg/renderer/html", g.modulePath): {},
		"net/http": {},
		"strings":  {},
	}

	// Detect special handlers (_404.go, _500.go) at routes root
	var notFoundAlias string
	var errorAlias string
	{
		rootImport := filepath.ToSlash(filepath.Join(g.modulePath, g.routesDir))
		if fileExists(filepath.Join(g.routesDir, "_404.go")) {
			pkgImports[rootImport] = struct{}{}
			notFoundAlias = packageAliasFromImport(rootImport)
		}
		if fileExists(filepath.Join(g.routesDir, "_500.go")) {
			pkgImports[rootImport] = struct{}{}
			errorAlias = packageAliasFromImport(rootImport)
		}
	}

	type wrapperSpec struct {
		FuncName        string
		ImportAlias     string
		ImportPath      string
		HandlerIdent    string
		IsAPI           bool
		Path            string
		MiddlewareExprs []string
		LayoutExprs     []string
	}
	var wrappers []wrapperSpec

	for _, r := range g.routes {
		relDir := filepath.Dir(strings.TrimPrefix(r.FilePath, g.routesDir))
		if relDir == "." {
			relDir = ""
		}
		importPath := filepath.ToSlash(filepath.Join(g.modulePath, g.routesDir, relDir))
		if relDir == "" {
			importPath = filepath.ToSlash(filepath.Join(g.modulePath, g.routesDir))
		}
		pkgImports[importPath] = struct{}{}
		alias := packageAliasFromImport(importPath)
		fn := g.pathToFuncName(r.Path)
		mw, layouts := g.collectWrappersFor(r.FilePath)
		wrappers = append(wrappers, wrapperSpec{
			FuncName:        fn,
			ImportAlias:     alias,
			ImportPath:      importPath,
			HandlerIdent:    r.ComponentName,
			IsAPI:           r.IsAPI,
			Path:            r.Path,
			MiddlewareExprs: mw,
			LayoutExprs:     layouts,
		})
	}

	type importSpec struct{ Alias, Path string }
	var importList []importSpec
	var importPaths []string
	for p := range pkgImports {
		importPaths = append(importPaths, p)
	}
	sort.Strings(importPaths)
	for _, p := range importPaths {
		alias := packageAliasFromImport(p)
		if strings.HasSuffix(p, "/pkg/server") {
			alias = "server"
		}
		if strings.HasSuffix(p, "/pkg/vango/vdom") {
			alias = "vdom"
		}
		if strings.HasSuffix(p, "/pkg/renderer/html") {
			alias = "htmlrender"
		}
		if p == "net/http" || p == "strings" {
			alias = ""
		}
		importList = append(importList, importSpec{Alias: alias, Path: p})
	}

	sort.Slice(wrappers, func(i, j int) bool { return wrappers[i].FuncName < wrappers[j].FuncName })

	tmpl := `// Code generated by vango; DO NOT EDIT.
package router

import (
{{ range .Imports }}
    {{- if .Alias }}{{ .Alias }} {{ end }}"{{ .Path }}"
{{ end }}
)

type Handler func(ctx server.Ctx) (*vdom.VNode, error)

type edgeKind uint8
const (
    edgeStatic edgeKind = iota
    edgeParam
    edgeCatchAll
)

type node struct {
    label     string
    kind      edgeKind
    paramName string
    paramType string
    handler   Handler
    mws       []server.Middleware
    statics   []*node
    param     *node
    catchAll  *node
}

var root = &node{}
var notFound Handler
var internalError Handler

func init() {
{{- range .Wrappers }}
    registerRoute_{{ .FuncName }}()
{{- end }}
    {{- if .NotFoundAlias }}
    notFound = func(ctx server.Ctx) (*vdom.VNode, error) { return {{ .NotFoundAlias }}.Page(ctx) }
    {{- end }}
    {{- if .ErrorAlias }}
    internalError = func(ctx server.Ctx) (*vdom.VNode, error) { return {{ .ErrorAlias }}.Page(ctx) }
    {{- end }}
}

// ServeHTTP adapts the generated router to net/http
func ServeHTTP(w http.ResponseWriter, req *http.Request) {
    h, params, mws, ok := Match(req.URL.Path)
    if !ok || h == nil {
        if notFound != nil {
            ctx := server.NewContext(w, req)
            vnode, err := notFound(ctx)
            if err == nil && vnode != nil {
                html, rerr := htmlrender.RenderToString(vnode)
                if rerr == nil {
                    w.WriteHeader(http.StatusNotFound)
                    w.Header().Set("Content-Type", "text/html; charset=utf-8")
                    _, _ = w.Write([]byte(html))
                    return
                }
            }
        }
        w.WriteHeader(http.StatusNotFound)
        _, _ = w.Write([]byte("Not Found"))
        return
    }
    ctx := server.NewContext(w, req)
    ctx = server.WithParams(ctx, params)
    final := h
    // apply middleware outer-to-inner
    for i := len(mws) - 1; i >= 0; i-- {
        mw := mws[i]
        next := final
        final = func(c server.Ctx) (*vdom.VNode, error) {
            if err := mw.Before(c); err != nil {
                if err == server.Stop() { return nil, nil }
                return nil, err
            }
            vnode, err := next(c)
            if afterErr := mw.After(c); afterErr != nil { _ = afterErr }
            return vnode, err
        }
    }
    vnode, err := final(ctx)
    if err != nil {
        if internalError != nil {
            ivnode, ierr := internalError(ctx)
            if ierr == nil && ivnode != nil {
                html, rerr := htmlrender.RenderToString(ivnode)
                if rerr == nil {
                    w.WriteHeader(http.StatusInternalServerError)
                    w.Header().Set("Content-Type", "text/html; charset=utf-8")
                    _, _ = w.Write([]byte(html))
                    return
                }
            }
        }
        w.WriteHeader(http.StatusInternalServerError)
        _, _ = w.Write([]byte("Internal Server Error"))
        return
    }
    if vnode == nil { return }
    html, err := htmlrender.RenderToString(vnode)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        _, _ = w.Write([]byte("Render Error"))
        return
    }
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _, _ = w.Write([]byte(html))
}

func Match(path string) (Handler, map[string]string, []server.Middleware, bool) {
    if path == "" || path[0] != '/' { path = "/" + path }
    i := 1
    n := root
    params := map[string]string{}
    for {
        if n == nil { return nil, nil, nil, false }
        matched := false
        for _, s := range n.statics {
            if i-1+len(s.label) <= len(path) && path[i-1:i-1+len(s.label)] == s.label {
                i = i - 1 + len(s.label)
                if i < len(path) && path[i] == '/' { i++ }
                n = s
                matched = true
                break
            }
        }
        if matched { continue }
        if n.param != nil {
            start := i
            for i < len(path) && path[i] != '/' { i++ }
            seg := path[start:i]
            if seg == "" { return nil, nil, nil, false }
            if !validateParam(seg, n.param.paramType) { return nil, nil, nil, false }
            params[n.param.paramName] = seg
            if i < len(path) && path[i] == '/' { i++ }
            n = n.param
            continue
        }
        if n.catchAll != nil {
            if i <= len(path) { params[n.catchAll.paramName] = path[i:] }
            n = n.catchAll
            break
        }
        break
    }
    if n != nil && n.handler != nil { return n.handler, params, n.mws, true }
    return nil, nil, nil, false
}

func validateParam(v, t string) bool {
    switch t {
    case "int", "int64":
        if len(v) == 0 { return false }
        for i := 0; i < len(v); i++ { if v[i] < '0' || v[i] > '9' { return false } }
        return true
    case "uuid":
        if len(v) != 36 { return false }
        return v[8]=='-' && v[13]=='-' && v[18]=='-' && v[23]=='-'
    default:
        return len(v) > 0
    }
}

{{- range .Wrappers }}
func registerRoute_{{ .FuncName }}() {
    var h Handler
    {{- if .IsAPI }}
    h = func(ctx server.Ctx) (*vdom.VNode, error) {
        res, err := {{ .ImportAlias }}.{{ .HandlerIdent }}(ctx)
        if err != nil { return nil, err }
        if err := ctx.JSON(200, res); err != nil { return nil, err }
        return nil, nil
    }
    {{- else }}
    h = func(ctx server.Ctx) (*vdom.VNode, error) {
        vnode, err := {{ .ImportAlias }}.{{ .HandlerIdent }}(ctx)
        if err != nil { return nil, err }
        {{- range .LayoutExprs }}
        vnode = {{ . }}(vnode)
        {{- end }}
        return vnode, nil
    }
    {{- end }}
    var m []server.Middleware
    {{- range .MiddlewareExprs }}
    m = append(m, ({{ . }}))
    {{- end }}
    insertCompiledRoute(root, "{{ .Path }}", h, m)
}
{{- end }}

func insertCompiledRoute(root *node, path string, h Handler, m []server.Middleware) {
    cur := root
    i := 1
    for i <= len(path) {
        if path[i-1] == '[' {
            j := i - 1
            for j < len(path) && path[j] != ']' { j++ }
            raw := path[i-1+1 : j]
            if strings.HasPrefix(raw, "...") {
                name := raw[3:]
                cur.catchAll = &node{kind: edgeCatchAll, paramName: name}
                cur = cur.catchAll
                i = j + 2
                break
            }
            name := raw
            ptype := "string"
            if k := strings.Index(raw, ":"); k != -1 { name = raw[:k]; ptype = raw[k+1:] }
            if cur.param == nil { cur.param = &node{kind: edgeParam, paramName: name, paramType: ptype} }
            cur = cur.param
            i = j + 2
            continue
        }
        start := i - 1
        for i <= len(path) && path[i-1] != '[' { i++ }
        label := strings.TrimSuffix(path[start:i-1], "/")
        if label != "" {
            var next *node
            for _, s := range cur.statics { if s.label == label { next = s; break } }
            if next == nil { next = &node{kind: edgeStatic, label: label}; cur.statics = append(cur.statics, next) }
            cur = next
        }
        if i > len(path) { break }
    }
    cur.handler = h
    cur.mws = m
}
`

	var buf bytes.Buffer
	t := template.Must(template.New("radix").Parse(tmpl))
	if err := t.Execute(&buf, map[string]any{
		"Imports":       importList,
		"Wrappers":      wrappers,
		"NotFoundAlias": notFoundAlias,
		"ErrorAlias":    errorAlias,
	}); err != nil {
		return err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}
	if err := os.MkdirAll(filepath.Join("pkg", "internal", "router"), 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join("pkg", "internal", "router", "tree_gen.go"), formatted, 0644)
}
