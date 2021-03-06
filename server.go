package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/russross/blackfriday"
)

var templates_dir string
var static_dir string
var cert_config_file string
var pubs_file string

var csp string = strings.Join([]string{
	"default-src 'self'",
	"child-src 'self' *.google.com",
	"frame-src 'self' *.google.com",
	"style-src 'unsafe-inline' 'self'",
}, "; ")

var server_scheme string = "https"

type requestMapper map[string]func(http.ResponseWriter, *http.Request)

var request_mux requestMapper

type myHandler struct{}

type MetaTag struct {
	Content     string
	Description string
}

type BasicPage struct {
	ExtraCSS     []string
	ExtraMeta    []MetaTag
	ExtraScripts []string
	Header       string
	NoContent    bool
	NoHomeLink   bool
	Pubs         *PubsInfo
	Title        string
}

type Author struct {
	Homepage string
	Name     string
}

type Paper struct {
	Abstract     string
	Authors      []string
	Booktitle    string
	Citeseer     string
	Conference   string
	Extended     string
	Homepage     string
	Institution  string
	Nobibtex     bool
	Notes        string
	Number       string
	Pdf          string
	Presentation string
	Proceedings  string
	Textitle     string
	Title        string
	Url          string
	Year         string
}

type AbstractPage struct {
	BasicPage
	Abstract string
	Title    string
	NoLayout bool
}

type BibtexPage struct {
	BasicPage
	AuthorList string
	NoLayout   bool
	Paper      Paper
}

var pubs *PubsInfo

var pages map[string]*BasicPage = map[string]*BasicPage{
	"abstract.html": &BasicPage{
		ExtraCSS: []string{
			"/css/generic/basic-page.css",
			"/css/generic/header.css",
		},
		ExtraMeta:    []MetaTag{},
		ExtraScripts: []string{},
		Header:       "jww (at) joelweinberger (dot) us -- abstract",
		NoContent:    false,
		Title:        "Joel H. W. Weinberger -- Paper Abstract",
	},
	"bibtex.html": &BasicPage{
		ExtraCSS: []string{
			"/css/generic/basic-page.css",
			"/css/generic/header.css",
		},
		ExtraMeta:    []MetaTag{},
		ExtraScripts: []string{},
		Header:       "jww (at) joelweinberger (dot) us -- bibtex",
		NoContent:    false,
		Title:        "Joel H. W. Weinberger -- Paper BibTeX",
	},
	"calendar.html": &BasicPage{
		ExtraCSS: []string{
			"/css/page/calendar.css",
		},
		ExtraMeta:    []MetaTag{},
		ExtraScripts: []string{},
		Header:       "",
		NoContent:    true,
		NoHomeLink:   true,
		Title:        "Joel H. W. Weinberger -- Calendar",
	},
	"index.html": &BasicPage{
		ExtraCSS: []string{
			"/css/generic/basic-page.css",
			"/css/generic/header.css",
			"/css/page/index.css",
		},
		ExtraMeta:    []MetaTag{},
		ExtraScripts: []string{},
		Header:       "jww (at) joelweinberger (dot) us",
		NoContent:    false,
		NoHomeLink:   true,
		Title:        "Joel H. W. Weinberger -- jww",
	},
	"offline.html": &BasicPage{
		ExtraCSS: []string{
			"/css/generic/basic-page.css",
			"/css/generic/header.css",
			"/css/page/index.css",
		},
		ExtraMeta:    []MetaTag{},
		ExtraScripts: []string{},
		Header:       "offline",
		NoContent:    false,
		NoHomeLink:   false,
		Title:        "Joel H. W. Weinberger -- Offline",
	},
	"publications.html": &BasicPage{
		ExtraCSS: []string{
			"/css/generic/basic-page.css",
			"/css/generic/header.css",
			"/css/page/index.css",
		},
		ExtraMeta: []MetaTag{},
		ExtraScripts: []string{
			"/js/index.js",
		},
		Header:     "publications",
		NoContent:  false,
		NoHomeLink: false,
		Title:      "Joel H. W. Weinberger -- Publications",
	},
	"wedding.html": &BasicPage{
		ExtraCSS: []string{
			"/css/generic/basic-page.css",
			"/css/generic/header.css",
			"/css/page/index.css",
		},
		ExtraMeta:    []MetaTag{},
		ExtraScripts: []string{},
		Header:       "wedding",
		NoContent:    false,
		NoHomeLink:   false,
		Title:        "Joel H. W. Weinberger -- Wedding",
	},
}

