package notes

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *Service) createTemplateFiles() error {
	templateDir := filepath.Join(s.config.BaseDir, "templates")
	
	templateFiles := map[string]string{
		"daily.md":    "Daily note template",
		"project.md":  "Project documentation template", 
		"meeting.md":  "Meeting notes template",
		"design.md":   "Design document template",
		"learning.md": "Learning notes template",
	}
	
	for filename, description := range templateFiles {
		templateFile := filepath.Join(templateDir, filename)
		if _, err := os.Stat(templateFile); os.IsNotExist(err) {
			content := fmt.Sprintf("# %s\n\nThis is a template for %s.\n", filename, description)
			if err := os.WriteFile(templateFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create template %s: %w", filename, err)
			}
			fmt.Printf("✓ Created template: templates/%s\n", filename)
		} else {
			fmt.Printf("⚠ Template already exists: templates/%s\n", filename)
		}
	}
	
	return nil
}

func (s *Service) createReadme() error {
	readmePath := filepath.Join(s.config.BaseDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		readme := `# Notes

Organized note-taking system with templates and folder structure.

## Usage

` + "```bash" + `
notes init                          # Initialize folder structure
notes create daily                  # Create today's daily note
notes create project "Feature Name" # Create project documentation
notes create meeting "Team Standup" # Create meeting notes
notes create design "API Design"    # Create design document
notes create learning "New Topic"   # Create learning notes
notes list                          # List all notes
notes tasks                         # Show incomplete tasks
` + "```" + `

## Structure

- ` + "`daily/`" + ` - Daily notes (date-based)
- ` + "`projects/`" + ` - Project documentation
- ` + "`meetings/`" + ` - Meeting notes
- ` + "`design/`" + ` - Technical design documents
- ` + "`learning/`" + ` - Learning notes and tutorials
- ` + "`todos/`" + ` - Task management
- ` + "`templates/`" + ` - Note templates
- ` + "`archive/`" + ` - Completed/old items

## Templates

Each note type uses a structured template to maintain consistency:

- **Daily**: Tasks, notes, follow-ups
- **Project**: Overview, goals, status, actions, decisions
- **Meeting**: Agenda, discussion, decisions, action items
- **Design**: Problem statement, solutions, implementation plan
- **Learning**: Key concepts, examples, code, insights
`
		if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
			return fmt.Errorf("failed to create README: %w", err)
		}
		fmt.Printf("✓ Created README.md\n")
	} else {
		fmt.Printf("⚠ README.md already exists\n")
	}
	
	return nil
}

// formatRelativeTime converts a date to relative time display
func formatRelativeTime(date *time.Time) string {
	if date == nil {
		return ""
	}
	
	now := time.Now()
	today := now.Format("2006-01-02")
	dateStr := date.Format("2006-01-02")
	
	if dateStr == today {
		return "today"
	}
	
	duration := now.Sub(*date)
	days := int(duration.Hours() / 24)
	
	if days == 1 {
		if dateStr < today {
			return "1 day overdue"
		}
		return "tomorrow"
	}
	
	if days > 0 && dateStr < today {
		return fmt.Sprintf("%d days overdue", days)
	}
	
	if days < 0 {
		futureDays := -days
		if futureDays == 1 {
			return "tomorrow"
		}
		return fmt.Sprintf("in %d days", futureDays)
	}
	
	return date.Format("Jan 2")
}

// detectCurrentContext checks if we're in a specific project/context
func (s *Service) detectCurrentContext() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	
	// Check if we're inside the notes directory structure
	relPath, err := filepath.Rel(s.config.BaseDir, cwd)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return ""
	}
	
	// If we're in a specific subdirectory, return it as context
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) > 0 && parts[0] != "." {
		return parts[0]
	}
	
	return ""
}

// getTreeChars returns appropriate tree drawing characters
func getTreeChars(isLast bool) string {
	if isLast {
		return "└─ "
	}
	return "├─ "
}

// TaskStats holds summary statistics about tasks
type TaskStats struct {
	Total      int
	Urgent     []TaskInfo
	Today      []TaskInfo
	Overdue    []TaskInfo
	Other      []TaskInfo
	QuickWins  []TaskInfo
	EnergyNeeded []TaskInfo
	Blocked    []TaskInfo
}

