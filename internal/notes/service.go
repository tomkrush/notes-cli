package notes

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"notes/internal/config"
	"notes/internal/templates"
)

var directories = []string{"daily", "projects", "meetings", "design", "learning", "todos", "templates", "archive"}

type TaskInfo struct {
	Text     string
	Line     int
	Indent   int
	DueDate  *time.Time
	Tags     []string
	FilePath string
}

type SearchResult struct {
	FilePath string
	Line     int
	Content  string
	Tags     []string
}

type Service struct {
	config *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{config: cfg}
}

func (s *Service) Initialize() error {
	fmt.Printf("Initializing notes folder structure in: %s\n", s.config.BaseDir)
	
	for _, dir := range directories {
		dirPath := filepath.Join(s.config.BaseDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		
		if _, err := os.Stat(dirPath); err == nil {
			fmt.Printf("✓ Created directory: %s/\n", dir)
		}
	}
	
	if err := s.createTemplateFiles(); err != nil {
		return err
	}
	
	if err := s.createReadme(); err != nil {
		return err
	}
	
	if err := s.initGitRepo(); err != nil {
		return err
	}
	
	fmt.Printf("\n✅ Notes folder structure initialized!\n")
	return nil
}

func (s *Service) Create(noteType NoteType, title string) error {
	if err := ValidateNoteType(noteType); err != nil {
		return err
	}
	
	var targetDir, filename string
	
	switch noteType {
	case Daily:
		targetDir = filepath.Join(s.config.BaseDir, "daily")
		filename = time.Now().Format("2006-01-02") + ".md"
	case Project:
		if title == "" {
			return fmt.Errorf("project notes require a title")
		}
		targetDir = filepath.Join(s.config.BaseDir, "projects")
		filename = kebabCase(title) + ".md"
	case Meeting:
		if title == "" {
			return fmt.Errorf("meeting notes require a title")
		}
		targetDir = filepath.Join(s.config.BaseDir, "meetings")
		filename = time.Now().Format("2006-01-02") + "-" + kebabCase(title) + ".md"
	case Design:
		if title == "" {
			return fmt.Errorf("design docs require a title")
		}
		targetDir = filepath.Join(s.config.BaseDir, "design")
		filename = kebabCase(title) + ".md"
	case Learning:
		if title == "" {
			return fmt.Errorf("learning notes require a title")
		}
		targetDir = filepath.Join(s.config.BaseDir, "learning")
		filename = kebabCase(title) + ".md"
	}
	
	filePath := filepath.Join(targetDir, filename)
	
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	if _, err := os.Stat(filePath); err == nil {
		fmt.Printf("⚠ Note already exists: %s\n", filePath)
		fmt.Printf("Opening existing file...\n")
		return s.openEditor(filePath)
	}
	
	templateData := templates.GetTemplateData(title)
	content := templates.Render(noteType, templateData)
	
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}
	
	fmt.Printf("✅ Created new %s note: %s\n", noteType, filePath)
	
	if err := s.commitNote(filePath, fmt.Sprintf("Add %s note: %s", noteType, filename)); err != nil {
		fmt.Printf("⚠ Warning: Failed to commit note to git: %v\n", err)
	}
	
	return s.openEditor(filePath)
}

func (s *Service) List() error {
	fmt.Printf("Existing notes:\n\n")
	
	notesDirs := []string{"daily", "projects", "meetings", "design", "learning"}
	
	for _, dir := range notesDirs {
		dirPath := filepath.Join(s.config.BaseDir, dir)
		
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		
		if len(entries) == 0 {
			continue
		}
		
		fmt.Printf("📁 %s/\n", dir)
		for _, entry := range entries {
			if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".md") || strings.HasSuffix(entry.Name(), ".txt")) {
				fmt.Printf("  %s\n", entry.Name())
			}
		}
		fmt.Println()
	}
	
	return nil
}

