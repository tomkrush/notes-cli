package notes

import (
	"fmt"
	"os"
	"path/filepath"
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