// analyzeTaskStats categorizes tasks for summary display
func analyzeTaskStats(tasks []TaskInfo, service *Service) TaskStats {
	stats := TaskStats{
		Total:     len(tasks),
		Urgent:    []TaskInfo{},
		Today:     []TaskInfo{},
		Overdue:   []TaskInfo{},
		Other:     []TaskInfo{},
		QuickWins: []TaskInfo{},
		EnergyNeeded: []TaskInfo{},
		Blocked:   []TaskInfo{},
	}
	
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	
	for _, task := range tasks {
		priority := service.detectPriority(task.Text)
		taskLower := strings.ToLower(task.Text)
		
		// Categorize by urgency
		if priority == "🔴" {
			stats.Urgent = append(stats.Urgent, task)
		} else if task.DueDate != nil {
			taskDateStr := task.DueDate.Format("2006-01-02")
			if taskDateStr < todayStr {
				stats.Overdue = append(stats.Overdue, task)
			} else if taskDateStr == todayStr {
				stats.Today = append(stats.Today, task)
			} else {
				stats.Other = append(stats.Other, task)
			}
		} else {
			stats.Other = append(stats.Other, task)
		}
		
		// Categorize by effort/status
		if strings.Contains(taskLower, "quick") || strings.Contains(taskLower, "simple") || 
		   strings.Contains(taskLower, "easy") || strings.Contains(taskLower, "fix typo") ||
		   strings.Contains(taskLower, "update") && len(task.Text) < 30 {
			stats.QuickWins = append(stats.QuickWins, task)
		} else if strings.Contains(taskLower, "design") || strings.Contains(taskLower, "architecture") ||
		         strings.Contains(taskLower, "refactor") || strings.Contains(taskLower, "implement") {
			stats.EnergyNeeded = append(stats.EnergyNeeded, task)
		}
		
		if strings.Contains(taskLower, "blocked") || strings.Contains(taskLower, "waiting") ||
		   strings.Contains(taskLower, "pending") {
			stats.Blocked = append(stats.Blocked, task)
		}
	}
	
	return stats
}

// estimateTaskEffort returns a rough effort estimate based on task content
func estimateTaskEffort(taskText string) string {
	taskLower := strings.ToLower(taskText)
	
	quickKeywords := []string{"fix typo", "update", "change", "quick", "simple", "easy"}
	mediumKeywords := []string{"add", "create", "write", "test", "review"}
	largeKeywords := []string{"implement", "design", "refactor", "architecture", "migrate"}
	
	for _, keyword := range quickKeywords {
		if strings.Contains(taskLower, keyword) && len(taskText) < 40 {
			return "15m"
		}
	}
	
	for _, keyword := range largeKeywords {
		if strings.Contains(taskLower, keyword) {
			return "2-4h"
		}
	}
	
	for _, keyword := range mediumKeywords {
		if strings.Contains(taskLower, keyword) {
			return "1h"
		}
	}
	
	if len(taskText) > 60 {
		return "1-2h"
	}
	
	return "30m"
}

// parseTimeEntry parses a time log entry line
// Format: "• 2024-01-15 09:30-10:45 (1h15m) - Initial component setup"
func parseTimeEntry(line string) (*TimeEntry, error) {
	// Remove leading bullet and whitespace
	line = strings.TrimSpace(strings.TrimPrefix(line, "•"))
	line = strings.TrimSpace(line)
	
	// Parse the pattern: "YYYY-MM-DD HH:MM-HH:MM (duration) - description"
	parts := strings.SplitN(line, " - ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid time entry format")
	}
	
	description := parts[1]
	timePart := parts[0]
	
	// Extract duration in parentheses
	durPattern := regexp.MustCompile(`\(([^)]+)\)`)
	durMatch := durPattern.FindStringSubmatch(timePart)
	if len(durMatch) != 2 {
		return nil, fmt.Errorf("duration not found")
	}
	
	duration, err := parseDuration(durMatch[1])
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}
	
	// Remove duration from time part
	timePart = durPattern.ReplaceAllString(timePart, "")
	timePart = strings.TrimSpace(timePart)
	
	// Parse date and time range
	parts = strings.Split(timePart, " ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid date-time format")
	}
	
	dateStr := parts[0]
	timeRange := parts[1]
	
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	
	timeRangeParts := strings.Split(timeRange, "-")
	if len(timeRangeParts) != 2 {
		return nil, fmt.Errorf("invalid time range")
	}
	
	startTime, err := time.Parse("15:04", timeRangeParts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}
	
	endTime, err := time.Parse("15:04", timeRangeParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid end time: %w", err)
	}
	
	// Combine date with times
	startDateTime := time.Date(date.Year(), date.Month(), date.Day(), 
		startTime.Hour(), startTime.Minute(), 0, 0, time.Local)
	endDateTime := time.Date(date.Year(), date.Month(), date.Day(), 
		endTime.Hour(), endTime.Minute(), 0, 0, time.Local)
	
	return &TimeEntry{
		Date:        date,
		StartTime:   startDateTime,
		EndTime:     endDateTime,
		Duration:    duration,
		Description: description,
	}, nil
}