func (s *Service) ShowTasks() error {
	fmt.Printf("\033[1;36m📋 Incomplete Tasks\033[0m\n")
	fmt.Printf("\033[90m" + strings.Repeat("─", 50) + "\033[0m\n\n")
	
	allTasks := []TaskInfo{}
	searchDirs := []string{"daily", "projects", "meetings", "design", "learning", "todos"}
	
	for _, dir := range searchDirs {
		dirPath := filepath.Join(s.config.BaseDir, dir)
		
		err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			
			if d.IsDir() || (!strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".txt")) {
				return nil
			}
			
			tasks := s.extractTasks(path)
			allTasks = append(allTasks, tasks...)
			
			return nil
		})
		
		if err != nil {
			continue
		}
	}
	
	if len(allTasks) == 0 {
		fmt.Printf("\033[1;32m✅ No incomplete tasks found!\033[0m\n")
		fmt.Printf("\033[90mYou're all caught up! 🎉\033[0m\n")
		return nil
	}
	
	sort.Slice(allTasks, func(i, j int) bool {
		if allTasks[i].DueDate != nil && allTasks[j].DueDate != nil {
			return allTasks[i].DueDate.Before(*allTasks[j].DueDate)
		}
		if allTasks[i].DueDate != nil {
			return true
		}
		if allTasks[j].DueDate != nil {
			return false
		}
		return allTasks[i].FilePath < allTasks[j].FilePath
	})
	
	currentFile := ""
	overdueTasks := 0
	todayTasks := 0
	
	for _, task := range allTasks {
		relPath, _ := filepath.Rel(s.config.BaseDir, task.FilePath)
		
		if relPath != currentFile {
			if currentFile != "" {
				fmt.Println()
			}
			fmt.Printf("\033[1;34m📝 %s\033[0m\n", relPath)
			currentFile = relPath
		}
		
		priority := s.detectPriority(task.Text)
		priorityColor := s.getPriorityColor(priority)
		indentStr := strings.Repeat("  ", task.Indent/2)
		
		taskDisplay := task.Text
		dueDateStr := ""
		
		if task.DueDate != nil {
			now := time.Now()
			if task.DueDate.Before(now) {
				dueDateStr = fmt.Sprintf(" \033[1;31m(overdue: %s)\033[0m", task.DueDate.Format("Jan 2"))
				overdueTasks++
			} else if task.DueDate.Format("2006-01-02") == now.Format("2006-01-02") {
				dueDateStr = fmt.Sprintf(" \033[1;33m(due today)\033[0m")
				todayTasks++
			} else {
				dueDateStr = fmt.Sprintf(" \033[90m(due: %s)\033[0m", task.DueDate.Format("Jan 2"))
			}
		}
		
		if len(task.Tags) > 0 {
			tagStr := strings.Join(task.Tags, " ")
			taskDisplay = fmt.Sprintf("%s \033[36m%s\033[0m", taskDisplay, tagStr)
		}
		
		if len(taskDisplay) > 60 {
			taskDisplay = taskDisplay[:57] + "..."
		}
		
		fmt.Printf("  %s\033[90mL%d:\033[0m %s%s\033[0m %s%s\n", 
			indentStr, task.Line, priorityColor, priority, taskDisplay, dueDateStr)
	}
	
	fmt.Println()
	fmt.Printf("\033[90m" + strings.Repeat("─", 50) + "\033[0m\n")
	fmt.Printf("\033[1mTotal: %d task%s", len(allTasks), pluralize(len(allTasks)))
	
	if overdueTasks > 0 {
		fmt.Printf(" \033[1;31m(%d overdue)\033[0m", overdueTasks)
	}
	if todayTasks > 0 {
		fmt.Printf(" \033[1;33m(%d due today)\033[0m", todayTasks)
	}
	fmt.Printf("\033[0m\n")
	
	return nil
}

func kebabCase(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	s = reg.ReplaceAllString(s, "-")
	s = strings.ToLower(s)
	s = strings.Trim(s, "-")
	return s
}

func (s *Service) openEditor(filePath string) error {
	editors := []string{"code", os.Getenv("EDITOR"), "nano", "vim"}
	
	for _, editor := range editors {
		if editor == "" {
			continue
		}
		
		if editor == "code" {
			fmt.Printf("📝 Open in editor: %s\n", filePath)
			return nil
		}
	}
	
	fmt.Printf("📄 File created at: %s\n", filePath)
	return nil
}

func (s *Service) detectPriority(taskText string) string {
	taskLower := strings.ToLower(taskText)
	
	highPriorityKeywords := []string{"urgent", "asap", "critical", "important", "!!!"}
	mediumPriorityKeywords := []string{"!!", "soon", "priority"}
	
	for _, keyword := range highPriorityKeywords {
		if strings.Contains(taskLower, keyword) {
			return "🔴"
		}
	}
	
	for _, keyword := range mediumPriorityKeywords {
		if strings.Contains(taskLower, keyword) {
			return "🟡"
		}
	}
	
	return "⚪"
}

func (s *Service) getPriorityColor(priority string) string {
	switch priority {
	case "🔴":
		return "\033[1;31m"
	case "🟡":
		return "\033[1;33m"
	default:
		return "\033[90m"
	}
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func (s *Service) initGitRepo() error {
	gitPath := filepath.Join(s.config.BaseDir, ".git")
	if _, err := os.Stat(gitPath); err == nil {
		fmt.Printf("✓ Git repository already exists\n")
		return nil
	}
	
	cmd := exec.Command("git", "init")
	cmd.Dir = s.config.BaseDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}
	
	gitignoreContent := `# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Editor files
.vscode/
.idea/
*.swp
*.swo
*~

# Temporary files
*.tmp
*.temp
`
	
	gitignorePath := filepath.Join(s.config.BaseDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}
	
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = s.config.BaseDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files to git: %w", err)
	}
	
	cmd = exec.Command("git", "commit", "-m", "Initial commit: Set up notes structure")
	cmd.Dir = s.config.BaseDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}
	
	fmt.Printf("✓ Initialized git repository\n")
	return nil
}

func (s *Service) commitNote(filePath, message string) error {
	cmd := exec.Command("git", "add", filePath)
	cmd.Dir = s.config.BaseDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add file to git: %w", err)
	}
	
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = s.config.BaseDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit file: %w", err)
	}
	
	return nil
}

