package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type ServerConfig struct {
	SiteDir         string `toml:"site_dir"`
	Port            string `toml:"port"`
	UploadsAllowed  bool   `toml:"uploads_allowed"`
	Secret          string `toml:"secret"`
	UploadsDir      string `toml:"uploads_dir"`
	DBPath          string `toml:"db_path"`
	ResendApiKey    string `toml:"resend_api_key"`
	ResendFromEmail string `toml:"resend_from_email"`
	AuthDisabled    bool   `toml:"auth_disabled"`
}

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
type SiteConfig struct {
	NavFiles []string `toml:"nav_files"`
}

type Config struct {
	ServerConfig
	SiteConfig
	templates *template.Template
	navItems  []NavItem // computed from NavFiles
}

// datatypes for template rendering

type Page struct {
	Title   string
	Content template.HTML
	Nav     []NavItem
	User    *AuthClaims
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
	Email string
}

type SetupPage struct {
	Error string
	Token string
	Email string
}

type ChangePasswordPage struct {
	Error   string
	Success string
	User    *AuthClaims
	Nav     []NavItem
}

type AddUsersPage struct {
	Error   string
	Success string
	User    *AuthClaims
	Nav     []NavItem
}

type ManageUsersPage struct {
	Error        string
	Success      string
	Users        []*User
	TotalUsers   int
	SetupUsers   int
	PendingUsers int
	User         *AuthClaims
	Nav          []NavItem
}

const siteConfigFname = "site-config.toml"
const serverConfigFname = "server-config.toml"

var config *Config
var authManager *AuthManager

func init() {
	if len(os.Args) != 2 {
		panic("Expected config file path as command-line argument.")
	}

	// Initialize config from toml files
	serverConfigFile := os.Args[1]
	var serverConfig ServerConfig
	var siteConfig SiteConfig
	_, err := toml.DecodeFile(serverConfigFile, &serverConfig)
	if err != nil {
		panic(fmt.Sprintf("Bad server config file %s: %v", serverConfigFile, err))
	}
	siteDir := filepath.Join(serverConfig.SiteDir, siteConfigFname)
	_, err = toml.DecodeFile(siteDir, &siteConfig)
	if err != nil {
		panic(fmt.Sprintf("Bad site config file %s: %v", siteDir, err))
	}
	config = &Config{SiteConfig: siteConfig, ServerConfig: serverConfig}

	// Initialize auth manager
	authManager, err = NewAuthManager(config.DBPath)
	if err != nil {
		log.Fatal("Error initializing auth manager: ", err)
	}

	checkSiteFiles()

	// Set computed config fields
	config.navItems = mkNavItems(config.NavFiles)
	config.templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal("Error parsing templates: ", err)
	}

	// Env can override selected config field values
	port := os.Getenv("PORT")
	if port != "" {
		config.Port = port
	}
	if os.Getenv("USER") == "comp3007" {
		// we're on the SCS site
		config.SiteDir = "../site"
	}

	setupRouting()
}

func main() {
	fmt.Println("Server starting on port " + config.Port)
	log.Print(http.ListenAndServe(":"+config.Port, nil))
}

func setupRouting() {

	// Public routes
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/setup", handleSetup)
	http.HandleFunc("/logout", handleLogout)

	// Protected routes
	http.HandleFunc("/", authManager.RequireAuth(handleAll))
	http.HandleFunc("/change-password", authManager.RequireAuth(handleChangePassword))

	// Admin-only routes
	http.HandleFunc("/admin/add-users", authManager.RequireAdmin(handleAddUsers))
	http.HandleFunc("/admin/manage-users", authManager.RequireAdmin(handleManageUsers))
	http.HandleFunc("/admin/resend-setup-email", authManager.RequireAdmin(handleResendSetupEmail))

	// Upload route
	if config.UploadsAllowed {
		url := filepath.Join(uploadRequestURL(), "{filename}")
		fmt.Print("Adding handler for: ", url)
		http.HandleFunc(url, authManager.RequireAuth(handleUpload))
	}
}

func uploadRequestURL() string {
	return filepath.Join("/upload", config.Secret)
}

