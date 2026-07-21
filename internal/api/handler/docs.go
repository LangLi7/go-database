package handler

import (
	"bytes"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md goldmark.Markdown
var docsLayout *template.Template
var docsFiles []DocFile

type DocFile struct {
	Name string
	Path string
	Slug string
}

const docsDir = "docs"

// InitDocs loads markdown files and pre-compiles templates.
func InitDocs() {
	md = goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	docsLayout = template.Must(template.New("docs").Parse(docLayoutHTML))

	entries, err := os.ReadDir(docsDir)
	if err != nil {
		slog.Warn("docs directory not found", "path", docsDir)
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		name := strings.ReplaceAll(slug, "_", " ")
		name = strings.ReplaceAll(name, "-", " ")
		name = strings.Title(strings.ToLower(name))
		docsFiles = append(docsFiles, DocFile{
			Name: name,
			Path: e.Name(),
			Slug: slug,
		})
	}
	sort.Slice(docsFiles, func(i, j int) bool { return docsFiles[i].Name < docsFiles[j].Name })
	slog.Info("docs loaded", "count", len(docsFiles))
}

// HandleDocs renders a markdown doc as HTML.
func HandleDocs(c *gin.Context) {
	slug := strings.TrimPrefix(c.Param("slug"), "/")
	if slug == "" {
		slug = "README"
	}

	var file *DocFile
	for _, d := range docsFiles {
		if d.Slug == slug {
			file = &d
			break
		}
	}
	if file == nil {
		c.String(404, "Document not found")
		return
	}

	content, err := os.ReadFile(filepath.Join(docsDir, file.Path))
	if err != nil {
		c.String(500, "Error reading document")
		return
	}

	var buf bytes.Buffer
	if err := md.Convert(content, &buf); err != nil {
		c.String(500, "Error rendering markdown")
		return
	}

	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = docsLayout.Execute(c.Writer, map[string]any{
		"Title":   file.Name,
		"Content": template.HTML(buf.String()),
		"Files":   docsFiles,
		"Active":  file.Slug,
	})
}

// HandleDocsRedirect redirects /docs to /docs/README.
func HandleDocsRedirect(c *gin.Context) {
	c.Redirect(http.StatusFound, "/docs/README")
}

const docLayoutHTML = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}} — go-database</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">
<style>
body { display: flex; min-height: 100vh; margin: 0; }
nav.docs-nav { width: 260px; padding: 1rem; background: var(--pico-background-color); border-right: 1px solid var(--pico-muted-border-color); overflow-y: auto; }
nav.docs-nav h3 { margin-bottom: 0.5rem; font-size: 1rem; }
nav.docs-nav a { display: block; padding: 0.25rem 0.5rem; border-radius: 4px; color: var(--pico-color); text-decoration: none; font-size: 0.85rem; }
nav.docs-nav a.active { background: var(--pico-primary-background); color: var(--pico-primary-inverse); }
nav.docs-nav a:hover:not(.active) { background: var(--pico-muted-border-color); }
main.docs-content { flex: 1; padding: 2rem; max-width: 900px; overflow-y: auto; }
main.docs-content h1:first-child { margin-top: 0; }
main.docs-content pre { border-radius: 6px; }
main.docs-content table { display: block; overflow-x: auto; }
header.docs-header { border-bottom: 1px solid var(--pico-muted-border-color); padding-bottom: 0.5rem; margin-bottom: 1.5rem; }
header.docs-header a { text-decoration: none; color: var(--pico-primary); font-weight: 600; }
footer.docs-footer { margin-top: 2rem; padding-top: 1rem; border-top: 1px solid var(--pico-muted-border-color); font-size: 0.8rem; color: var(--pico-muted-color); }
@media (max-width: 768px) { nav.docs-nav { display: none; } main.docs-content { padding: 1rem; } }
</style>
</head>
<body>
<nav class="docs-nav">
  <h3><a href="/docs/README" style="font-size:1rem;padding:0;color:var(--pico-primary);">📖 go-database</a></h3>
  <hr>
  {{range .Files}}
  <a href="/docs/{{.Slug}}"{{if eq .Slug $.Active}} class="active"{{end}}>{{.Name}}</a>
  {{end}}
  <hr>
  <a href="/" style="color:var(--pico-muted-color);">← API</a>
</nav>
<main class="docs-content">
  <header class="docs-header"><a href="/docs/README">go-database Docs</a></header>
  {{.Content}}
  <footer class="docs-footer">go-database — {{.Title}}</footer>
</main>
</body>
</html>`
