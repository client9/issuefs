package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/client9/doublebrace"
	"github.com/nickg/issuefs/internal/embedded"
)

type listTemplateData struct {
	Entries []listItem
	Count   int
	Now     time.Time
}

func renderList(stdout io.Writer, data listTemplateData, templatePath string) error {
	src, name, markdown, err := loadListTemplate(templatePath)
	if err != nil {
		return err
	}

	tmpl, err := template.New(name).Funcs(listTemplateFuncs()).Parse(string(src))
	if err != nil {
		return fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template %s: %w", name, err)
	}

	out := buf.String()
	if strings.TrimSpace(out) == "" {
		return nil
	}

	if markdown {
		style := pickStyle("auto", stdout)
		out, err = renderMarkdown(out, style)
		if err != nil {
			return err
		}
	}

	_, err = io.WriteString(stdout, out)
	return err
}

func loadListTemplate(templatePath string) ([]byte, string, bool, error) {
	if templatePath == "" {
		src, err := embedded.Templates.ReadFile("templates/list.md")
		if err != nil {
			return nil, "", false, err
		}
		return src, "builtin:list.md", true, nil
	}

	src, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, "", false, err
	}
	return src, filepath.Base(templatePath), isMarkdownTemplate(templatePath), nil
}

func listTemplateFuncs() template.FuncMap {
	return doublebrace.Merge(doublebrace.FuncMap(), template.FuncMap{
		"escapeCell": escapeMarkdownCell,
		"mdCell":     escapeMarkdownCell,
	})
}

func isMarkdownTemplate(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdown", ".mkdn", ".mkd":
		return true
	default:
		return false
	}
}

func escapeMarkdownCell(s string) string {
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		`|`, `\|`,
		"\r\n", "<br>",
		"\n", "<br>",
	)
	return replacer.Replace(s)
}
