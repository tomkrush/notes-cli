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

var directories = []string{"daily", "projects", "meetings", "design", "learning", "todos", "archive"}

type TaskInfo struct {
	Text        string
	Line        int
	Indent      int
	DueDate     *time.Time
	Tags        []string
	FilePath    string
	Estimate    string
	TimeEntries []TimeEntry
	TotalTime   time.Duration
	Remaining   string
	IsActive    bool
}

type TimeEntry struct {
	Date        time.Time
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	Description string
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

func (s *Service) ShowTasks(filters TaskFilters) error {
	// Apply smart defaults if no explicit flags
	if !filters.All && !filters.Focus && !filters.Overdue && !filters.Today && len(filters.Tags) == 0 && filters.Priority == "" && filters.FilePattern == "" && !filters.Summary && !filters.Full {
		// Check current context
		context := s.detectCurrentContext()
		if context != "" {
			filters.FilePattern = context
			fmt.Printf("\033[1;36m📋 Tasks in %s/\033[0m\n", context)
		} else {
			// Default to focus mode (overdue + today)
			filters.Focus = true
			fmt.Printf("\033[1;36m📋 Focus: Overdue & Today's Tasks\033[0m\n")
		}
	} else if filters.Summary {
		fmt.Printf("\033[1;36m📊 Task Overview\033[0m\n")
	} else if filters.Focus {
		fmt.Printf("\033[1;36m📋 Focus: Overdue & Today's Tasks\033[0m\n")
	} else {
		fmt.Printf("\033[1;36m📋 All Incomplete Tasks\033[0m\n")
	}
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
	
	// Apply focus filter if needed
	if filters.Focus {
		filters.Overdue = true
		filters.Today = true
	}
	
	filteredTasks := s.filterTasks(allTasks, filters)
	
	if len(filteredTasks) == 0 {
		if len(allTasks) == 0 {
			fmt.Printf("\033[1;32m✅ No incomplete tasks found!\033[0m\n")
			fmt.Printf("\033[90mYou're all caught up! 🎉\033[0m\n")
		} else {
			fmt.Printf("\033[1;33m⚠ No tasks match your filters\033[0m\n")
			fmt.Printf("\033[90mTry adjusting your filter criteria\033[0m\n")
		}
		return nil
	}
	
	// Handle summary mode
	if filters.Summary {
		return s.showTaskSummary(filteredTasks)
	}
	
	s.sortTasks(filteredTasks, filters.SortBy)
	
	// Show information scent (category overview) for full view
	if !filters.Summary {
		s.showInformationScent(filteredTasks)
	}
	
	currentFile := ""
	overdueTasks := 0
	todayTasks := 0
	
	for _, task := range filteredTasks {
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
			todayStr := now.Format("2006-01-02")
			dueDateStr := task.DueDate.Format("2006-01-02")
			relativeTime := formatRelativeTime(task.DueDate)
			
			if dueDateStr < todayStr {
				dueDateStr = fmt.Sprintf(" \033[1;31m(%s)\033[0m", relativeTime)
				overdueTasks++
			} else if dueDateStr == todayStr {
				dueDateStr = fmt.Sprintf(" \033[1;33m(due %s)\033[0m", relativeTime)
				todayTasks++
			} else {
				dueDateStr = fmt.Sprintf(" \033[90m(due %s)\033[0m", relativeTime)
			}
		}
		
		if len(task.Tags) > 0 {
			tagStr := strings.Join(task.Tags, " ")
			taskDisplay = fmt.Sprintf("%s \033[36m%s\033[0m", taskDisplay, tagStr)
		}
		
		if len(taskDisplay) > 60 {
			taskDisplay = taskDisplay[:57] + "..."
		}
		
		// Use tree characters for better visual hierarchy
		treeChar := "├─"
		if task.Indent > 0 {
			treeChar = "└─"
		}
		
		// Add effort estimate and time tracking info
		estimate := task.Estimate
		if estimate == "" {
			estimate = estimateTaskEffort(task.Text)
		}
		
		// Show time tracking information
		timeInfo := ""
		if task.TotalTime > 0 {
			totalStr := formatDuration(task.TotalTime)
			if task.Remaining != "" {
				timeInfo = fmt.Sprintf(" \033[33m[%s worked, %s left]\033[0m", totalStr, task.Remaining)
			} else if task.Estimate != "" {
				if estimateDur, err := parseDuration(task.Estimate); err == nil {
					if task.TotalTime >= estimateDur {
						timeInfo = fmt.Sprintf(" \033[32m[%s completed]\033[0m", totalStr)
					} else {
						timeInfo = fmt.Sprintf(" \033[33m[%s/%s]\033[0m", totalStr, formatDuration(estimateDur))
					}
				} else {
					timeInfo = fmt.Sprintf(" \033[33m[%s worked]\033[0m", totalStr)
				}
			} else {
				timeInfo = fmt.Sprintf(" \033[33m[%s worked]\033[0m", totalStr)
			}
		}
		
		fmt.Printf("  %s%s %s%s\033[0m %s%s%s \033[90m~%s (L%d)\033[0m\n", 
			indentStr, treeChar, priorityColor, priority, taskDisplay, dueDateStr, timeInfo, estimate, task.Line)
	}
	
	fmt.Println()
	fmt.Printf("\033[90m" + strings.Repeat("─", 50) + "\033[0m\n")
	fmt.Printf("\033[1mTotal: %d task%s", len(filteredTasks), pluralize(len(filteredTasks)))
	
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
	estimatePattern := regexp.MustCompile(`est:([^\\s]+)`)
	tagPattern := regexp.MustCompile(`#(\w+)`)
	timeLogPattern := regexp.MustCompile(`^\s*Time log:\s*$`)
	timeEntryPattern := regexp.MustCompile(`^\s*•`)
	remainingPattern := regexp.MustCompile(`^\s*Remaining:\s*(.+)$`)
	totalPattern := regexp.MustCompile(`^\s*Total:\s*(.+)$`)
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	tasks := []TaskInfo{}
	var currentTask *TaskInfo
	inTimeLog := false
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
		// Check for task line
		if match := taskPattern.FindStringSubmatch(line); match != nil {
			// Save previous task if exists
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}
			
			taskText := strings.TrimSpace(match[2])
			indent := len(match[1])
			
			currentTask = &TaskInfo{
				Text:        taskText,
				Line:        lineNum,
				Indent:      indent,
				FilePath:    filePath,
				TimeEntries: []TimeEntry{},
			}
			inTimeLog = false
			
			// Parse due date
			if dueDateMatch := dueDatePattern.FindStringSubmatch(taskText); dueDateMatch != nil {
				if dueDate, err := time.Parse("2006-01-02", dueDateMatch[1]); err == nil {
					currentTask.DueDate = &dueDate
				}
				currentTask.Text = dueDatePattern.ReplaceAllString(currentTask.Text, "")
			}
			
			// Parse estimate
			if estimateMatch := estimatePattern.FindStringSubmatch(taskText); estimateMatch != nil {
				currentTask.Estimate = estimateMatch[1]
				currentTask.Text = estimatePattern.ReplaceAllString(currentTask.Text, "")
			}
			
			// Parse tags
			tagMatches := tagPattern.FindAllStringSubmatch(taskText, -1)
			for _, tagMatch := range tagMatches {
				currentTask.Tags = append(currentTask.Tags, "#"+tagMatch[1])
			}
			
			currentTask.Text = strings.TrimSpace(currentTask.Text)
			continue
		}
		
		// Skip if no current task
		if currentTask == nil {
			continue
		}
		
		// Check for "Time log:" line
		if timeLogPattern.MatchString(line) {
			inTimeLog = true
			continue
		}
		
		// Parse time entries
		if inTimeLog && timeEntryPattern.MatchString(line) {
			if entry, err := parseTimeEntry(line); err == nil {
				currentTask.TimeEntries = append(currentTask.TimeEntries, *entry)
				currentTask.TotalTime += entry.Duration
			}
			continue
		}
		
		// Parse remaining time
		if remainingMatch := remainingPattern.FindStringSubmatch(line); remainingMatch != nil {
			currentTask.Remaining = strings.TrimSpace(remainingMatch[1])
			inTimeLog = false
			continue
		}
		
		// Parse total time
		if totalMatch := totalPattern.FindStringSubmatch(line); totalMatch != nil {
			if duration, err := parseDuration(strings.TrimSpace(totalMatch[1])); err == nil {
				currentTask.TotalTime = duration
			}
			inTimeLog = false
			continue
		}
		
		// If we hit a non-indented line that's not part of time log, we're done with this task
		if !strings.HasPrefix(line, "  ") && strings.TrimSpace(line) != "" {
			inTimeLog = false
		}
	}
	
	// Don't forget the last task
	if currentTask != nil {
		tasks = append(tasks, *currentTask)
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

func (s *Service) filterTasks(tasks []TaskInfo, filters TaskFilters) []TaskInfo {
	if len(filters.Tags) == 0 && filters.Priority == "" && !filters.Overdue && !filters.Today && filters.FilePattern == "" {
		return tasks
	}
	
	filtered := []TaskInfo{}
	now := time.Now()
	
	for _, task := range tasks {
		if !s.matchesFilters(task, filters, now) {
			continue
		}
		filtered = append(filtered, task)
	}
	
	return filtered
}

func (s *Service) matchesFilters(task TaskInfo, filters TaskFilters, now time.Time) bool {
	if len(filters.Tags) > 0 {
		hasMatchingTag := false
		for _, filterTag := range filters.Tags {
			for _, taskTag := range task.Tags {
				if strings.EqualFold(taskTag, filterTag) {
					hasMatchingTag = true
					break
				}
			}
			if hasMatchingTag {
				break
			}
		}
		if !hasMatchingTag {
			return false
		}
	}
	
	if filters.Priority != "" {
		taskPriority := s.detectPriority(task.Text)
		priorityLevel := s.getPriorityLevel(taskPriority)
		if !s.matchesPriorityFilter(priorityLevel, filters.Priority) {
			return false
		}
	}
	
	// Handle focus mode (both overdue and today) with OR logic
	if filters.Overdue && filters.Today {
		if task.DueDate == nil {
			return false
		}
		taskDateStr := task.DueDate.Format("2006-01-02")
		todayStr := now.Format("2006-01-02")
		isOverdue := taskDateStr < todayStr
		isToday := taskDateStr == todayStr
		if !isOverdue && !isToday {
			return false
		}
	} else {
		// Handle individual filters with AND logic
		if filters.Overdue && (task.DueDate == nil || task.DueDate.Format("2006-01-02") >= now.Format("2006-01-02")) {
			return false
		}
		
		if filters.Today && (task.DueDate == nil || task.DueDate.Format("2006-01-02") != now.Format("2006-01-02")) {
			return false
		}
	}
	
	if filters.FilePattern != "" && !strings.Contains(strings.ToLower(task.FilePath), strings.ToLower(filters.FilePattern)) {
		return false
	}
	
	return true
}

func (s *Service) getPriorityLevel(priorityEmoji string) string {
	switch priorityEmoji {
	case "🔴":
		return "high"
	case "🟡":
		return "medium"
	default:
		return "low"
	}
}

func (s *Service) matchesPriorityFilter(taskPriority, filterPriority string) bool {
	return strings.EqualFold(taskPriority, filterPriority)
}

func (s *Service) sortTasks(tasks []TaskInfo, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "priority":
		sort.Slice(tasks, func(i, j int) bool {
			iPriority := s.detectPriority(tasks[i].Text)
			jPriority := s.detectPriority(tasks[j].Text)
			iPriorityLevel := s.getPriorityLevel(iPriority)
			jPriorityLevel := s.getPriorityLevel(jPriority)
			
			priorityOrder := map[string]int{"high": 0, "medium": 1, "low": 2}
			if priorityOrder[iPriorityLevel] != priorityOrder[jPriorityLevel] {
				return priorityOrder[iPriorityLevel] < priorityOrder[jPriorityLevel]
			}
			
			if tasks[i].DueDate != nil && tasks[j].DueDate != nil {
				return tasks[i].DueDate.Before(*tasks[j].DueDate)
			}
			if tasks[i].DueDate != nil {
				return true
			}
			if tasks[j].DueDate != nil {
				return false
			}
			return tasks[i].FilePath < tasks[j].FilePath
		})
	case "file":
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].FilePath < tasks[j].FilePath
		})
	default:
		sort.Slice(tasks, func(i, j int) bool {
			if tasks[i].DueDate != nil && tasks[j].DueDate != nil {
				return tasks[i].DueDate.Before(*tasks[j].DueDate)
			}
			if tasks[i].DueDate != nil {
				return true
			}
			if tasks[j].DueDate != nil {
				return false
			}
			return tasks[i].FilePath < tasks[j].FilePath
		})
	}
}