// parseDuration parses duration strings like "1h15m", "30m", "2h"
func parseDuration(s string) (time.Duration, error) {
	// Handle common formats
	s = strings.ToLower(s)
	
	// Try Go's built-in duration parser first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	
	// Handle formats like "1h15m", "45m", "2h"
	var hours, minutes int
	var err error
	
	if strings.Contains(s, "h") && strings.Contains(s, "m") {
		parts := strings.Split(s, "h")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid duration format")
		}
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		minuteStr := strings.TrimSuffix(parts[1], "m")
		minutes, err = strconv.Atoi(minuteStr)
		if err != nil {
			return 0, err
		}
	} else if strings.Contains(s, "h") {
		hourStr := strings.TrimSuffix(s, "h")
		hours, err = strconv.Atoi(hourStr)
		if err != nil {
			return 0, err
		}
	} else if strings.Contains(s, "m") {
		minuteStr := strings.TrimSuffix(s, "m")
		minutes, err = strconv.Atoi(minuteStr)
		if err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("invalid duration format")
	}
	
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0m"
	}
	
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	
	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}

// formatTimeEntry formats a time entry for markdown output
func formatTimeEntry(entry TimeEntry) string {
	dateStr := entry.Date.Format("2006-01-02")
	startStr := entry.StartTime.Format("15:04")
	endStr := entry.EndTime.Format("15:04")
	durationStr := formatDuration(entry.Duration)
	
	return fmt.Sprintf("  • %s %s-%s (%s) - %s", 
		dateStr, startStr, endStr, durationStr, entry.Description)
}

// TimerState represents the current timer state
type TimerState struct {
	IsActive    bool          `json:"is_active"`
	TaskText    string        `json:"task_text"`
	FilePath    string        `json:"file_path"`
	TaskLine    int           `json:"task_line"`
	StartTime   time.Time     `json:"start_time"`
	IsPaused    bool          `json:"is_paused"`
	PausedTime  time.Time     `json:"paused_time"`
	TotalPaused time.Duration `json:"total_paused"`
}

// Timer state file management
func (s *Service) getTimerStatePath() string {
	return filepath.Join(s.config.BaseDir, ".timer_state.json")
}

func (s *Service) saveTimerState(state TimerState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(s.getTimerStatePath(), data, 0644)
}

func (s *Service) loadTimerState() (TimerState, error) {
	var state TimerState
	data, err := os.ReadFile(s.getTimerStatePath())
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(data, &state)
	return state, err
}

func (s *Service) clearTimerState() error {
	return os.Remove(s.getTimerStatePath())
}

// findTaskByText searches for a task by partial text match
func (s *Service) findTaskByText(searchText string) (*TaskInfo, error) {
	searchDirs := []string{"daily", "projects", "meetings", "design", "learning", "todos"}
	searchLower := strings.ToLower(searchText)
	
	var matches []TaskInfo
	
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
			for _, task := range tasks {
				if strings.Contains(strings.ToLower(task.Text), searchLower) {
					matches = append(matches, task)
				}
			}
			
			return nil
		})
		
		if err != nil {
			continue
		}
	}
	
	if len(matches) == 0 {
		return nil, fmt.Errorf("no task found matching: %s", searchText)
	}
	
	if len(matches) == 1 {
		return &matches[0], nil
	}
	
	// Multiple matches - return the best one (exact match preferred)
	for _, match := range matches {
		if strings.EqualFold(match.Text, searchText) {
			return &match, nil
		}
	}
	
	// Return first match if no exact match
	return &matches[0], nil
}

// TimeReportData holds aggregated time data for reporting
type TimeReportData struct {
	Tasks []TaskTimeData
	TotalTime time.Duration
	Period string
	StartDate time.Time
	EndDate time.Time
}

type TaskTimeData struct {
	TaskInfo TaskInfo
	Entries []TimeEntry
	TotalTime time.Duration
}

