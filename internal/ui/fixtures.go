package ui

import (
	"fmt"
	"net/http"
	"strings"
)

func (w *WebUI) handleFixturePage(rw http.ResponseWriter, r *http.Request) {
	pageName := strings.TrimPrefix(r.URL.Path, "/fixture/")
	if pageName == "" || pageName == "/" {
		pageName = "start"
	}

	baseURL := requestBaseURL(r)

	pages := map[string]string{
		"start": fixtureDocument(
			"Python Crawl Start",
			`<p>Python explorers use this page to test the crawler. This python page links to python practice notes, program patterns, and page ranking signals.</p>
			<p>Every page in this mini site is local, deterministic, and safe to crawl during the assignment review.</p>`,
			[]string{
				baseURL + "/fixture/python-basics",
				baseURL + "/fixture/program-patterns",
				baseURL + "/fixture/page-signals",
			},
		),
		"python-basics": fixtureDocument(
			"Python Basics Page",
			`<p>Python learners start here. Python syntax, python variables, python functions, and python loops appear together on this page so the crawler can store repeated python terms.</p>
			<p>This page also references practical program structure and page scoring.</p>`,
			[]string{
				baseURL + "/fixture/page-signals",
				baseURL + "/fixture/pipeline-notes",
			},
		),
		"program-patterns": fixtureDocument(
			"Program Patterns For Python",
			`<p>Program design patterns help python services stay predictable. This page mentions python pipelines, python indexing, and program planning for production style crawler work.</p>
			<p>Practice pages like this keep the sample crawl data easy to inspect.</p>`,
			[]string{
				baseURL + "/fixture/pipeline-notes",
			},
		),
		"page-signals": fixtureDocument(
			"Page Signals For Python Search",
			`<p>Page ranking signals are intentionally simple here. Python appears repeatedly because python frequency should clearly influence the relevance score, while page depth adds a small penalty.</p>
			<p>This page talks about python recall, python precision, and python search explainability.</p>`,
			[]string{
				baseURL + "/fixture/pipeline-notes",
			},
		),
		"pipeline-notes": fixtureDocument(
			"Pipeline Notes",
			`<p>Pipeline notes describe how a crawler can parse pages, persist terms, and publish search results while indexing is still active.</p>
			<p>Python is mentioned again here so deeper pages still contribute useful quiz data.</p>`,
			nil,
		),
	}

	document, ok := pages[pageName]
	if !ok {
		http.NotFound(rw, r)
		return
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(rw, document)
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func fixtureDocument(title, body string, links []string) string {
	var navLinks strings.Builder
	for _, link := range links {
		navLinks.WriteString(fmt.Sprintf(`<li><a href="%s">%s</a></li>`, link, link))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
</head>
<body>
    <main>
        <h1>%s</h1>
        %s
        <h2>Links</h2>
        <ul>%s</ul>
    </main>
</body>
</html>`, title, title, body, navLinks.String())
}