func (s *Service) showTaskSummary(tasks []TaskInfo) error {
	stats := analyzeTaskStats(tasks, s)
	
	// Show category counts with visual indicators
	fmt.Printf("🔥 \033[1;31mURGENT (%d)\033[0m     📅 \033[1;33mTODAY (%d)\033[0m     ⏰ \033[1;31mOVERDUE (%d)\033[0m     📋 \033[90mOTHER (%d)\033[0m\n",
		len(stats.Urgent), len(stats.Today), len(stats.Overdue), len(stats.Other))
	fmt.Printf("\033[90m" + strings.Repeat("─", 65) + "\033[0m\n\n")
	
	// Show status awareness
	if len(stats.QuickWins) > 0 {
		fmt.Printf("💡 Quick wins available (%d tasks <30min)\n", len(stats.QuickWins))
	}
	if len(stats.EnergyNeeded) > 0 {
		fmt.Printf("⚡ Energy needed (%d complex tasks requiring focus)\n", len(stats.EnergyNeeded))
	}
	if len(stats.Blocked) > 0 {
		fmt.Printf("🤝 Waiting on others (%d blocked tasks)\n", len(stats.Blocked))
	}
	if len(stats.QuickWins) > 0 || len(stats.EnergyNeeded) > 0 || len(stats.Blocked) > 0 {
		fmt.Println()
	}
	
	// Show top 5 most critical tasks
	criticalTasks := []TaskInfo{}
	criticalTasks = append(criticalTasks, stats.Urgent...)
	criticalTasks = append(criticalTasks, stats.Overdue...)
	criticalTasks = append(criticalTasks, stats.Today...)
	
	if len(criticalTasks) > 5 {
		criticalTasks = criticalTasks[:5]
	}
	
	if len(criticalTasks) > 0 {
		fmt.Printf("\033[1mTop Critical Tasks:\033[0m\n")
		for i, task := range criticalTasks {
			priority := s.detectPriority(task.Text)
			priorityColor := s.getPriorityColor(priority)
			
			relPath, _ := filepath.Rel(s.config.BaseDir, task.FilePath)
			estimate := estimateTaskEffort(task.Text)
			
			taskDisplay := task.Text
			if len(taskDisplay) > 50 {
				taskDisplay = taskDisplay[:47] + "..."
			}
			
			dueDateStr := ""
			if task.DueDate != nil {
				relativeTime := formatRelativeTime(task.DueDate)
				dueDateStr = fmt.Sprintf(" \033[90m(%s)\033[0m", relativeTime)
			}
			
			fmt.Printf("[%d] %s%s\033[0m %s%s \033[90m~%s • %s:L%d\033[0m\n",
				i+1, priorityColor, priority, taskDisplay, dueDateStr, estimate, relPath, task.Line)
		}
	}
	
	fmt.Printf("\n\033[90m" + strings.Repeat("─", 50) + "\033[0m\n")
	fmt.Printf("\033[1mTotal: %d task%s\033[0m\n", len(tasks), pluralize(len(tasks)))
	fmt.Printf("\033[90mUse --full to see detailed view\033[0m\n")
	
	return nil
}