// collectTimeData gathers all time entries for the specified period
func (s *Service) collectTimeData(period string) (*TimeReportData, error) {
	now := time.Now()
	var startDate, endDate time.Time
	
	switch strings.ToLower(period) {
	case "today":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = startDate.Add(24 * time.Hour)
	case "week":
		// Start of current week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 { // Sunday
			weekday = 7
		}
		startDate = now.Add(-time.Duration(weekday-1) * 24 * time.Hour)
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, now.Location())
		endDate = startDate.Add(7 * 24 * time.Hour)
	case "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0)
	default:
		return nil, fmt.Errorf("invalid period: %s. Use 'today', 'week', or 'month'", period)
	}
	
	report := &TimeReportData{
		Tasks: []TaskTimeData{},
		Period: period,
		StartDate: startDate,
		EndDate: endDate,
	}
	
	// Collect all tasks with time entries
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
			for _, task := range tasks {
				if len(task.TimeEntries) == 0 {
					continue
				}
				
				// Filter time entries for the period
				var filteredEntries []TimeEntry
				var taskTotal time.Duration
				
				for _, entry := range task.TimeEntries {
					if (entry.Date.After(startDate) || entry.Date.Equal(startDate)) && entry.Date.Before(endDate) {
						filteredEntries = append(filteredEntries, entry)
						taskTotal += entry.Duration
					}
				}
				
				if len(filteredEntries) > 0 {
					report.Tasks = append(report.Tasks, TaskTimeData{
						TaskInfo: task,
						Entries: filteredEntries,
						TotalTime: taskTotal,
					})
					report.TotalTime += taskTotal
				}
			}
			
			return nil
		})
		
		if err != nil {
			continue
		}
	}
	
	// Sort tasks by total time (descending)
	sort.Slice(report.Tasks, func(i, j int) bool {
		return report.Tasks[i].TotalTime > report.Tasks[j].TotalTime
	})
	
	return report, nil
}

// formatTimeReport formats the time report for display
func (s *Service) formatTimeReport(report *TimeReportData) {
	// Header
	periodTitle := strings.Title(report.Period)
	if report.Period == "today" {
		periodTitle = "Today"
	} else if report.Period == "week" {
		periodTitle = "This Week"
	} else if report.Period == "month" {
		periodTitle = "This Month"
	}
	
	fmt.Printf("\033[1;36m⏰ Time Report - %s\033[0m\n", periodTitle)
	fmt.Printf("\033[90m%s to %s\033[0m\n", 
		report.StartDate.Format("Jan 2"), 
		report.EndDate.Add(-24*time.Hour).Format("Jan 2, 2006"))
	fmt.Printf("\033[90m" + strings.Repeat("─", 60) + "\033[0m\n\n")
	
	if len(report.Tasks) == 0 {
		fmt.Printf("\033[90mNo time tracked for this period.\033[0m\n")
		return
	}
	
	// Summary
	fmt.Printf("\033[1mTotal Time: %s\033[0m across %d task%s\n\n", 
		formatDuration(report.TotalTime), 
		len(report.Tasks), 
		pluralize(len(report.Tasks)))
	
	// Task breakdown
	fmt.Printf("\033[1mTask Breakdown:\033[0m\n")
	for i, taskData := range report.Tasks {
		relPath, _ := filepath.Rel(s.config.BaseDir, taskData.TaskInfo.FilePath)
		
		// Task header
		taskDisplay := taskData.TaskInfo.Text
		if len(taskDisplay) > 50 {
			taskDisplay = taskDisplay[:47] + "..."
		}
		
		percentage := float64(taskData.TotalTime) / float64(report.TotalTime) * 100
		
		fmt.Printf("\n[%d] \033[1m%s\033[0m \033[90m(%s, %.1f%%)\033[0m\n", 
			i+1, taskDisplay, formatDuration(taskData.TotalTime), percentage)
		fmt.Printf("    \033[90m📄 %s:L%d\033[0m\n", relPath, taskData.TaskInfo.Line)
		
		// Show individual time entries if there are multiple sessions
		if len(taskData.Entries) > 1 {
			fmt.Printf("    \033[90mSessions:\033[0m\n")
			for _, entry := range taskData.Entries {
				fmt.Printf("    • %s %s-%s (%s) - %s\n",
					entry.Date.Format("Jan 2"),
					entry.StartTime.Format("15:04"),
					entry.EndTime.Format("15:04"),
					formatDuration(entry.Duration),
					entry.Description)
			}
		} else if len(taskData.Entries) == 1 {
			entry := taskData.Entries[0]
			fmt.Printf("    • %s %s-%s - %s\n",
				entry.Date.Format("Jan 2"),
				entry.StartTime.Format("15:04"),
				entry.EndTime.Format("15:04"),
				entry.Description)
		}
	}
	
	// Daily breakdown for week/month reports
	if report.Period == "week" || report.Period == "month" {
		fmt.Printf("\n\033[1mDaily Breakdown:\033[0m\n")
		s.showDailyBreakdown(report)
	}
	
	fmt.Printf("\n\033[90m" + strings.Repeat("─", 60) + "\033[0m\n")
	fmt.Printf("\033[90mAverage per day: %s\033[0m\n", s.calculateDailyAverage(report))
}

