package main

import (
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// P = local fs document root = config.SiteDir
// u = request URL
// cs = names of collections = config.Collections
// navs = navigation links = config.NavFiles
//
// P restrictions
// - P/index.md exists
// - For f ∈ cs, P/f is a directory
// - For f ∈ navs-cs, f ends in ".md" and is not a directory
//
// To serve a file P/path/f:
// - If f ∈ cs, extend f/index.md with a listing of f, render and return it
// - If f is a directory and f is not in cs, "not found"
// - If f ends in ".md", render and return it
// - Otherwise, return f for the browser to display
type Config struct {
	SiteDir     string
	Port        int
	NavFiles    []string           // top-level files to put in navigation bar
	Collections []string           // top-level directories with auto-indexed markdown files
	Password    string             // site password
	templates   *template.Template // parsed from template/*.html
	navItems    []NavItem          // computed from NavFiles
}

var config *Config

// datatypes for template rendering

type Page struct {
	Title   string
	Content template.HTML
	Nav     []NavItem
}

type IndexPage struct {
	Title   string
	Content template.HTML
	Files   []FileInfo
	Nav     []NavItem
}

type NavItem struct {
	Name string
	URL  string
}

type FileInfo struct {
	Name        string
	Path        string
	DisplayName string
}

type LoginPage struct {
	Error string
}

// Assumes cfg exported fields are set
func run(cfg *Config) {

	var err error
	cfgCopy := *cfg
	config = &cfgCopy

	checkSiteFiles(cfg)
	config.navItems = mkNavItems(cfg.NavFiles)

	config.templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal("Error parsing templates: ", err)
	}

	http.HandleFunc("/", requireAuth(handleAll))
	http.HandleFunc("/login", handleLogin)

	// Run server
	server := "localhost:" + strconv.Itoa(config.Port)
	fmt.Println("Server starting at	:" + server)
	http.ListenAndServe(server, nil)
}

func checkSiteFiles(cfg *Config) {
	if !dirExists(cfg.SiteDir) {
		log.Fatal("site directory does not exist: ", cfg.SiteDir)
	}
	for _, file := range cfg.Collections {
		if filepath.Base(file) != file {
			log.Fatalf("Collection directory must be a top-level directory name: %s", file)
		}
		if !dirExists(absPath(file)) {
			log.Fatalf("Collection directory doesn't exist: %s", file)
		}
		// if !fileExists(filepath.Join(absPath, "index.md")) {
		// 	log.Fatalf("Collection directory must contain an index.md")
		// }
	}
	for _, file := range cfg.NavFiles {
		if filepath.Base(file) != file {
			log.Fatalf("Non-collection navigation target must be a regular filename: %s", file)
		}
		if slices.Contains(cfg.Collections, file) {
			continue
		}
		file = filepath.Clean(file)
		path := absPath(file)
		if dirExists(path) {
			log.Fatalf("A NavFile that is a directory must also be a collection: %s", path)
		}
		if !fileExists(path) {
			log.Fatalf("Navigation target doesn't exist: %s", path)
		}
	}
}

func mkNavItems(files []string) []NavItem {
	// var navItems = []NavItem{{Name: "Home", URL: "/"}}
	var navItems = []NavItem{{}}
	for _, file := range files {
		navItem := NavItem{
			Name: pathToDisplayName(file),
			URL:  filepath.Join("/", file),
		}
		navItems = append(navItems, navItem)
	}
	return navItems
}

func pathToDisplayName(path string) string {
	dir, file := filepath.Split(path)
	if file == "index.md" && (dir == "" || dir == "/") {
		return "Home"
	}
	rawName := file
	if file == "index.md" {
		rawName = filepath.Base(dir)
	}
	trimmed := strings.TrimSuffix(rawName, filepath.Ext(path))
	return Capitalize(strings.ReplaceAll(trimmed, "-", " "))
}

func main() {
	cfg := &Config{
		SiteDir:     "/Users/howe/Documents/teaching/3007/website",
		Port:        8080,
		NavFiles:    []string{"outline.md", "assignments", "quizzes", "lectures", "reference"},
		Collections: []string{"assignments", "quizzes", "lectures", "reference"},
		Password:    "ahsahbeequen",
	}
	run(cfg)
}

func notFound(w http.ResponseWriter, msg string) {
	http.NotFound(w, nil)
	log.Print("Not found" + msg)
}