func (s *Service) showInformationScent(tasks []TaskInfo) {
	stats := analyzeTaskStats(tasks, s)
	
	if len(stats.Urgent) == 0 && len(stats.Today) == 0 && len(stats.Overdue) == 0 {
		return
	}
	
	fmt.Printf("🔥 \033[1;31mURGENT (%d)\033[0m     📅 \033[1;33mTODAY (%d)\033[0m     ⏰ \033[1;31mOVERDUE (%d)\033[0m     📋 \033[90mOTHER (%d)\033[0m\n",
		len(stats.Urgent), len(stats.Today), len(stats.Overdue), len(stats.Other))
	fmt.Printf("\033[90m" + strings.Repeat("─", 65) + "\033[0m\n\n")
}

// HandleTimeCommand processes time tracking commands
func (s *Service) HandleTimeCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("time command requires a subcommand")
	}
	
	command := args[0]
	commandArgs := args[1:]
	
	switch command {
	case "start":
		if len(commandArgs) == 0 {
			return fmt.Errorf("start command requires a task description")
		}
		taskText := strings.Join(commandArgs, " ")
		return s.startTimer(taskText)
	case "pause":
		return s.pauseTimer()
	case "resume":
		var taskText string
		if len(commandArgs) > 0 {
			taskText = strings.Join(commandArgs, " ")
		}
		return s.resumeTimer(taskText)
	case "stop":
		return s.stopTimer()
	case "status":
		return s.showTimerStatus()
	case "report":
		period := "today"
		if len(commandArgs) > 0 {
			period = commandArgs[0]
		}
		return s.showTimeReport(period)
	default:
		return fmt.Errorf("unknown time command: %s", command)
	}
}


