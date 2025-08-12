package preview

import (
	_ "embed"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/russross/blackfriday/v2"
)

//go:embed templates/index.html
var indexTemplate string

//go:embed templates/preview.html
var previewTemplate string

type Server struct {
	NotesDir string
	Port     int
}

type FolderGroup struct {
	Folder string
	Files  []string
}

func NewServer(notesDir string, port int) *Server {
	return &Server{
		NotesDir: notesDir,
		Port:     port,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/preview/", s.handlePreview)
	http.HandleFunc("/static/", s.handleStatic)

	fmt.Printf("Starting markdown preview server on http://localhost:%d\n", s.Port)
	fmt.Printf("Serving notes from: %s\n", s.NotesDir)
	
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	fileGroups, err := s.findMarkdownFilesByFolder()
	if err != nil {
		http.Error(w, "Failed to read notes directory", http.StatusInternalServerError)
		return
	}

	t, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	t.Execute(w, fileGroups)
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/preview/")
	if filename == "" {
		http.Error(w, "No file specified", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.NotesDir, filename)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Configure blackfriday with extensions
	extensions := blackfriday.CommonExtensions | blackfriday.AutoHeadingIDs
	html := blackfriday.Run(content, blackfriday.WithExtensions(extensions))
	
	// Convert markdown checkboxes to HTML checkboxes using regex
	htmlStr := string(html)
	
	// Match unchecked boxes: <li>[ ] content</li> (including multiline and HTML content)
	uncheckedRegex := regexp.MustCompile(`(?s)<li>(?:- )?\[ \]\s*(.*?)</li>`)
	htmlStr = uncheckedRegex.ReplaceAllString(htmlStr, `<li><input type="checkbox" disabled> $1</li>`)
	
	// Match checked boxes: <li>[x] content</li> (including multiline and HTML content)
	checkedRegex := regexp.MustCompile(`(?s)<li>(?:- )?\[x\]\s*(.*?)</li>`)
	htmlStr = checkedRegex.ReplaceAllString(htmlStr, `<li><input type="checkbox" checked disabled> $1</li>`)
	
	html = []byte(htmlStr)
	
	t, err := template.New("preview").Parse(previewTemplate)
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title   string
		Content template.HTML
	}{
		Title:   filename,
		Content: template.HTML(html),
	}

	t.Execute(w, data)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// This could serve static assets if needed in the future
	http.NotFound(w, r)
}

func (s *Server) findMarkdownFilesByFolder() ([]FolderGroup, error) {
	folderMap := make(map[string][]string)
	
	err := filepath.Walk(s.NotesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip hidden directories and files
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".md") {
			relPath, err := filepath.Rel(s.NotesDir, path)
			if err != nil {
				return err
			}
			
			// Get the top-level folder (or "Root" for files in the root)
			parts := strings.Split(relPath, string(filepath.Separator))
			folder := "Root"
			if len(parts) > 1 {
				folder = parts[0]
			}
			
			folderMap[folder] = append(folderMap[folder], relPath)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Convert map to sorted slice
	var folderGroups []FolderGroup
	var folderNames []string
	for folder := range folderMap {
		folderNames = append(folderNames, folder)
	}
	sort.Strings(folderNames)
	
	// Put "Root" first if it exists
	for i, folder := range folderNames {
		if folder == "Root" {
			folderNames[0], folderNames[i] = folderNames[i], folderNames[0]
			break
		}
	}
	
	for _, folder := range folderNames {
		files := folderMap[folder]
		sort.Strings(files)
		folderGroups = append(folderGroups, FolderGroup{
			Folder: folder,
			Files:  files,
		})
	}
	
	return folderGroups, nil
}