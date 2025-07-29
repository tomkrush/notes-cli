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
	fmt.Println(`notes - Organized note-taking with enhanced markdown tasks

OVERVIEW
  This tool manages markdown files with enhanced task tracking. You write
  standard markdown with regular syntax, but tasks get special powers.

BASIC USAGE
  notes init                          # Initialize folder structure
  notes create <type> [title]         # Create a new note  
  notes list                          # List existing notes
  notes tasks [filters]               # Show enhanced task views
  notes status                        # Show changed notes and todos
  notes time <command> [args]         # Time tracking for tasks
  notes search <query> [#tag ...]     # Search notes by content and tags
  notes save [message]                # Commit all changes to git

MARKDOWN SUPPORT
  All files are standard markdown. Learn markdown basics at:
  https://www.markdownguide.org/basic-syntax/
  
  Standard markdown works normally:
  # Headings, **bold**, *italic*, [links](url), etc.

ENHANCED TASKS
  Standard markdown tasks work: - [ ] Basic task
  But this tool adds powerful enhancements when you use special syntax:

  Due dates:     - [ ] Task due:2024-12-31
  Estimates:     - [ ] Task est:2h #urgent  
  Tags:          - [ ] Task #urgent #work
  Priority:      - [ ] URGENT task !!! (keywords: urgent, critical, important)
  
  When you track time, the tool automatically adds structured logs:
  - [ ] Fix auth bug est:2h #backend
    Time log:
    • 2024-01-15 09:30-10:45 (1h15m) - Work session
    • 2024-01-15 14:00-15:30 (1h30m) - Testing fixes
    Remaining: ~15m

NOTE TYPES (Templates)
  daily    - Daily notes (auto-dated)
  project  - Project documentation  
  meeting  - Meeting notes
  design   - Technical design documents
  learning - Learning notes and tutorials

TASK VIEWS (CLI Commands)
  notes tasks                    # Smart context-aware view
  notes tasks --summary          # Overview: counts + top 5 critical
  notes tasks --focus            # Overdue + today's tasks only
  notes tasks --full             # Detailed view of all tasks
  notes tasks --all              # Override defaults, show everything

TASK FILTERS (CLI Commands)
  --tag <tag>       Filter by tag (--tag urgent)
  --priority <pri>  Filter by priority (high, medium, low)  
  --overdue         Show only overdue tasks
  --today           Show only tasks due today
  --file <pattern>  Filter by file pattern (--file daily/)
  --sort <method>   Sort by priority, due, or file

TIME TRACKING (CLI Commands)
  notes time start <task>    Find task and start timer
  notes time pause           Pause current timer
  notes time resume          Resume paused timer  
  notes time stop            Stop timer, auto-log to markdown
  notes time status          Show current timer status
  notes time report [period] Show time report (today, week, month)

EXAMPLES
  # Create and manage notes
  notes init
  notes create daily
  notes create project "Mobile App Redesign"
  
  # View tasks with different perspectives
  notes tasks                        # Smart defaults
  notes tasks --summary              # Quick overview
  notes tasks --tag urgent --overdue # Filtered view
  
  # Check what's changed
  notes status                       # See changed notes and todos
  
  # Track time on tasks  
  notes time start "Fix login bug"   # Start timer
  notes time status                  # Check progress
  notes time stop                    # Stop and log
  notes time report today            # See what you worked on today
  notes time report week             # Weekly time summary
  notes time report month            # Monthly time summary
  
  # Search and save
  notes search "API" #backend
  notes save "Updated project tasks"

For detailed time tracking help: notes time help`)
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

func showTimeHelp() {
	fmt.Println(`notes time - Time tracking for markdown tasks

OVERVIEW
  Track time on tasks using CLI commands. Time logs are automatically
  written to your markdown files in a structured, human-readable format.

CLI COMMANDS
  start <task>     Find task by partial text match and start timer
  pause            Pause current active timer  
  resume [task]    Resume paused timer or start new one
  stop             Stop timer and automatically log time to markdown
  status           Show current timer status
  report [period]  Show time report (today, week, month)

HOW IT WORKS
  1. You have tasks in markdown: - [ ] Fix authentication bug est:2h
  2. Start timer: notes time start "Fix auth"
  3. Tool finds task and starts timing
  4. When you stop, it automatically adds structured time logs:

  - [ ] Fix authentication bug est:2h #urgent
    Time log:
    • 2024-01-15 09:30-10:45 (1h15m) - Work session
    • 2024-01-15 14:00-15:30 (1h30m) - Testing fixes
    Remaining: ~15m

SMART FEATURES  
  • Task finding: Partial text search finds tasks automatically
  • Persistent timers: Timers survive app restarts and reboots
  • Progress tracking: Shows worked time vs estimates in task views
  • Clean integration: Time logs don't clutter your markdown

WORKFLOW EXAMPLES
  # Start working on a task
  $ notes time start "Fix login"
  ⏰ Started timer for: Fix login validation
  Location: projects/auth.md:L23
  
  # Check what you're working on  
  $ notes time status
  🕐 RUNNING: Fix login validation
  Elapsed: 1h23m • Location: projects/auth.md:L23
  
  # Take a break
  $ notes time pause
  ⏸️ Paused timer for: Fix login validation
  Elapsed time: 1h23m
  
  # Get back to work
  $ notes time resume  
  ▶️ Resumed timer for: Fix login validation
  
  # Finish and log time
  $ notes time stop
  ⏹️ Stopped timer for: Fix login validation  
  Time logged: 1h45m

TIME REPORTS
  Generate detailed reports of your time tracking activity:
  
  notes time report              # Today's activity (default)
  notes time report today        # Today's time breakdown
  notes time report week         # This week's summary
  notes time report month        # This month's summary
  
  Reports include:
  • Total time worked and task count
  • Task breakdown with percentages  
  • Individual work sessions with times
  • Daily breakdown (for week/month)
  • Average time per day
  • File locations for each task

REPORT EXAMPLES
  $ notes time report today
  ⏰ Time Report - Today
  Jul 29 to Jul 29, 2025
  ─────────────────────────────────────
  
  Total Time: 4h30m across 3 tasks
  
  Task Breakdown:
  [1] Fix authentication bug (2h15m, 50.0%)
      📄 projects/auth.md:L45
      Sessions:
      • Jul 29 09:00-10:15 (1h15m) - Initial debugging
      • Jul 29 14:00-15:00 (1h) - Testing fixes
  
  [2] Update documentation (1h30m, 33.3%)  
      📄 projects/docs.md:L12
      • Jul 29 11:00-12:30 - Writing guide
  
  $ notes time report week
  ⏰ Time Report - This Week  
  Jul 28 to Aug 3, 2025
  ─────────────────────────────────────
  
  Total Time: 12h45m across 8 tasks
  
  Daily Breakdown:
    Mon Jul 28: 3h15m
    Tue Jul 29: 4h30m
    Wed Jul 30: 2h45m
    Thu Jul 31: 2h15m
  
  Average per day: 3h11m

INTEGRATION WITH TASK VIEWS
  Time tracking integrates with all task views:
  
  notes tasks --summary    # Shows time progress in overview
  notes tasks --focus      # Time info on urgent tasks
  
  Tasks display time info:
  ├─ 🔴 Fix auth bug [1h30m/2h] ~2h (L45)    # Progress vs estimate
  ├─ 🟡 Add tests [45m worked] ~1h (L67)     # Time worked so far  
  └─ ⚪ Update docs [2h completed] ~1h30m     # Over estimate, done

The goal: seamless time tracking that enhances your markdown workflow.`)
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