func (s *Service) SaveChanges(message string) error {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = s.config.BaseDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}
	
	if len(output) == 0 {
		fmt.Printf("✅ No changes to commit\n")
		return nil
	}
	
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = s.config.BaseDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files to git: %w", err)
	}
	
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = s.config.BaseDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	
	fmt.Printf("✅ Changes committed: %s\n", message)
	return nil
}

func (s *Service) extractTasks(filePath string) []TaskInfo {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()
	
	taskPattern := regexp.MustCompile(`^(\s*)-\s*\[\s*\]\s*(.*)$`)
	dueDatePattern := regexp.MustCompile(`due:(\d{4}-\d{2}-\d{2})`)
	tagPattern := regexp.MustCompile(`#(\w+)`)
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	tasks := []TaskInfo{}
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
		if match := taskPattern.FindStringSubmatch(line); match != nil {
			taskText := strings.TrimSpace(match[2])
			indent := len(match[1])
			
			task := TaskInfo{
				Text:     taskText,
				Line:     lineNum,
				Indent:   indent,
				FilePath: filePath,
			}
			
			if dueDateMatch := dueDatePattern.FindStringSubmatch(taskText); dueDateMatch != nil {
				if dueDate, err := time.Parse("2006-01-02", dueDateMatch[1]); err == nil {
					task.DueDate = &dueDate
				}
				task.Text = dueDatePattern.ReplaceAllString(task.Text, "")
				task.Text = strings.TrimSpace(task.Text)
			}
			
			tagMatches := tagPattern.FindAllStringSubmatch(taskText, -1)
			for _, tagMatch := range tagMatches {
				task.Tags = append(task.Tags, "#"+tagMatch[1])
			}
			
			tasks = append(tasks, task)
		}
	}
	
	return tasks
}

func (s *Service) Search(query string, searchTags []string) error {
	fmt.Printf("\033[1;36m🔍 Search Results for: \"%s\"\033[0m", query)
	if len(searchTags) > 0 {
		fmt.Printf(" \033[36m%s\033[0m", strings.Join(searchTags, " "))
	}
	fmt.Printf("\n")
	fmt.Printf("\033[90m" + strings.Repeat("─", 50) + "\033[0m\n\n")
	
	results := []SearchResult{}
	searchDirs := []string{"daily", "projects", "meetings", "design", "learning", "todos"}
	
	queryLower := strings.ToLower(query)
	
	for _, dir := range searchDirs {
		dirPath := filepath.Join(s.config.BaseDir, dir)
		
		err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			
			if d.IsDir() || (!strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".txt")) {
				return nil
			}
			
			fileResults := s.searchInFile(path, queryLower, searchTags)
			results = append(results, fileResults...)
			
			return nil
		})
		
		if err != nil {
			continue
		}
	}
	
	if len(results) == 0 {
		fmt.Printf("\033[90mNo results found.\033[0m\n")
		return nil
	}
	
	currentFile := ""
	for _, result := range results {
		relPath, _ := filepath.Rel(s.config.BaseDir, result.FilePath)
		
		if relPath != currentFile {
			if currentFile != "" {
				fmt.Println()
			}
			fmt.Printf("\033[1;34m📄 %s\033[0m\n", relPath)
			currentFile = relPath
		}
		
		content := result.Content
		if len(content) > 100 {
			content = content[:97] + "..."
		}
		
		tagStr := ""
		if len(result.Tags) > 0 {
			tagStr = fmt.Sprintf(" \033[36m%s\033[0m", strings.Join(result.Tags, " "))
		}
		
		fmt.Printf("  \033[90mL%d:\033[0m %s%s\n", result.Line, content, tagStr)
	}
	
	fmt.Printf("\n\033[90m" + strings.Repeat("─", 50) + "\033[0m\n")
	fmt.Printf("\033[1mFound %d result%s\033[0m\n", len(results), pluralize(len(results)))
	
	return nil
}

func (s *Service) searchInFile(filePath, query string, searchTags []string) []SearchResult {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()
	
	tagPattern := regexp.MustCompile(`#(\w+)`)
	results := []SearchResult{}
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lineLower := strings.ToLower(line)
		
		lineMatches := query == "" || strings.Contains(lineLower, query)
		
		lineTags := []string{}
		tagMatches := tagPattern.FindAllStringSubmatch(line, -1)
		for _, tagMatch := range tagMatches {
			lineTags = append(lineTags, "#"+tagMatch[1])
		}
		
		tagSearchMatches := len(searchTags) == 0
		if len(searchTags) > 0 {
			for _, searchTag := range searchTags {
				for _, lineTag := range lineTags {
					if strings.EqualFold(lineTag, searchTag) {
						tagSearchMatches = true
						break
					}
				}
				if tagSearchMatches {
					break
				}
			}
		}
		
		if lineMatches && tagSearchMatches {
			results = append(results, SearchResult{
				FilePath: filePath,
				Line:     lineNum,
				Content:  strings.TrimSpace(line),
				Tags:     lineTags,
			})
		}
	}
	
	return results
}