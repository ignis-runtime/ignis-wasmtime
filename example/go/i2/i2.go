//go:build wasip1

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ignis-runtime/go-sdk/sdk"
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

// --- Dark Theme Styles ---

const darkStyles = `
<style>
    :root {
        --bg: #0f172a;
        --card-bg: rgba(30, 41, 59, 0.7);
        --accent: #38bdf8;
        --text-main: #f1f5f9;
        --text-dim: #94a3b8;
        --border: rgba(255, 255, 255, 0.1);
    }
    body { 
        font-family: 'Inter', system-ui, sans-serif; 
        background-color: var(--bg);
        background-image: radial-gradient(circle at top right, #1e293b, #0f172a);
        color: var(--text-main); 
        padding: 40px 20px; 
        line-height: 1.6;
        min-height: 100vh;
        margin: 0;
    }
    .container { 
        max-width: 900px; 
        margin: 0 auto; 
        background: var(--card-bg); 
        backdrop-filter: blur(12px);
        padding: 40px; 
        border-radius: 24px; 
        border: 1px solid var(--border);
        box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.3);
    }
    h1 { 
        font-size: 2.5rem;
        font-weight: 800;
        background: linear-gradient(to right, #38bdf8, #818cf8);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        margin-bottom: 30px;
    }
    table { width: 100%; border-collapse: separate; border-spacing: 0 8px; }
    th { text-align: left; padding: 12px; color: var(--text-dim); font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.1em; }
    td { 
        padding: 16px; 
        background: rgba(255, 255, 255, 0.03);
        border-top: 1px solid var(--border);
        border-bottom: 1px solid var(--border);
    }
    td:first-child { border-left: 1px solid var(--border); border-top-left-radius: 12px; border-bottom-left-radius: 12px; }
    td:last-child { border-right: 1px solid var(--border); border-top-right-radius: 12px; border-bottom-right-radius: 12px; }
    
    tr:hover td { background: rgba(255, 255, 255, 0.06); transform: translateY(-1px); transition: all 0.2s; }

    .badge { 
        padding: 4px 10px; 
        border-radius: 6px; 
        font-size: 0.7rem; 
        font-weight: 700; 
        background: rgba(56, 189, 248, 0.1); 
        color: var(--accent); 
        border: 1px solid rgba(56, 189, 248, 0.2);
    }
    a { color: var(--accent); text-decoration: none; transition: opacity 0.2s; }
    a:hover { opacity: 0.8; }
    code { 
        background: rgba(0,0,0,0.3); 
        padding: 4px 8px; 
        border-radius: 6px; 
        font-family: 'Fira Code', monospace; 
        font-size: 0.85em; 
        color: #f472b6;
    }
</style>
`

const homeTemplate = `
<!DOCTYPE html>
<html>
<head><title>Ignis API Index</title>` + darkStyles + `</head>
<body>
    <div class="container">
        <h1>🔥 Ignis Runtime</h1>
        <p style="color: var(--text-dim); margin-bottom: 30px;">High-performance WASM API exploration. Select an endpoint below.</p>
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
                    <td><span class="badge">{{.Method}}</span></td>
                    <td><a href="{{.Path}}"><code>{{.Path}}</code></a></td>
                    <td style="color: var(--text-dim)">{{.Description}}</td>
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
		{"GET", "/api/v1/", "API Dashboard"},
		{"GET", "/api/v1/user", "List all users"},
		{"GET", "/api/v1/user/1", "Get specific user"},
		{"GET", "/api/v1/joke", "Fetch external dad joke"},
		{"GET", "/api/v1/files", "WASM Filesystem Explorer"},
	}
	c.HTML(http.StatusOK, "home", gin.H{"Routes": routes})
}

func HandleListFiles(c *gin.Context) {
	pathToRead := "/"
	files, err := os.ReadDir(pathToRead)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
	// Note: Reusing home styling for simplicity, but you could add a dedicated files template
	c.JSON(http.StatusOK, gin.H{"path": pathToRead, "files": fileEntries})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	var res map[string]any
	json.NewDecoder(resp.Body).Decode(&res)
	c.JSON(http.StatusOK, res)
}

// --- Main ---

func main() {
	// Set Gin to Release Mode for WASM environments to save on footprint
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())

	// Initialize HTML Templates
	t, _ := template.New("home").Parse(homeTemplate)
	r.SetHTMLTemplate(t)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/", HandleHome)
		v1.GET("/user", HandleGetUsers)
		v1.GET("/user/:id", HandleGetUser)
		v1.GET("/joke", HandleJoke)
		v1.GET("/files", HandleListFiles)

		// Static file serving in Gin
		v1.StaticFS("/static", http.Dir("/"))
	}

	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204)
	})

	// Pass the gin engine to the Ignis SDK
	sdk.Handle(r, nil)
}
