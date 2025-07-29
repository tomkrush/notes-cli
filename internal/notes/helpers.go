package notes

import (
	"fmt"
	"os"
	"path/filepath"
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