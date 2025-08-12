package main

import (
	"fmt"
	"os"
	"strconv"
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
	case "status":
		if err := service.ShowStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Error showing status: %v\n", err)
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
	case "time":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Error: time command requires a subcommand\n")
			showTimeHelp()
			os.Exit(1)
		}
		if err := service.HandleTimeCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error with time command: %v\n", err)
			os.Exit(1)
		}
	case "preview":
		port := 8080
		if len(args) > 0 {
			if p, err := strconv.Atoi(args[0]); err == nil {
				port = p
			}
		}
		if err := service.StartPreview(port); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting preview server: %v\n", err)
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
		if len(args) > 0 {
			showCommandHelp(args[0])
		} else {
			showHelp()
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println(`notes - Organized note-taking with enhanced markdown tasks

QUICK START
  notes init                    # Set up your notes folder
  notes create daily            # Create today's daily note
  notes tasks                   # View your tasks
  notes time start "task text"  # Start tracking time on a task

COMMANDS
  init                         Initialize folder structure
  create <type> [title]        Create a new note
  list                         List existing notes  
  tasks [options]              Show tasks with filters
  status                       Show changed notes and todos
  time <command>               Time tracking (start/stop/status)
  search <query> [#tags]       Search notes by content/tags
  preview [port]               Start markdown preview server (default: 8080)
  save [message]               Commit changes to git

GETTING STARTED
  1. Run 'notes init' to set up folders
  2. Create your first note: 'notes create daily'
  3. Add tasks in markdown: - [ ] My first task est:1h #work
  4. View tasks: 'notes tasks'
  5. Track time: 'notes time start "My first"'

HELP TOPICS
  notes help create            # Note types and creation
  notes help tasks             # Task views and filters
  notes help time              # Time tracking system
  notes help markdown          # Enhanced markdown syntax
  notes help search            # Search and filtering

TIP: All files are standard markdown - learn basics at:
     https://www.markdownguide.org/basic-syntax/`)
}

func showCommandHelp(command string) {
	switch command {
	case "create":
		showCreateHelp()
	case "tasks":
		showTasksHelp()
	case "time":
		showTimeHelp()
	case "search":
		showSearchHelp()
	case "markdown":
		showMarkdownHelp()
	case "preview":
		showPreviewHelp()
	default:
		fmt.Printf("No detailed help available for '%s'\n", command)
		fmt.Println("Available help topics: create, tasks, time, search, markdown, preview")
	}
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

func showTasksHelp() {
	fmt.Println(`notes tasks - Enhanced task viewing and filtering

BASIC VIEWS
  notes tasks                  # Smart context-aware view
  notes tasks --summary        # Overview: counts + top 5 critical
  notes tasks --focus          # Overdue + today's tasks only
  notes tasks --full           # Detailed view of all tasks
  notes tasks --all            # Override defaults, show everything

FILTERS
  --tag <tag>       Filter by tag (--tag urgent)
  --priority <pri>  Filter by priority (high, medium, low)  
  --overdue         Show only overdue tasks
  --today           Show only tasks due today
  --file <pattern>  Filter by file pattern (--file daily/)
  --sort <method>   Sort by priority, due, or file

EXAMPLES
  notes tasks --summary                    # Quick overview
  notes tasks --tag urgent --overdue      # Urgent overdue tasks
  notes tasks --file daily/ --today       # Today's daily tasks
  notes tasks --priority high --sort due  # High priority by due date

TASK DISPLAY
  Tasks show time tracking progress and estimates:
  ├─ 🔴 Fix auth bug [1h30m/2h] ~2h (L45)    # Progress vs estimate
  ├─ 🟡 Add tests [45m worked] ~1h (L67)     # Time worked so far  
  └─ ⚪ Update docs [2h completed] ~1h30m     # Over estimate, done`)
}

func showMarkdownHelp() {
	fmt.Println(`Enhanced Markdown Tasks - Standard markdown with special powers

BASIC TASKS
  Standard markdown tasks work normally:
  - [ ] Basic task
  - [x] Completed task

ENHANCED SYNTAX
  Add due dates, estimates, tags, and priority:

  Due dates:     - [ ] Task due:2024-12-31
  Estimates:     - [ ] Task est:2h #urgent  
  Tags:          - [ ] Task #urgent #work #backend
  Priority:      - [ ] URGENT task !!! (keywords: urgent, critical, important)

TIME TRACKING INTEGRATION
  When you track time, structured logs are automatically added:
  
  - [ ] Fix auth bug est:2h #backend
    Time log:
    • 2024-01-15 09:30-10:45 (1h15m) - Work session
    • 2024-01-15 14:00-15:30 (1h30m) - Testing fixes
    Remaining: ~15m

PRIORITY KEYWORDS
  Use these words in task text for automatic priority detection:
  - urgent, critical, important = High priority
  - Use !!! for emphasis

TAG SYSTEM
  Tags help organize and filter tasks:
  - Use #tagname anywhere in task text
  - Multiple tags: #work #urgent #backend
  - Filter tasks: notes tasks --tag urgent`)
}

func showTimeHelp() {
	fmt.Println(`notes time - Time tracking for markdown tasks

COMMANDS
  start <task>     Find task by partial text match and start timer
  pause            Pause current active timer  
  resume           Resume paused timer
  stop             Stop timer and log time to markdown
  status           Show current timer status
  report [period]  Show time report (today, week, month)

WORKFLOW
  1. Create task in markdown: - [ ] Fix auth bug est:2h #urgent
  2. Start timer: notes time start "Fix auth"
  3. Work on task (timer persists across restarts)
  4. Stop timer: notes time stop
  5. Time is automatically logged to your markdown file

TIME LOGS
  Time tracking adds structured logs to your tasks:
  
  - [ ] Fix auth bug est:2h #urgent
    Time log:
    • 2024-01-15 09:30-10:45 (1h15m) - Work session
    • 2024-01-15 14:00-15:30 (1h30m) - Testing fixes
    Remaining: ~15m

REPORTS
  notes time report         # Today (default)
  notes time report week    # This week's summary
  notes time report month   # This month's summary

EXAMPLES  
  notes time start "Fix login bug"
  notes time status
  notes time pause
  notes time resume
  notes time stop
  notes time report today`)
}

func showPreviewHelp() {
	fmt.Println(`notes preview - Markdown preview server with Mermaid diagrams

USAGE
  notes preview [port]         # Start server on specified port (default: 8080)

FEATURES
  - Live markdown rendering with GitHub-flavored styling
  - Automatic Mermaid diagram rendering from code blocks
  - File browser to navigate all your notes
  - Responsive design for mobile viewing
  - Task list rendering with checkboxes

MERMAID DIAGRAMS
  Create diagrams using standard mermaid syntax in code blocks:
  
  graph TD
      A[Start] --> B{Decision}
      B -->|Yes| C[Action 1]
      B -->|No| D[Action 2]
  
  sequenceDiagram
      User->>Server: Request
      Server->>Database: Query
      Database-->>Server: Response
      Server-->>User: Response

EXAMPLES
  notes preview                # Start server on port 8080
  notes preview 3000           # Start server on port 3000
  
The server will display a file browser at the root and render any .md file
when clicked. The preview updates automatically when you save changes to files.`)
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
		case "--focus":
			filters.Focus = true
		case "--all":
			filters.All = true
		case "--summary":
			filters.Summary = true
		case "--full":
			filters.Full = true
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