// showDailyBreakdown shows time per day for week/month reports
func (s *Service) showDailyBreakdown(report *TimeReportData) {
	dailyTotals := make(map[string]time.Duration)
	
	// Aggregate time by day
	for _, taskData := range report.Tasks {
		for _, entry := range taskData.Entries {
			dayKey := entry.Date.Format("2006-01-02")
			dailyTotals[dayKey] += entry.Duration
		}
	}
	
	// Create sorted list of days
	var days []string
	current := report.StartDate
	for current.Before(report.EndDate) {
		dayKey := current.Format("2006-01-02")
		days = append(days, dayKey)
		current = current.Add(24 * time.Hour)
	}
	
	// Display daily totals
	for _, day := range days {
		if total, exists := dailyTotals[day]; exists && total > 0 {
			date, _ := time.Parse("2006-01-02", day)
			fmt.Printf("  %s: %s\n", 
				date.Format("Mon Jan 2"), 
				formatDuration(total))
		}
	}
}

// calculateDailyAverage calculates average time per day for the period
func (s *Service) calculateDailyAverage(report *TimeReportData) string {
	days := int(report.EndDate.Sub(report.StartDate).Hours() / 24)
	if days == 0 {
		days = 1
	}
	
	avgDuration := time.Duration(int64(report.TotalTime) / int64(days))
	return formatDuration(avgDuration)
}

// addTimeEntry adds a time entry to a task in its markdown file
func (s *Service) addTimeEntry(state TimerState, duration time.Duration) error {
	// Read the file
	content, err := os.ReadFile(state.FilePath)
	if err != nil {
		return err
	}
	
	lines := strings.Split(string(content), "\n")
	if state.TaskLine > len(lines) {
		return fmt.Errorf("task line %d not found in file", state.TaskLine)
	}
	
	// Create time entry
	entry := TimeEntry{
		Date:        state.StartTime,
		StartTime:   state.StartTime,
		EndTime:     state.StartTime.Add(duration),
		Duration:    duration,
		Description: "Work session", // Could be made configurable
	}
	
	// Find insertion point (after the task line)
	insertLine := state.TaskLine
	
	// Look for existing "Time log:" line
	timeLogExists := false
	for i := state.TaskLine; i < len(lines) && i < state.TaskLine+10; i++ {
		if strings.Contains(lines[i], "Time log:") {
			timeLogExists = true
			// Find where to insert new entry (before "Remaining:" or "Total:" if they exist)
			insertLine = i + 1
			for j := i + 1; j < len(lines) && strings.HasPrefix(lines[j], "  "); j++ {
				if strings.Contains(lines[j], "Remaining:") || strings.Contains(lines[j], "Total:") {
					insertLine = j
					break
				}
				if strings.HasPrefix(lines[j], "  •") {
					insertLine = j + 1
				}
			}
			break
		}
		// Stop if we hit another task or non-indented content
		if !strings.HasPrefix(lines[i], "  ") && strings.TrimSpace(lines[i]) != "" && i > state.TaskLine {
			break
		}
	}
	
	// Insert time log block if it doesn't exist
	if !timeLogExists {
		newLines := make([]string, 0, len(lines)+3)
		newLines = append(newLines, lines[:insertLine]...)
		newLines = append(newLines, "  Time log:")
		newLines = append(newLines, formatTimeEntry(entry))
		newLines = append(newLines, lines[insertLine:]...)
		lines = newLines
	} else {
		// Insert just the time entry
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:insertLine]...)
		newLines = append(newLines, formatTimeEntry(entry))
		newLines = append(newLines, lines[insertLine:]...)
		lines = newLines
	}
	
	// Write back to file
	newContent := strings.Join(lines, "\n")
	return os.WriteFile(state.FilePath, []byte(newContent), 0644)
}