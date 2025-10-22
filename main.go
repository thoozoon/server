package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	SiteDir        string
	Port           string
	NavFiles       []string // top-level files to put in navigation bar
	Password       string   // site password
	UploadsAllowed bool
	UploadsKey     string
	UploadsDir     string
	templates      *template.Template // parsed from template/*.html
	navItems       []NavItem          // computed from NavFiles
}

var config *Config

// datatypes for template rendering

type Page struct {
	Title   string
	Content template.HTML
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

func main() {
	cfg := &Config{
		SiteDir:        "./site",
		Port:           "8080",
		NavFiles:       []string{"outline.md", "assignments.md", "quizzes.md", "lectures.md", "reference.md"},
		Password:       "ahsahbeequen",
		UploadsAllowed: true,
		UploadsKey:     "fioD0aiZeer6ahzoovohs7Asheishepaefei8ue1taecuzohph8eeyaiphiel3oh",
		UploadsDir:     "uploads",
	}
	// TODO: fix this hack
	if os.Getenv("USER") == "comp3007" {
		// on SCS site
		cfg.SiteDir = "../site"
	}
	run(cfg)
}

// Assumes cfg exported fields are set
func run(cfg *Config) {

	var err error
	cfgCopy := *cfg
	config = &cfgCopy

	// Environment config
	applyEnv(config)

	checkSiteFiles(cfg)
	config.navItems = mkNavItems(cfg.NavFiles)

	config.templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal("Error parsing templates: ", err)
	}

	http.HandleFunc("/", requireAuth(handleAll))
	http.HandleFunc("/login", handleLogin)
	if config.UploadsAllowed {
		url := filepath.Join(uploadRequestURL(), "{filename}")
		fmt.Print("Adding handler for: ", url)
		http.HandleFunc(url, handleUpload)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = config.Port
	}
	// Run server
	fmt.Println("Server starting on port " + port)
	log.Print(http.ListenAndServe(":"+port, nil))
}

func uploadRequestURL() string {
	return filepath.Join("/upload", config.UploadsKey)
}

func checkSiteFiles(cfg *Config) {
	if !dirExists(cfg.SiteDir) {
		log.Fatal("site directory does not exist: ", cfg.SiteDir)
	}
	for _, file := range cfg.NavFiles {
		if filepath.Base(file) != file {
			log.Fatalf("Navigation target must be top-level: %s", file)
		}
		path := absPath(file)
		if dirExists(path) {
			log.Fatalf("Navigation target cannot be a directory: %s", path)
		}
		if !fileExists(path) {
			log.Fatalf("Navigation target doesn't exist: %s", path)
		}
	}
}

func mkNavItems(files []string) []NavItem {
	var navItems = []NavItem{{}}
	for _, file := range files {
		navItem := NavItem{
			Name: displayNameOfPath(file),
			URL:  filepath.Join("/", file),
		}
		navItems = append(navItems, navItem)
	}
	return navItems
}

func displayNameOfPath(path string) string {
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

func applyEnv(cfg *Config) {
	// siteDir := os.Getenv("SITE_DIR")
	// if siteDir != "" {
	// 	cfg.SiteDir = siteDir
	// }
	port := os.Getenv("PORT")
	if port != "" {
		cfg.Port = port
	}
}

func notFound(w http.ResponseWriter, msg string) {
	http.Error(w, "404 page not found: "+msg, http.StatusNotFound)
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
	if !isAccessible(path) {
		notFound(w, "files/directories starting with '_' are not accessible")
		return
	}
	if path == "/" {
		serveRegularFile(w, r, "/index.md")
		return
	}
	if dirExists(absPath(path)) {
		notFound(w, "Directory listing is not supported: "+path)
		return
	}
	serveRegularFile(w, r, path)
}

func isAccessible(path string) bool {
	for _, name := range SplitPath(path) {
		if (name != "") && name[0] == '_' {
			return false
		}
	}
	return true
}

func absPath(relPath string) string {
	path := string(http.Dir(config.SiteDir))
	return filepath.Join(path, relPath)
}

func serveRegularFile(w http.ResponseWriter, r *http.Request, path string) {
	if dirExists(absPath(path)) {
		notFound(w, "Request to list a directory: "+path)
		return
	}
	if filepath.Ext(path) == ".md" {
		serveMarkdownFile(w, r, path)
		return
	}
	if filepath.Ext(path) == ".html" {
		serveHTMLFile(w, r, path)
		return
	}

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
		Title:   displayNameOfPath(path),
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

func debugPrint(args ...any) {
	log.Print(args...)
}

func servePage(w http.ResponseWriter, page *Page) {
	if err := config.templates.ExecuteTemplate(w, "base.html", page); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Here!")
	if err := os.MkdirAll(config.UploadsDir, 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}
	// Only accept PUT requests
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", "PUT")
		http.Error(w, "Method not allowed. Use PUT to upload files.", http.StatusMethodNotAllowed)
		return
	}
	// Extract filename from URL path
	path := strings.TrimPrefix(r.PathValue("filename"), filepath.Join(uploadRequestURL(), "/"))
	if path == "" {
		http.Error(w, "Filename is required in URL path", http.StatusBadRequest)
		return
	}

	// Clean the filename to prevent directory traversal attacks
	filename := filepath.Base(path)
	if filename == "." || filename == ".." {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Create the full file path
	filePath := filepath.Join(config.UploadsDir, filename)

	log.Printf("Receiving file upload: %s -> %s", path, filePath)

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file %s: %v", filePath, err)
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Copy the request body to the file
	bytesWritten, err := io.Copy(file, r.Body)
	if err != nil {
		log.Printf("Error writing file %s: %v", filePath, err)
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully uploaded file: %s (%d bytes)", filename, bytesWritten)

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
	"status": "success",
	"message": "File uploaded successfully",
	"filename": "%s",
	"bytes_written": %d,
	"path": "%s"
}`, filename, bytesWritten, filePath)
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

// SplitPath takes a path string and returns a slice of strings containing
// all directory and file names that make up the path
// For example, "/foo/bar/baz" returns {"foo", "bar", "baz"}
func SplitPath(path string) []string {
	// Clean the path to remove any redundant separators
	path = filepath.Clean(path)

	// Remove leading separator if present
	if strings.HasPrefix(path, string(filepath.Separator)) {
		path = path[1:]
	}

	// Handle empty path or root
	if path == "" || path == "." {
		return []string{}
	}

	// Split by filepath separator
	return strings.Split(path, string(filepath.Separator))
}