func (s *Service) startTimer(taskText string) error {
	// First, stop any existing timer
	if err := s.stopTimer(); err != nil {
		// Continue even if stop fails (no active timer)
	}
	
	// Find the task in markdown files
	task, err := s.findTaskByText(taskText)
	if err != nil {
		return fmt.Errorf("could not find task: %w", err)
	}
	
	// Save timer state
	state := TimerState{
		IsActive:   true,
		TaskText:   task.Text,
		FilePath:   task.FilePath,
		TaskLine:   task.Line,
		StartTime:  time.Now(),
		IsPaused:   false,
	}
	
	if err := s.saveTimerState(state); err != nil {
		return fmt.Errorf("failed to save timer state: %w", err)
	}
	
	relPath, _ := filepath.Rel(s.config.BaseDir, task.FilePath)
	fmt.Printf("⏰ Started timer for: \033[1m%s\033[0m\n", task.Text)
	fmt.Printf("\033[90mLocation: %s:L%d\033[0m\n", relPath, task.Line)
	
	return nil
}

func (s *Service) pauseTimer() error {
	state, err := s.loadTimerState()
	if err != nil {
		return fmt.Errorf("no active timer found")
	}
	
	if !state.IsActive {
		return fmt.Errorf("no active timer found")
	}
	
	if state.IsPaused {
		return fmt.Errorf("timer is already paused")
	}
	
	state.IsPaused = true
	state.PausedTime = time.Now()
	
	if err := s.saveTimerState(state); err != nil {
		return fmt.Errorf("failed to save timer state: %w", err)
	}
	
	elapsed := time.Since(state.StartTime) - state.TotalPaused
	fmt.Printf("⏸️  Paused timer for: \033[1m%s\033[0m\n", state.TaskText)
	fmt.Printf("\033[90mElapsed time: %s\033[0m\n", formatDuration(elapsed))
	
	return nil
}

