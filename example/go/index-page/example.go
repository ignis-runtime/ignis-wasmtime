package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ignis-runtime/go-sdk/sdk"

	"github.com/gin-gonic/gin"
)

// --- Models & Data ---

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type RouteInfo struct {
	Method      string
	Path        string
	Description string
}

var Users = map[int]User{
	1: {ID: 1, Username: "FastTiger123", CreatedAt: time.Date(2023, 8, 14, 15, 30, 0, 0, time.UTC)},
	2: {ID: 2, Username: "CleverEagle99", CreatedAt: time.Date(2023, 11, 7, 10, 15, 0, 0, time.UTC)},
	3: {ID: 3, Username: "BraveWolf42", CreatedAt: time.Date(2024, 2, 19, 20, 45, 0, 0, time.UTC)},
}

// --- HTML Templates ---

const baseStyles = `
<style>
    body { font-family: 'Inter', -apple-system, sans-serif; background: #f4f7f9; color: #2d3748; padding: 40px; line-height: 1.6; }
    .container { max-width: 900px; margin: 0 auto; background: white; padding: 30px; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.05); }
    h1 { color: #1a202c; border-bottom: 2px solid #edf2f7; padding-bottom: 15px; margin-bottom: 25px; }
    table { width: 100%; border-collapse: collapse; }
    th { text-align: left; background: #f8fafc; padding: 12px; color: #64748b; font-size: 0.85rem; text-transform: uppercase; letter-spacing: 0.05em; }
    td { padding: 15px 12px; border-bottom: 1px solid #edf2f7; }
    tr:last-child td { border-bottom: none; }
    .badge { padding: 4px 8px; border-radius: 4px; font-size: 0.75rem; font-weight: bold; background: #ebf8ff; color: #3182ce; }
    .method-get { background: #f0fff4; color: #38a169; }
    a { color: #3182ce; text-decoration: none; font-weight: 500; }
    a:hover { text-decoration: underline; }
    code { background: #f1f5f9; padding: 2px 6px; border-radius: 4px; font-family: monospace; font-size: 0.9em; }
</style>
`

const homeTemplate = `
<!DOCTYPE html>
<html>
<head><title>Ignis API Index</title>` + baseStyles + `</head>
<body>
    <div class="container">
        <h1>🔥 Ignis Runtime Sandbox</h1>
        <p>Welcome to the Ignis WASM Runtime - Go Example. Below are the available endpoints:</p>
        <table>
            <thead>
                <tr>
                    <th>Method</th>
                    <th>Endpoint</th>
                    <th>Description</th>
                </tr>
            </thead>
            <tbody>
                {{range .Routes}}
                <tr>
                    <td><span class="badge method-{{.Method | lower}}">{{.Method}}</span></td>
                    <td><a href="{{.Path}}"><code>{{.Path}}</code></a></td>
                    <td>{{.Description}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</body>
</html>
`

const filesTemplate = `
<!DOCTYPE html>
<html>
<head><title>Index of {{.Path}}</title>` + baseStyles + `</head>
<body>
    <div class="container">
        <h1>📁 Directory Index</h1>
        <p>Path: <code>{{.Path}}</code></p>
        <table>
            <thead>
                <tr>
                    <th>Name</th>
                    <th>Size</th>
                    <th>Modified</th>
                </tr>
            </thead>
            <tbody>
                <tr>
                    <td><a href="..">⤴️ .. (Parent)</a></td>
                    <td>-</td>
                    <td>-</td>
                </tr>
                {{range .Files}}
                <tr>
                    <td>
                        {{if .IsDir}}📁{{else}}📄{{end}}
                        <a href="/api/v1/static/{{.Name}}">{{.Name}}</a>
                    </td>
                    <td>{{if .IsDir}}-{{else}}{{.Size}}{{end}}</td>
                    <td>{{.ModTime}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</body>
</html>
`

// --- Handlers ---

func HandleHome(c *gin.Context) {
	routes := []RouteInfo{
		{"GET", "/api/v1/", "This documentation page"},
		{"GET", "/api/v1/user", "Retrieve all registered users"},
		{"GET", "/api/v1/user/:id", "Retrieve a specific user by ID (e.g., /user/1)"},
		{"GET", "/api/v1/joke", "Get a random dad joke from an external API"},
		{"GET", "/api/v1/files", "Browse the server filesystem (FTP-style)"},
	}

	funcMap := template.FuncMap{"lower": func(s string) string { return "get" }} // Simplified for this example
	tmpl, _ := template.New("home").Funcs(funcMap).Parse(homeTemplate)
	c.Header("Content-Type", "text/html")
	tmpl.Execute(c.Writer, gin.H{"Routes": routes})
}

func HandleListFiles(c *gin.Context) {
	pathToRead := "/"
	files, err := os.ReadDir(pathToRead)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var fileEntries []map[string]any
	for _, file := range files {
		info, _ := file.Info()
		fileEntries = append(fileEntries, map[string]any{
			"Name":    file.Name(),
			"IsDir":   file.IsDir(),
			"Size":    fmt.Sprintf("%.1f KB", float64(info.Size())/1024),
			"ModTime": info.ModTime().Format("2006-01-02 15:04"),
		})
	}

	tmpl, _ := template.New("files").Parse(filesTemplate)
	c.Header("Content-Type", "text/html")
	tmpl.Execute(c.Writer, gin.H{"Path": pathToRead, "Files": fileEntries})
}

func HandleGetUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	user, ok := Users[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func HandleGetUsers(c *gin.Context) {
	c.JSON(http.StatusOK, Users)
}

func HandleJoke(c *gin.Context) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "https://icanhazdadjoke.com", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	var res map[string]any
	json.NewDecoder(resp.Body).Decode(&res)
	c.JSON(200, res)
}

// --- Main ---

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	{
		// Home Route (Discovery)
		v1.GET("/", HandleHome)

		// Resource Routes
		v1.GET("/user", HandleGetUsers)
		v1.GET("/user/:id", HandleGetUser)
		v1.GET("/joke", HandleJoke)

		// File Browser Routes
		v1.GET("/files", HandleListFiles)
		v1.StaticFS("/static", http.Dir("/")) // Serve actual files

		v1.GET("/favicon.ico", func(c *gin.Context) {
			c.Status(204)
		})
	}

	sdk.Handle(router, nil)
}
