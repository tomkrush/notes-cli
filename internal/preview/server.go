package preview

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/russross/blackfriday/v2"
)

type Server struct {
	NotesDir string
	Port     int
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
	files, err := s.findMarkdownFiles()
	if err != nil {
		http.Error(w, "Failed to read notes directory", http.StatusInternalServerError)
		return
	}

	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Notes Preview</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; margin: 40px; }
        .file-list { list-style: none; padding: 0; }
        .file-list li { margin: 10px 0; }
        .file-list a { text-decoration: none; color: #0066cc; font-size: 16px; }
        .file-list a:hover { text-decoration: underline; }
        h1 { color: #333; }
    </style>
</head>
<body>
    <h1>Notes Preview</h1>
    <ul class="file-list">
        {{range .}}
        <li><a href="/preview/{{.}}">{{.}}</a></li>
        {{end}}
    </ul>
</body>
</html>`

	t := template.Must(template.New("index").Parse(tmpl))
	t.Execute(w, files)
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/preview/")
	if filename == "" {
		http.Error(w, "No file specified", http.StatusBadRequest)
		return
	}

	filepath := filepath.Join(s.NotesDir, filename)
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	html := blackfriday.Run(content)
	
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, sans-serif; 
            line-height: 1.6; 
            max-width: 800px; 
            margin: 0 auto; 
            padding: 20px; 
            color: #333;
        }
        h1, h2, h3, h4, h5, h6 { color: #2c3e50; }
        code { 
            background-color: #f4f4f4; 
            padding: 2px 4px; 
            border-radius: 3px; 
            font-family: 'Monaco', 'Menlo', monospace;
        }
        pre { 
            background-color: #f8f8f8; 
            border: 1px solid #ddd; 
            border-radius: 5px; 
            padding: 15px; 
            overflow-x: auto; 
        }
        pre code { 
            background: none; 
            padding: 0; 
        }
        blockquote { 
            border-left: 4px solid #ddd; 
            margin: 0; 
            padding-left: 20px; 
            color: #666; 
        }
        table { 
            border-collapse: collapse; 
            width: 100%; 
        }
        th, td { 
            border: 1px solid #ddd; 
            padding: 8px 12px; 
            text-align: left; 
        }
        th { 
            background-color: #f2f2f2; 
        }
        .mermaid { 
            text-align: center; 
            margin: 20px 0; 
        }
        .back-link { 
            display: inline-block; 
            margin-bottom: 20px; 
            text-decoration: none; 
            color: #0066cc; 
        }
        .back-link:hover { 
            text-decoration: underline; 
        }
        /* Task list styling */
        ul li[data-task] { list-style: none; margin-left: -20px; }
        input[type="checkbox"] { margin-right: 8px; }
        input[type="checkbox"]:checked + * { text-decoration: line-through; opacity: 0.6; }
    </style>
</head>
<body>
    <a href="/" class="back-link">← Back to file list</a>
    <div id="content">{{.Content}}</div>
    
    <script>
        mermaid.initialize({ 
            startOnLoad: true,
            theme: 'default',
            themeVariables: {
                primaryColor: '#0066cc',
                primaryTextColor: '#333',
                primaryBorderColor: '#0066cc',
                lineColor: '#666',
                secondaryColor: '#f4f4f4',
                tertiaryColor: '#fff'
            }
        });

        // Convert code blocks with language 'mermaid' to mermaid diagrams
        document.addEventListener('DOMContentLoaded', function() {
            const codeBlocks = document.querySelectorAll('pre code');
            codeBlocks.forEach(function(block) {
                const text = block.textContent.trim();
                if (text.startsWith('graph ') || text.startsWith('flowchart ') || 
                    text.startsWith('sequenceDiagram') || text.startsWith('classDiagram') ||
                    text.startsWith('gantt') || text.startsWith('gitGraph') ||
                    text.startsWith('erDiagram') || text.startsWith('journey') ||
                    text.startsWith('pie ') || text.startsWith('stateDiagram')) {
                    
                    const mermaidDiv = document.createElement('div');
                    mermaidDiv.className = 'mermaid';
                    mermaidDiv.textContent = text;
                    block.parentNode.replaceWith(mermaidDiv);
                }
            });
            
            // Re-initialize mermaid after adding new diagrams
            mermaid.init();
        });
    </script>
</body>
</html>`

	data := struct {
		Title   string
		Content template.HTML
	}{
		Title:   filename,
		Content: template.HTML(html),
	}

	t := template.Must(template.New("preview").Parse(tmpl))
	t.Execute(w, data)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// This could serve static assets if needed in the future
	http.NotFound(w, r)
}

func (s *Server) findMarkdownFiles() ([]string, error) {
	var files []string
	
	err := filepath.Walk(s.NotesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".md") {
			relPath, err := filepath.Rel(s.NotesDir, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		
		return nil
	})
	
	return files, err
}