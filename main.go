package main

import (
	"fmt"
	"os"
	"strings"

	"notes/internal/config"
	"notes/internal/notes"
)

func main() {
	cfg := config.New()
	service := notes.NewService(cfg)

	if len(os.Args) < 2 {
		showHelp()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		if err := service.Initialize(); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing notes: %v\n", err)
			os.Exit(1)
		}
	case "create":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Error: create command requires a note type\n")
			showCreateHelp()
			os.Exit(1)
		}
		noteType := notes.NoteType(args[0])
		var title string
		if len(args) > 1 {
			title = args[1]
		}
		if err := service.Create(noteType, title); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating note: %v\n", err)
			os.Exit(1)
		}
	case "list":
		if err := service.List(); err != nil {
			fmt.Fprintf(os.Stderr, "Error listing notes: %v\n", err)
			os.Exit(1)
		}
	case "tasks":
		filters := parseTaskFilters(args)
		if err := service.ShowTasks(filters); err != nil {
			fmt.Fprintf(os.Stderr, "Error showing tasks: %v\n", err)
			os.Exit(1)
		}
	case "save":
		var message string
		if len(args) > 0 {
			message = strings.Join(args, " ")
		} else {
			message = "Update notes"
		}
		if err := service.SaveChanges(message); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving changes: %v\n", err)
			os.Exit(1)
		}
	case "search":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Error: search command requires a query\n")
			showSearchHelp()
			os.Exit(1)
		}
		
		query := ""
		var tags []string
		
		for _, arg := range args {
			if strings.HasPrefix(arg, "#") {
				tags = append(tags, arg)
			} else if query == "" {
				query = arg
			} else {
				query += " " + arg
			}
		}
		
		if err := service.Search(query, tags); err != nil {
			fmt.Fprintf(os.Stderr, "Error searching notes: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		showHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println(`notes - Organized note-taking with templates

Usage:
  notes init                          # Initialize folder structure
  notes create <type> [title]         # Create a new note
  notes list                          # List existing notes
  notes tasks [filters]               # Show incomplete tasks with due dates
  notes search <query> [#tag ...]     # Search notes by content and tags
  notes save [message]                # Commit all changes to git
  notes help                          # Show this help

Available note types:
  daily    - Daily notes (auto-dated)
  project  - Project documentation
  meeting  - Meeting notes
  design   - Technical design documents
  learning - Learning notes and tutorials

Task Features:
  - Add due dates: "- [ ] Task due:2024-12-31"
  - Add tags: "- [ ] Task #urgent #work"
  - Priority keywords: urgent, critical, important (!!!), !!, soon

Task Filters:
  --tag <tag>       Filter by tag (e.g., --tag urgent)
  --priority <pri>  Filter by priority (high, medium, low)
  --overdue         Show only overdue tasks
  --today           Show only tasks due today
  --file <pattern>  Filter by file pattern (e.g., --file daily/)
  --sort <method>   Sort by priority, due, or file

Examples:
  notes init
  notes create daily
  notes create project "My New Project"
  notes create meeting "Team Standup"
  notes search "API design" #work
  notes search "" #urgent
  notes tasks
  notes tasks --tag urgent --overdue
  notes tasks --priority high --sort priority`)
}

func showCreateHelp() {
	fmt.Println(`Usage: notes create <type> [title]

Available types:
  daily    - Daily notes (auto-dated)
  project  - Project documentation  
  meeting  - Meeting notes
  design   - Technical design documents
  learning - Learning notes and tutorials

Examples:
  notes create daily
  notes create project "My New Project"
  notes create meeting "Team Standup"`)
}

func showSearchHelp() {
	fmt.Println(`Usage: notes search <query> [#tag ...]

Search for notes by content and/or tags:
  - Search by text content in any note
  - Filter by tags using #tagname
  - Combine text search with tag filtering

Examples:
  notes search "API design"          # Search for "API design" text
  notes search "" #work              # Find all notes with #work tag
  notes search "meeting" #urgent     # Find "meeting" text with #urgent tag
  notes search #project #active      # Find notes with both tags`)
}

func parseTaskFilters(args []string) notes.TaskFilters {
	filters := notes.TaskFilters{}
	
	for i := 0; i < len(args); i++ {
		arg := args[i]
		
		switch arg {
		case "--tag":
			if i+1 < len(args) {
				i++
				tag := args[i]
				if !strings.HasPrefix(tag, "#") {
					tag = "#" + tag
				}
				filters.Tags = append(filters.Tags, tag)
			}
		case "--priority":
			if i+1 < len(args) {
				i++
				filters.Priority = args[i]
			}
		case "--overdue":
			filters.Overdue = true
		case "--today":
			filters.Today = true
		case "--file":
			if i+1 < len(args) {
				i++
				filters.FilePattern = args[i]
			}
		case "--sort":
			if i+1 < len(args) {
				i++
				filters.SortBy = args[i]
			}
		}
	}
	
	return filters
}