var templates map[string]*template.Template
var abstractTemplate *template.Template
var bibtexTemplate *template.Template

func addHeaders(w http.ResponseWriter, useCSP bool) {
	header := w.Header()
	if useCSP {
		header.Set("Content-Security-Policy", csp)
	}
	header.Set("Strict-Transport-Security", "max-age=12096000")
}

func generateBasicHandle(page string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		addHeaders(w, true)
		err := templates[page].Execute(w, pages[page])

		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func reversePapersInPlace(papers []Paper) {
	total_len := len(papers)
	for i, _ := range papers {
		if i >= total_len/2 {
			break
		}
		tmp := papers[total_len-i-1]
		papers[total_len-i-1] = papers[i]
		papers[i] = tmp
	}
}

type PubsInfo struct {
	Authors map[string]Author
	Papers  []Paper
	Techs   []Paper
}

func loadPubsInfo() bool {
	jsonBlob, err := ioutil.ReadFile(pubs_file)

	if err != nil {
		fmt.Println("Error loading pubs.json: ", err)
		return false
	}

	err = json.Unmarshal(jsonBlob, &pubs)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return false
	}

	// For historical reasons, pubs are stored in the pubs.json file in
	// chronological order. However, for purposes of templates and what-not, we
	// actually want them to be in reverse chronological order, so we do that
	// here.
	reversePapersInPlace(pubs.Papers)
	reversePapersInPlace(pubs.Techs)

	pages["publications.html"].Pubs = pubs

	return true
}

func abstractHandle(isAjax bool, w http.ResponseWriter, r *http.Request) {
	addHeaders(w, true)
	pubError := func(url string, msg string) {
		fmt.Println("Error extracting pub number from URL \"", url, "\": ", msg)
	}

	var abstract string
	if isAjax {
		abstract = strings.TrimPrefix(r.URL.Path, "/ajax/abstracts/")
	} else {
		abstract = strings.TrimPrefix(r.URL.Path, "/abstracts/")
	}

	fmt.Println("Serving abstract ", abstract)

	paperArray := pubs.Papers
	var validPub = regexp.MustCompile(`\/abstracts\/pub([0-9]+)`)
	groups := validPub.FindStringSubmatch(r.URL.Path)
	if len(groups) < 2 {
		paperArray = pubs.Techs
		validPub = regexp.MustCompile(`\/abstracts\/tech([0-9]+)`)
		groups = validPub.FindStringSubmatch(r.URL.Path)

		if len(groups) < 2 {
			pubError(r.URL.Path, "No number present")
			http.NotFound(w, r)
			return
		}
	}

	var index int
	var err error
	if index, err = strconv.Atoi(groups[1]); err != nil {
		pubError(r.URL.Path, "Not a number")
		http.NotFound(w, r)
		return
	}

	if index >= len(paperArray) {
		pubError(r.URL.Path, "Pub doesn't exist")
		http.NotFound(w, r)
		return
	}

	abstractPage := AbstractPage{
		BasicPage: *pages["abstract.html"],
		Abstract:  paperArray[index].Abstract,
		Title:     paperArray[index].Title,
		NoLayout:  isAjax,
	}
	if isAjax {
		abstractTemplate.Execute(w, abstractPage)
	} else {
		err = templates["abstract.html"].Execute(w, abstractPage)
	}

	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
}