// Authentication middleware
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if user has valid auth cookie
		cookie, err := r.Cookie("auth")
		if err == nil && cookie.Value == "authenticated" {
			next(w, r)
			return
		}

		// Redirect to login
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// Handle login page and authentication
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Show login form
		loginPage := LoginPage{}
		if err := config.templates.ExecuteTemplate(w, "login.html", loginPage); err != nil {
			log.Printf("Error executing login template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == "POST" {
		// Process login
		password := r.FormValue("password")

		if password == config.Password {
			// Set auth cookie
			cookie := &http.Cookie{
				Name:     "auth",
				Value:    "authenticated",
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // Set to true if using HTTPS
				SameSite: http.SameSiteLaxMode,
				MaxAge:   86400 * 30, // 30 days
			}
			http.SetCookie(w, cookie)

			// Redirect to home
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		} else {
			// Show login form with error
			loginPage := LoginPage{Error: "Invalid password"}
			if err := config.templates.ExecuteTemplate(w, "login.html", loginPage); err != nil {
				log.Printf("Error executing login template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleAll(w http.ResponseWriter, r *http.Request) {
	if !(r.Method == "" || r.Method == "GET") {
		notFound(w, "Only the GET method is allowed.")
		return
	}
	path := r.URL.Path
	path = filepath.Clean(path)
	base := filepath.Base(path)
	dir := filepath.Dir(path)
	if path == "/" {
		serveRegularFile(w, r, "/index.md")
		return
	}
	if dir == "/" && slices.Contains(config.Collections, base) {
		serveCollection(w, r, base)
		return
	}
	if dirExists(absPath(path)) {
		notFound(w, "Directory listing is not supported except for Collections:: "+path)
		return
	}
	serveRegularFile(w, r, path)
}

func absPath(relPath string) string {
	path := string(http.Dir(config.SiteDir))
	return filepath.Join(path, relPath)
}

func serveRegularFile(w http.ResponseWriter, r *http.Request, path string) {
	if dirExists(absPath(path)) {
		notFound(w, "Request to serve a non-collection directory as a regular file: "+path)
		return
	}
	if strings.HasSuffix(path, ".md") {
		serveMarkdownFile(w, r, path)
		return
	}
	if strings.HasSuffix(path, ".html") {
		serveHTMLFile(w, r, path)
	}
	_ = mime.AddExtensionType(".pdf", "application/pdf")
	_ = mime.AddExtensionType(".hs", "text/plain")
	_ = mime.AddExtensionType(".go", "text/plain")
	_ = mime.AddExtensionType(".txt", "text/plain")
	http.ServeFile(w, r, absPath(path))
}

func serveMarkdownFile(w http.ResponseWriter, _ *http.Request, path string) {
	content, err := os.ReadFile(absPath(path))
	if err != nil {
		notFound(w, "Could not read file: "+path)
		return
	}

	rendered := renderMarkdown(content)
	// htmlContent := template.HTML(rendered)
	servePage(w, mkPage(rendered, path))
}

func serveHTMLFile(w http.ResponseWriter, _ *http.Request, path string) {
	htmlContent, err := os.ReadFile(absPath(path))
	if err != nil {
		notFound(w, "Could not read file: "+path)
		return
	}
	servePage(w, mkPage(htmlContent, path))
}

func mkPage(htmlContent []byte, path string) *Page {
	page := Page{
		Title:   pathToDisplayName(path),
		Content: template.HTML(htmlContent),
		Nav:     config.navItems,
	}
	return &page
}

func renderMarkdown(content []byte) []byte {

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(content)

	// Configure HTML renderer with security and usability flags
	htmlFlags := html.CommonFlags | html.HrefTargetBlank // Open external links in new tab
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

func serveCollection(w http.ResponseWriter, _ *http.Request, name string) {

	indexPath := filepath.Join(config.SiteDir, name, "index.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		content = []byte{}
	}
	htmlContent := renderMarkdown(content)

	// Get files in the folder (excluding index.md)
	files := getFilesExcluding(name, "index.md")

	indexPage := IndexPage{
		Title:   pathToDisplayName(indexPath),
		Content: template.HTML(htmlContent),
		Files:   files,
		Nav:     config.navItems,
	}

	if err := config.templates.ExecuteTemplate(w, "index-with-listing.html", indexPage); err != nil {
		log.Printf("Error executing index template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func debugPrint(args ...any) {
	log.Print(args...)
}

func getFilesExcluding(folder string, excludeFile string) []FileInfo {
	var files []FileInfo
	entries, err := os.ReadDir(absPath(folder))
	if err != nil {
		log.Println("Could not list directory" + folder)
	}
	for _, e := range entries {
		if !e.IsDir() && e.Name() != excludeFile {
			file := FileInfo{
				Name:        e.Name(),
				Path:        filepath.Join("/", folder, e.Name()),
				DisplayName: pathToDisplayName(e.Name()),
			}
			files = append(files, file)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files
}

func servePage(w http.ResponseWriter, page *Page) {
	if err := config.templates.ExecuteTemplate(w, "base.html", page); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ----------------------
// General utilities
// ----------------------

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func IsAlphanumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// identity on strings not starting with a letter
func Capitalize(word string) string {
	if word == "" {
		return word
	}
	runes := []rune(word)
	if unicode.IsLetter(runes[0]) {
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes)
	}
	return word
}
