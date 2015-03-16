package main

import (
	"fmt"
	"html/template"
	"path/filepath"
	//"io"
	"net/http"
)

type requestMapper map[string]func(http.ResponseWriter, *http.Request)

var request_mux requestMapper

type myHandler struct{}

type MetaTag struct {
	Content     string
	Description string
}

type Page struct {
	ExtraCSS     []string
	ExtraMeta    []MetaTag
	ExtraScripts []string
	Header       string
	NoContent    bool
	NoHomeLink   bool
	Title        string
}

func basicHandle(w http.ResponseWriter, r *http.Request) {
	p := &Page{ExtraCSS: []string{"/css/generic/basic-page.css",
		"/css/generic/header.css",
		"/css/page/index.css"},
		ExtraMeta:    []MetaTag{},
		ExtraScripts: []string{},
		Header:       "jww (at) joelweinberger (dot) us",
		NoContent:    false,
		NoHomeLink:   true,
		Title:        "Joel H. W. Weinberger -- jww"}
	t, _ := template.ParseFiles("layout.html", "body.html")
	err := t.Execute(w, p)

	if err != nil {
		fmt.Println(err)
		return
	}
}

func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handle, ok := request_mux[r.URL.String()]; !ok {
		path := r.URL.Path
		// This should never be the case, so this should probably be an assert,
		// but just in case something wacky occurs, return a 404 if the URL is
		// not an absolute path.
		if !filepath.IsAbs(path) {
			http.NotFound(w, r)
			return
		}
		// The following cleanup is necessary to avoid directory traversals.
		// Since the above check makes sure that the path is absolute, this call
		// to Clean removes any ../ references so a directory traversal is not
		// possible. That is, this call treats the Path as if it is at root, and
		// removes anything that would go beyond root.
		path = filepath.Clean(path)
		fmt.Println("Serving static file ./public" + path)
		http.ServeFile(w, r, "./public"+path)
	} else {
		handle(w, r)
	}
}

func main() {
	http_port := "8000"
	request_mux = requestMapper{
		"/":      basicHandle,
		"/index": basicHandle}

	server := http.Server{
		Addr:    ":" + http_port,
		Handler: &myHandler{},
	}

	fmt.Println("Listening on port " + http_port)
	server.ListenAndServe()
}