func bibtexHandle(isAjax bool, w http.ResponseWriter, r *http.Request) {
	addHeaders(w, true)
	pubError := func(url string, msg string) {
		fmt.Println("Error extracting pub number from URL \"", url, "\": ", msg)
	}

	var bibtex string
	if isAjax {
		bibtex = strings.TrimPrefix(r.URL.Path, "/ajax/bibtexs/")
	} else {
		bibtex = strings.TrimPrefix(r.URL.Path, "/bibtexs/")
	}

	fmt.Println("Serving bibtex ", bibtex)

	paperArray := pubs.Papers
	var validPub = regexp.MustCompile(`\/bibtexs\/pub([0-9]+)`)
	groups := validPub.FindStringSubmatch(r.URL.Path)
	if len(groups) < 2 {
		paperArray = pubs.Techs
		validPub = regexp.MustCompile(`\/bibtexs\/tech([0-9]+)`)
		groups = validPub.FindStringSubmatch(r.URL.Path)

		if len(groups) < 2 {
			pubError(r.URL.Path, "No number present")
			http.NotFound(w, r)
			return
		}
	}

	var index int
	var err error
	if index, err = strconv.Atoi(groups[1]); err != nil {
		pubError(r.URL.Path, "Not a number")
		http.NotFound(w, r)
		return
	}

	if index >= len(paperArray) {
		pubError(r.URL.Path, "Pub doesn't exist")
		http.NotFound(w, r)
		return
	}

	var authors string
	for j, author := range paperArray[index].Authors {
		if j != 0 {
			authors = authors + " and "
		}
		authors = authors + pubs.Authors[author].Name
	}

	bibtexPage := BibtexPage{
		BasicPage:  *pages["bibtex.html"],
		AuthorList: authors,
		Paper:      paperArray[index],
		NoLayout:   isAjax,
	}
	if isAjax {
		bibtexTemplate.Execute(w, bibtexPage)
	} else {
		err = templates["bibtex.html"].Execute(w, bibtexPage)
	}

	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
}

func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only serve from "www."
	if strings.Index(strings.ToLower(r.Host), "www.") != 0 {
		redirectToWww(w, r)
		return
	}

	if handle, ok := request_mux[r.URL.EscapedPath()]; ok {
		// Dynamic page
		handle(w, r)
		return
	}

	// Static Content
	// Note that, for now, there are no static files that CSP is applied to. If
	// that should change, the second argument to addHeaders() should be changed
	// appropriately.
	addHeaders(w, false)
	path := r.URL.Path

	// For legacy reasons (namely, the original blog), we need to redirect
	// links from the original blog path to the new blog path so that old
	// permalinks still work.
	if strings.Index(path+"/", "/blog/") == 0 {
		redirectToBlog(w, r)
		return
	}

	// Abstracts and bibliographies are special cases because their
	// particular pages are generated dynamically.
	if strings.Index(path, "/abstracts/") == 0 {
		abstractHandle(false, w, r)
		return
	}

	if strings.Index(path, "/ajax/abstracts/") == 0 {
		abstractHandle(true, w, r)
		return
	}

	if strings.Index(path, "/bibtexs/") == 0 {
		bibtexHandle(false, w, r)
		return
	}

	if strings.Index(path, "/ajax/bibtexs/") == 0 {
		bibtexHandle(true, w, r)
		return
	}

	// All other cases are static files that need to be loaded from the static/
	// directory.

	// The following should never be the case, so this should probably be an
	// assert, but just in case something wacky occurs, return a 404 if the URL
	// is not an absolute path.
	if !filepath.IsAbs(path) {
		http.NotFound(w, r)
		return
	}

	// The following cleanup is necessary to avoid directory traversals.  Since
	// the above check makes sure that the path is absolute, this call to
	// Clean() removes any ../ references so a directory traversal is not
	// possible. That is, this call treats the Path as if it is at root, and
	// removes anything that would go beyond root.
	path = filepath.Clean(path)
	fmt.Println("Serving static file static" + path)
	http.ServeFile(w, r, filepath.Join(static_dir, path))
}

func markdowner(args ...interface{}) template.HTML {
	s := blackfriday.MarkdownCommon([]byte(fmt.Sprintf("%s", args...)))
	return template.HTML(s)
}

func redirectToBlog(w http.ResponseWriter, r *http.Request) {
	addHeaders(w, true)
	dst := "http://blog.joelweinberger.us"
	path := strings.TrimPrefix(r.URL.Path, "/blog")
	fmt.Println("Redirecting to ", dst+path)
	http.Redirect(w, r, dst+path, http.StatusMovedPermanently)
}

func redirectToWww(w http.ResponseWriter, r *http.Request) {
	addHeaders(w, true)
	dst := server_scheme + "://www." + r.Host + r.URL.Path
	fmt.Println("Redirecting to ", dst)
	http.Redirect(w, r, dst, http.StatusMovedPermanently)
}