func (s *Service) resumeTimer(taskText string) error {
	state, err := s.loadTimerState()
	if err != nil || !state.IsActive {
		// No existing timer, start a new one
		if taskText == "" {
			return fmt.Errorf("no paused timer found and no task specified")
		}
		return s.startTimer(taskText)
	}
	
	if !state.IsPaused {
		return fmt.Errorf("timer is not paused")
	}
	
	// Add paused duration to total
	state.TotalPaused += time.Since(state.PausedTime)
	state.IsPaused = false
	state.PausedTime = time.Time{}
	
	if err := s.saveTimerState(state); err != nil {
		return fmt.Errorf("failed to save timer state: %w", err)
	}
	
	fmt.Printf("▶️  Resumed timer for: \033[1m%s\033[0m\n", state.TaskText)
	
	return nil
}

func (s *Service) stopTimer() error {
	state, err := s.loadTimerState()
	if err != nil || !state.IsActive {
		return fmt.Errorf("no active timer found")
	}
	
	// Calculate total elapsed time
	elapsed := time.Since(state.StartTime) - state.TotalPaused
	if state.IsPaused {
		elapsed -= time.Since(state.PausedTime)
	}
	
	// Add time entry to the task
	if err := s.addTimeEntry(state, elapsed); err != nil {
		return fmt.Errorf("failed to add time entry: %w", err)
	}
	
	// Clear timer state
	if err := s.clearTimerState(); err != nil {
		return fmt.Errorf("failed to clear timer state: %w", err)
	}
	
	fmt.Printf("⏹️  Stopped timer for: \033[1m%s\033[0m\n", state.TaskText)
	fmt.Printf("\033[32mTime logged: %s\033[0m\n", formatDuration(elapsed))
	
	return nil
}

