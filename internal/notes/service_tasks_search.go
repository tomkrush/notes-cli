package notes

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"notes/internal/config"
)

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

type TaskSearchService struct {
	config *config.Config
}

func NewTaskSearchService(cfg *config.Config) *TaskSearchService {
	return &TaskSearchService{config: cfg}
}

func (s *TaskSearchService) ShowTasks() error {
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

func (s *TaskSearchService) Search(query string, searchTags []string) error {
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

func (s *TaskSearchService) extractTasks(filePath string) []TaskInfo {
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

func (s *TaskSearchService) searchInFile(filePath, query string, searchTags []string) []SearchResult {
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

func (s *TaskSearchService) detectPriority(taskText string) string {
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

func (s *TaskSearchService) getPriorityColor(priority string) string {
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