func checkSiteFiles() {
	if !dirExists(config.SiteDir) {
		log.Fatal("site directory does not exist: ", config.SiteDir)
	}
	for _, file := range config.NavFiles {
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

func notFound(w http.ResponseWriter, msg string) {
	http.Error(w, "404 page not found: "+msg, http.StatusNotFound)
	log.Print("Not found" + msg)
}

// Handle login page and authentication
func handleLogin(w http.ResponseWriter, r *http.Request) {
	// If user is already logged in, redirect to home
	if cookie, err := r.Cookie("auth_token"); err == nil {
		if _, err := authManager.ValidateJWT(cookie.Value); err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	if r.Method == "GET" {
		loginPage := LoginPage{}
		if err := config.templates.ExecuteTemplate(w, "user-login.html", loginPage); err != nil {
			panicf("Error executing login template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == "POST" {
		email := r.FormValue("email")
		password := r.FormValue("password")

		user, err := authManager.ValidateCredentials(email, password)
		if err != nil {
			loginPage := LoginPage{Error: "Invalid email or password", Email: email}
			if err := config.templates.ExecuteTemplate(w, "user-login.html", loginPage); err != nil {
				panicf("Error executing login template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Generate JWT token
		token, err := authManager.GenerateJWT(user)
		if err != nil {
			panicf("Error generating JWT: %v", err)
			loginPage := LoginPage{Error: "Authentication failed", Email: email}
			if err := config.templates.ExecuteTemplate(w, "user-login.html", loginPage); err != nil {
				panicf("Error executing login template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Set JWT cookie
		cookie := &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // Set to true if using HTTPS
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400, // 24 hours
		}
		http.SetCookie(w, cookie)

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleSetup(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Setup token is required", http.StatusBadRequest)
		return
	}

	user, err := authManager.GetUserBySetupToken(token)
	if err != nil || user == nil {
		http.Error(w, "Invalid or expired setup token", http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		setupPage := SetupPage{Token: token, Email: user.Email}
		if err := config.templates.ExecuteTemplate(w, "setup.html", setupPage); err != nil {
			panicf("Error executing setup template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == "POST" {
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")
		formToken := r.FormValue("token")

		if formToken != token {
			setupPage := SetupPage{Error: "Invalid token", Token: token, Email: user.Email}
			if err := config.templates.ExecuteTemplate(w, "setup.html", setupPage); err != nil {
				panicf("Error executing setup template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		if len(password) < 8 {
			setupPage := SetupPage{Error: "Password must be at least 8 characters long", Token: token, Email: user.Email}
			if err := config.templates.ExecuteTemplate(w, "setup.html", setupPage); err != nil {
				panicf("Error executing setup template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		if password != confirmPassword {
			setupPage := SetupPage{Error: "Passwords do not match", Token: token, Email: user.Email}
			if err := config.templates.ExecuteTemplate(w, "setup.html", setupPage); err != nil {
				panicf("Error executing setup template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		err := authManager.SetupUserPassword(token, password)
		if err != nil {
			panicf("Error setting up user password: %v", err)
			setupPage := SetupPage{Error: "Failed to set up account. Please try again.", Token: token, Email: user.Email}
			if err := config.templates.ExecuteTemplate(w, "setup.html", setupPage); err != nil {
				panicf("Error executing setup template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Redirect to login with success message
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear the auth cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Delete the cookie
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	userClaims := GetUserFromContext(r.Context())
	if userClaims == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "GET" {
		page := ChangePasswordPage{User: userClaims, Nav: config.navItems}
		if err := config.templates.ExecuteTemplate(w, "change-password.html", page); err != nil {
			panicf("Error executing change password template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == "POST" {
		currentPassword := r.FormValue("current_password")
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")

		// Validate current password
		user, err := authManager.GetUserByID(userClaims.UserID)
		if err != nil || user == nil {
			page := ChangePasswordPage{Error: "User not found", User: userClaims, Nav: config.navItems}
			if err := config.templates.ExecuteTemplate(w, "change-password.html", page); err != nil {
				panicf("Error executing change password template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		_, err = authManager.ValidateCredentials(user.Email, currentPassword)
		if err != nil {
			page := ChangePasswordPage{Error: "Current password is incorrect", User: userClaims, Nav: config.navItems}
			if err := config.templates.ExecuteTemplate(w, "change-password.html", page); err != nil {
				panicf("Error executing change password template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		if len(newPassword) < 8 {
			page := ChangePasswordPage{Error: "New password must be at least 8 characters long", User: userClaims, Nav: config.navItems}
			if err := config.templates.ExecuteTemplate(w, "change-password.html", page); err != nil {
				panicf("Error executing change password template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		if newPassword != confirmPassword {
			page := ChangePasswordPage{Error: "New passwords do not match", User: userClaims, Nav: config.navItems}
			if err := config.templates.ExecuteTemplate(w, "change-password.html", page); err != nil {
				panicf("Error executing change password template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		if currentPassword == newPassword {
			page := ChangePasswordPage{Error: "New password must be different from current password", User: userClaims, Nav: config.navItems}
			if err := config.templates.ExecuteTemplate(w, "change-password.html", page); err != nil {
				panicf("Error executing change password template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		err = authManager.UpdateUserPassword(userClaims.UserID, newPassword)
		if err != nil {
			panicf("Error updating user password: %v", err)
			page := ChangePasswordPage{Error: "Failed to update password. Please try again.", User: userClaims, Nav: config.navItems}
			if err := config.templates.ExecuteTemplate(w, "change-password.html", page); err != nil {
				panicf("Error executing change password template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		page := ChangePasswordPage{Success: "Password updated successfully", User: userClaims, Nav: config.navItems}
		if err := config.templates.ExecuteTemplate(w, "change-password.html", page); err != nil {
			panicf("Error executing change password template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleAddUsers(w http.ResponseWriter, r *http.Request) {
	userClaims := GetUserFromContext(r.Context())
	if userClaims == nil || !userClaims.IsAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if r.Method == "GET" {
		page := AddUsersPage{User: userClaims, Nav: config.navItems}
		if err := config.templates.ExecuteTemplate(w, "admin-add-users.html", page); err != nil {
			panicf("Error executing add users template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == "POST" {
		formType := r.FormValue("type")

		if formType == "single" {
			email := strings.TrimSpace(r.FormValue("email"))
			isAdmin := r.FormValue("is_admin") == "on"

			if email == "" {
				page := AddUsersPage{Error: "Email address is required", User: userClaims, Nav: config.navItems}
				if err := config.templates.ExecuteTemplate(w, "admin-add-users.html", page); err != nil {
					panicf("Error executing add users template: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}

			// Check if user already exists
			existingUser, err := authManager.GetUserByEmail(email)
			if err != nil {
				panicf("Error checking for existing user: %v", err)
				page := AddUsersPage{Error: "Failed to check for existing user", User: userClaims, Nav: config.navItems}
				if err := config.templates.ExecuteTemplate(w, "admin-add-users.html", page); err != nil {
					panicf("Error executing add users template: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}

			if existingUser != nil {
				page := AddUsersPage{Error: fmt.Sprintf("User with email %s already exists", email), User: userClaims, Nav: config.navItems}
				if err := config.templates.ExecuteTemplate(w, "admin-add-users.html", page); err != nil {
					panicf("Error executing add users template: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}

			user, err := authManager.CreateUser(email, isAdmin)
			if err != nil {
				panicf("Error creating user: %v", err)
				page := AddUsersPage{Error: "Failed to create user", User: userClaims, Nav: config.navItems}
				if err := config.templates.ExecuteTemplate(w, "admin-add-users.html", page); err != nil {
					panicf("Error executing add users template: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}

			// Send setup email
			err = authManager.SendSetupEmail(user, fmt.Sprintf("http://%s", r.Host))
			if err != nil {
				panicf("Error sending setup email: %v", err)
			}

			page := AddUsersPage{Success: fmt.Sprintf("User %s created successfully. Setup email sent.", email), User: userClaims, Nav: config.navItems}
			if err := config.templates.ExecuteTemplate(w, "admin-add-users.html", page); err != nil {
				panicf("Error executing add users template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return

		} else if formType == "bulk" {
			emailsText := r.FormValue("emails")
			bulkAdmin := r.FormValue("bulk_admin") == "on"

			emails := strings.Split(emailsText, "\n")
			var createdUsers []string
			var skippedUsers []string
			var errorUsers []string

			for _, email := range emails {
				email = strings.TrimSpace(email)
				if email == "" {
					continue
				}

				// Basic email validation
				if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
					skippedUsers = append(skippedUsers, email+" (invalid format)")
					continue
				}

				// Check if user already exists
				existingUser, err := authManager.GetUserByEmail(email)
				if err != nil {
					errorUsers = append(errorUsers, email+" (database error)")
					continue
				}

				if existingUser != nil {
					skippedUsers = append(skippedUsers, email+" (already exists)")
					continue
				}

				user, err := authManager.CreateUser(email, bulkAdmin)
				if err != nil {
					errorUsers = append(errorUsers, email+" (creation failed)")
					continue
				}

				// Send setup email
				err = authManager.SendSetupEmail(user, fmt.Sprintf("http://%s", r.Host))
				if err != nil {
					panicf("Error sending setup email to %s: %v", email, err)
				}

				createdUsers = append(createdUsers, email)
			}

			var message string
			if len(createdUsers) > 0 {
				message = fmt.Sprintf("Successfully created %d users: %s", len(createdUsers), strings.Join(createdUsers, ", "))
			}
			if len(skippedUsers) > 0 {
				if message != "" {
					message += ". "
				}
				message += fmt.Sprintf("Skipped %d users: %s", len(skippedUsers), strings.Join(skippedUsers, ", "))
			}
			if len(errorUsers) > 0 {
				if message != "" {
					message += ". "
				}
				message += fmt.Sprintf("Errors with %d users: %s", len(errorUsers), strings.Join(errorUsers, ", "))
			}

			page := AddUsersPage{Success: message, User: userClaims, Nav: config.navItems}
			if err := config.templates.ExecuteTemplate(w, "admin-add-users.html", page); err != nil {
				panicf("Error executing add users template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleManageUsers(w http.ResponseWriter, r *http.Request) {
	userClaims := GetUserFromContext(r.Context())
	if userClaims == nil || !userClaims.IsAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	users, err := authManager.GetAllUsers()
	if err != nil {
		panicf("Error getting all users: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Calculate statistics
	totalUsers := len(users)
	setupUsers := 0
	pendingUsers := 0

	for _, user := range users {
		if user.IsSetup {
			setupUsers++
		} else {
			pendingUsers++
		}
	}

	page := ManageUsersPage{
		Users:        users,
		TotalUsers:   totalUsers,
		SetupUsers:   setupUsers,
		PendingUsers: pendingUsers,
		User:         userClaims,
		Nav:          config.navItems,
	}

	if err := config.templates.ExecuteTemplate(w, "admin-manage-users.html", page); err != nil {
		panicf("Error executing manage users template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func handleResendSetupEmail(w http.ResponseWriter, r *http.Request) {
	userClaims := GetUserFromContext(r.Context())
	if userClaims == nil || !userClaims.IsAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.FormValue("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := authManager.GetUserByID(userID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	if user.IsSetup {
		http.Error(w, "User has already completed setup", http.StatusBadRequest)
		return
	}

	// Generate new setup token
	token, err := authManager.RegenerateSetupToken(userID)
	if err != nil {
		panicf("Error regenerating setup token: %v", err)
		http.Error(w, "Failed to regenerate setup token", http.StatusInternalServerError)
		return
	}

	// Update user with new token
	user.SetupToken = token

	// Send setup email
	err = authManager.SendSetupEmail(user, fmt.Sprintf("http://%s", r.Host))
	if err != nil {
		panicf("Error sending setup email: %v", err)
		http.Error(w, "Failed to send setup email", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/manage-users?success=Setup+email+resent+successfully", http.StatusSeeOther)
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

func serveMarkdownFile(w http.ResponseWriter, r *http.Request, path string) {
	content, err := os.ReadFile(absPath(path))
	if err != nil {
		notFound(w, "Could not read file: "+path)
		return
	}

	rendered := renderMarkdown(content)
	servePage(w, r, mkPage(rendered, path))
}

func serveHTMLFile(w http.ResponseWriter, r *http.Request, path string) {
	htmlContent, err := os.ReadFile(absPath(path))
	if err != nil {
		notFound(w, "Could not read file: "+path)
		return
	}
	servePage(w, r, mkPage(htmlContent, path))
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

func servePage(w http.ResponseWriter, r *http.Request, page *Page) {
	// Get user information from context if available
	userClaims := GetUserFromContext(r.Context())
	page.User = userClaims

	if err := config.templates.ExecuteTemplate(w, "base.html", page); err != nil {
		panicf("Error executing template: %v", err)
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

	panicf("Receiving file upload: %s -> %s", path, filePath)

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		panicf("Error creating file %s: %v", filePath, err)
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Copy the request body to the file
	bytesWritten, err := io.Copy(file, r.Body)
	if err != nil {
		panicf("Error writing file %s: %v", filePath, err)
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	panicf("Successfully uploaded file: %s (%d bytes)", filename, bytesWritten)

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

// loadEnvFile loads environment variables from a .env file
// func loadEnvFile(filename string) {
// 	file, err := os.Open(filename)
// 	if err != nil {
// 		// .env file is optional, don't error if it doesn't exist
// 		return
// 	}
// 	defer file.Close()

// 	scanner := bufio.NewScanner(file)
// 	for scanner.Scan() {
// 		line := strings.TrimSpace(scanner.Text())

// 		// Skip empty lines and comments
// 		if line == "" || strings.HasPrefix(line, "#") {
// 			continue
// 		}

// 		// Split on first = sign
// 		parts := strings.SplitN(line, "=", 2)
// 		if len(parts) == 2 {
// 			key := strings.TrimSpace(parts[0])
// 			value := strings.TrimSpace(parts[1])

// 			// Only set if not already set in environment
// 			if os.Getenv(key) == "" {
// 				os.Setenv(key, value)
// 			}
// 		}
// 	}

// 	if err := scanner.Err(); err != nil {
// 		panicf("Error reading .env file: %v", err)
// 	}
// }

func panicf(format string, args ...any) {
	panic(fmt.Sprintf(format, args...))
}