func (s *Service) showTimerStatus() error {
	state, err := s.loadTimerState()
	if err != nil || !state.IsActive {
		fmt.Printf("\033[90mNo active timer\033[0m\n")
		return nil
	}
	
	elapsed := time.Since(state.StartTime) - state.TotalPaused
	if state.IsPaused {
		elapsed -= time.Since(state.PausedTime)
	}
	
	status := "🕐 RUNNING"
	if state.IsPaused {
		status = "⏸️  PAUSED"
	}
	
	relPath, _ := filepath.Rel(s.config.BaseDir, state.FilePath)
	
	fmt.Printf("%s: \033[1m%s\033[0m\n", status, state.TaskText)
	fmt.Printf("\033[90mElapsed: %s • Location: %s:L%d\033[0m\n", 
		formatDuration(elapsed), relPath, state.TaskLine)
	
	return nil
}

func (s *Service) showTimeReport(period string) error {
	report, err := s.collectTimeData(period)
	if err != nil {
		return err
	}
	
	s.formatTimeReport(report)
	return nil
}

// ShowStatus displays a summary of changed notes and todos
func (s *Service) ShowStatus() error {
	fmt.Printf("\033[1;36m📊 Notes Status\033[0m\n")
	fmt.Printf("\033[90m" + strings.Repeat("─", 50) + "\033[0m\n\n")
	
	// Get git status for changed files
	changedFiles, err := s.getChangedFiles()
	if err != nil {
		fmt.Printf("\033[90mCould not check git status: %v\033[0m\n", err)
	} else {
		s.showChangedNotes(changedFiles)
	}
	
	// Show changed todos
	s.showChangedTodos(changedFiles)
	
	return nil
}

// getChangedFiles returns files that have been modified according to git
func (s *Service) getChangedFiles() (map[string]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = s.config.BaseDir
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	changedFiles := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		
		status := strings.TrimSpace(line[:2])
		filepath := strings.TrimSpace(line[3:])
		
		if strings.HasSuffix(filepath, ".md") || strings.HasSuffix(filepath, ".txt") {
			changedFiles[filepath] = status
		}
	}
	
	return changedFiles, nil
}