func generateRedirectToHttps(https_port string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		addHeaders(w, true)
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			// Assume that the error was because there is no port present.
			host = r.Host
		}
		redirect_url := *r.URL
		redirect_url.Scheme = "https"
		redirect_url.Host = host + ":" + https_port
		fmt.Println("Redirecting '", r.URL.String(), "' to '", redirect_url.String(), "'")
		http.Redirect(w, r, redirect_url.String(), http.StatusMovedPermanently)
	}
}

type CertsConfig struct {
	PrivateKey string
	FullChain  string
}

func loadCertConfig() *CertsConfig {
	jsonBlob, err := ioutil.ReadFile(cert_config_file)

	if err != nil {
		fmt.Println("Error loading certs.json: ", err)
		return nil
	}

	var config CertsConfig
	err = json.Unmarshal(jsonBlob, &config)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return nil
	}

	return &config
}

func setConfigFiles(resources_dir *string) {
	templates_dir = filepath.Join(*resources_dir, "templates")
	static_dir = filepath.Join(*resources_dir, "static")
	cert_config_file = filepath.Join(*resources_dir, "cert_config.json")
	pubs_file = filepath.Join(*resources_dir, "pubs.json")
}

func main() {
	http_port_int := flag.Int("http-port", 8080, "The HTTP port to listen on that will redirect to HTTPS.")
	https_port_int := flag.Int("https-port", 8443, "The HTTPS port to listen on for the main server.")
	unsafely_run_on_http := flag.Bool("unsafely-run-on-http", false, "Whether to unsafely run the main server on HTTP.")
	resources_dir := flag.String("resources-dir", ".", "Directory where resources reside.")

	flag.Parse()

	if *http_port_int < 1 || *http_port_int > 65535 ||
		*https_port_int < 1 || *https_port_int > 65535 {
		fmt.Println("Port numbers must be between 1 and 65535, inclusive.")
		return
	}

	http_port := strconv.Itoa(*http_port_int)
	https_port := strconv.Itoa(*https_port_int)

	setConfigFiles(resources_dir)

	request_mux = requestMapper{
		"/":             generateBasicHandle("index.html"),
		"/calendar":     generateBasicHandle("calendar.html"),
		"/index":        generateBasicHandle("index.html"),
		"/offline":      generateBasicHandle("offline.html"),
		"/publications": generateBasicHandle("publications.html"),
		"/wedding":      generateBasicHandle("wedding.html"),
	}

	if !loadPubsInfo() {
		panic("Failed to read the publications configuration.")
	}

	templates = make(map[string]*template.Template)
	funcMap := template.FuncMap{"markdown": markdowner}
	layout := template.Must(template.ParseFiles(filepath.Join(templates_dir, "/layout.html"))).Funcs(funcMap)
	for name, _ := range pages {
		templates[name] = template.Must(template.Must(layout.Clone()).ParseFiles(filepath.Join(templates_dir, name)))
	}

	abstractTemplateBytes, err := ioutil.ReadFile(filepath.Join(templates_dir, "/abstract.html"))
	if err != nil {
		panic("Could not read abstract.html template")
	}
	abstractTemplate = template.Must(template.New("abstract").Funcs(funcMap).Parse(string(abstractTemplateBytes)))

	bibtexTemplateBytes, err := ioutil.ReadFile(filepath.Join(templates_dir, "/bibtex.html"))
	if err != nil {
		panic("Could not read bibtex.html template")
	}
	bibtexTemplate = template.Must(template.New("bibtex").Funcs(funcMap).Parse(string(bibtexTemplateBytes)))

	// The HTTP server is strictly for redirecting to HTTPS, unless the
	// --unsafely-run-on-http flag is specified.
	if *unsafely_run_on_http {
		server_scheme = "http"
		fmt.Println("Unsafely running on HTTP on port " + http_port)
		http.ListenAndServe(":"+http_port, &myHandler{})
	} else {
		fmt.Println("Redirecting HTTP on port " + http_port + " to HTTPS on port " + https_port)
		server := http.Server{
			Addr:    ":" + https_port,
			Handler: &myHandler{},
		}

		cert_config := loadCertConfig()
		if cert_config == nil {
			return
		}

		go http.ListenAndServe(":"+http_port, http.HandlerFunc(generateRedirectToHttps(https_port)))

		fmt.Println("Listening on port " + https_port)
		if err := server.ListenAndServeTLS(cert_config.FullChain, cert_config.PrivateKey); err != nil {
			fmt.Println("ListenAndServe error: %v", err)
			return
		}
	}
}