// showChangedNotes displays the changed notes summary  
func (s *Service) showChangedNotes(changedFiles map[string]string) {
	if len(changedFiles) == 0 {
		fmt.Printf("\033[1;32m✅ No changed notes\033[0m\n")
		fmt.Printf("\033[90mAll notes are up to date\033[0m\n\n")
		return
	}
	
	fmt.Printf("\033[1mChanged Notes (%d):\033[0m\n", len(changedFiles))
	
	// Group by status
	modified := []string{}
	added := []string{}
	untracked := []string{}
	
	for filepath, status := range changedFiles {
		switch {
		case strings.Contains(status, "M"):
			modified = append(modified, filepath)
		case strings.Contains(status, "A"):
			added = append(added, filepath)
		case strings.Contains(status, "?"):
			untracked = append(untracked, filepath)
		}
	}
	
	// Sort each category
	sort.Strings(modified)
	sort.Strings(added)
	sort.Strings(untracked)
	
	// Display modified files
	if len(modified) > 0 {
		fmt.Printf("  \033[33mModified:\033[0m\n")
		for _, file := range modified {
			summary := s.getNoteSummary(file)
			fmt.Printf("    📝 %s%s\n", file, summary)
		}
	}
	
	// Display added files
	if len(added) > 0 {
		fmt.Printf("  \033[32mAdded:\033[0m\n")
		for _, file := range added {
			summary := s.getNoteSummary(file)
			fmt.Printf("    📄 %s%s\n", file, summary)
		}
	}
	
	// Display untracked files  
	if len(untracked) > 0 {
		fmt.Printf("  \033[36mUntracked:\033[0m\n")
		for _, file := range untracked {
			summary := s.getNoteSummary(file)
			fmt.Printf("    📋 %s%s\n", file, summary)
		}
	}
	
	fmt.Println()
}

// getNoteSummary returns a brief summary of what's in a note file
func (s *Service) getNoteSummary(relativeFilePath string) string {
	fullPath := filepath.Join(s.config.BaseDir, relativeFilePath)
	
	file, err := os.Open(fullPath)
	if err != nil {
		return ""
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineCount := 0
	taskCount := 0
	var title string
	
	taskPattern := regexp.MustCompile(`^(\s*)-\s*\[\s*[\sx]\s*\]\s*(.*)$`)
	
	for scanner.Scan() && lineCount < 20 { // Only scan first 20 lines for performance
		line := scanner.Text()
		lineCount++
		
		// Get title (first non-empty line that starts with # or is substantial)
		if title == "" {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") {
				title = strings.TrimPrefix(trimmed, "# ")
				if len(title) > 30 {
					title = title[:27] + "..."
				}
			} else if len(trimmed) > 10 && !strings.HasPrefix(trimmed, "---") {
				title = trimmed
				if len(title) > 30 {
					title = title[:27] + "..."
				}
			}
		}
		
		// Count tasks
		if taskPattern.MatchString(line) {
			taskCount++
		}
	}
	
	var summary []string
	if title != "" {
		summary = append(summary, fmt.Sprintf("\"%s\"", title))
	}
	if taskCount > 0 {
		summary = append(summary, fmt.Sprintf("%d task%s", taskCount, pluralize(taskCount)))
	}
	
	if len(summary) > 0 {
		return " \033[90m(" + strings.Join(summary, ", ") + ")\033[0m"
	}
	
	return ""
}

// showChangedTodos analyzes git diffs to show actual todo changes
func (s *Service) showChangedTodos(changedFiles map[string]string) {
	if len(changedFiles) == 0 {
		return
	}
	
	fmt.Printf("\033[1mTodo Changes:\033[0m\n")
	
	newTodos := []string{}
	completedTodos := []string{}
	modifiedTodos := []string{}
	
	// Analyze each changed file for todo changes
	for filepath, status := range changedFiles {
		changes := s.analyzeTodoChanges(filepath, status)
		newTodos = append(newTodos, changes.New...)
		completedTodos = append(completedTodos, changes.Completed...)
		modifiedTodos = append(modifiedTodos, changes.Modified...)
	}
	
	// Display changes
	totalChanges := len(newTodos) + len(completedTodos) + len(modifiedTodos)
	if totalChanges == 0 {
		fmt.Printf("  \033[90mNo todo changes in modified files\033[0m\n\n")
		return
	}
	
	// Show new todos
	if len(newTodos) > 0 {
		fmt.Printf("  \033[32mNew todos (%d):\033[0m\n", len(newTodos))
		for _, todo := range newTodos {
			fmt.Printf("    + %s\n", todo)
		}
		fmt.Println()
	}
	
	// Show completed todos  
	if len(completedTodos) > 0 {
		fmt.Printf("  \033[33mCompleted todos (%d):\033[0m\n", len(completedTodos))
		for _, todo := range completedTodos {
			fmt.Printf("    ✓ %s\n", todo)
		}
		fmt.Println()
	}
	
	// Show modified todos
	if len(modifiedTodos) > 0 {
		fmt.Printf("  \033[36mModified todos (%d):\033[0m\n", len(modifiedTodos))
		for _, todo := range modifiedTodos {
			fmt.Printf("    ~ %s\n", todo)
		}
		fmt.Println()
	}
}

type TodoChanges struct {
	New       []string
	Completed []string
	Modified  []string
}

// analyzeTodoChanges examines git diff to detect todo changes
func (s *Service) analyzeTodoChanges(relativeFilePath, status string) TodoChanges {
	changes := TodoChanges{
		New:       []string{},
		Completed: []string{},
		Modified:  []string{},
	}
	
	// Skip untracked files for diff analysis
	if strings.Contains(status, "?") {
		// For new files, just extract all todos as "new"
		fullPath := filepath.Join(s.config.BaseDir, relativeFilePath)
		tasks := s.extractTasks(fullPath)
		for _, task := range tasks {
			todoText := s.formatTodoForStatus(task, relativeFilePath)
			changes.New = append(changes.New, todoText)
		}
		return changes
	}
	
	// Get git diff for the file
	cmd := exec.Command("git", "diff", "HEAD", "--", relativeFilePath)
	cmd.Dir = s.config.BaseDir
	output, err := cmd.Output()
	if err != nil {
		return changes
	}
	
	// Parse diff output for todo changes
	diffLines := strings.Split(string(output), "\n")
	
	todoPattern := regexp.MustCompile(`^[\+\-]\s*-\s*\[\s*([\sx]?)\s*\]\s*(.*)$`)
	
	for _, line := range diffLines {
		if match := todoPattern.FindStringSubmatch(line); match != nil {
			isCompleted := strings.TrimSpace(match[1]) == "x"
			todoText := strings.TrimSpace(match[2])
			changeType := string(line[0]) // '+' or '-'
			
			if changeType == "+" {
				// Added line
				if isCompleted {
					// Added a completed todo (probably moved from incomplete)
					changes.Completed = append(changes.Completed, 
						fmt.Sprintf("%s \033[90m(%s)\033[0m", todoText, relativeFilePath))
				} else {
					// Added a new incomplete todo
					changes.New = append(changes.New, 
						fmt.Sprintf("%s \033[90m(%s)\033[0m", todoText, relativeFilePath))
				}
			} else if changeType == "-" {
				// Look ahead for corresponding + line to detect modifications
				// For now, just treat as modified
				if !isCompleted {
					changes.Modified = append(changes.Modified, 
						fmt.Sprintf("%s \033[90m(%s)\033[0m", todoText, relativeFilePath))
				}
			}
		}
	}
	
	// Remove duplicates and clean up
	changes.New = s.removeDuplicates(changes.New)
	changes.Completed = s.removeDuplicates(changes.Completed)
	changes.Modified = s.removeDuplicates(changes.Modified)
	
	return changes
}

// formatTodoForStatus formats a todo for status display
func (s *Service) formatTodoForStatus(task TaskInfo, relativeFilePath string) string {
	todoText := task.Text
	if len(todoText) > 60 {
		todoText = todoText[:57] + "..."
	}
	
	return fmt.Sprintf("%s \033[90m(%s:L%d)\033[0m", todoText, relativeFilePath, task.Line)
}

// removeDuplicates removes duplicate strings from a slice
func (s *Service) removeDuplicates(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

