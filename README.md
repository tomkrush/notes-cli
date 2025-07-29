# notes

A CLI tool for organized note-taking with templates and task management. Designed for developers who prefer markdown files and git-based workflows over heavy note-taking apps.

## Features

- Template-based note creation for different contexts
- Task extraction and due date tracking across all notes
- Advanced task filtering and smart views (summary, focus modes)
- **Time tracking with structured markdown logs**
- Full-text search with tag filtering
- Git integration for version control
- Lightweight and fast - just markdown files

## Usage

```bash
notes init                          # Initialize folder structure
notes create <type> [title]         # Create a new note
notes list                          # List existing notes
notes tasks [filters]               # Show incomplete tasks with smart views
notes time <command> [args]         # Time tracking for tasks
notes search <query> [#tag ...]     # Search notes by content and tags
notes save [message]                # Commit all changes to git
```

## Note Types

- `daily` - Daily notes (auto-dated)
- `project` - Project documentation
- `meeting` - Meeting notes
- `design` - Technical design documents
- `learning` - Learning notes and tutorials

## Task Management

### Task Syntax
- Basic task: `- [ ] Task description`
- Due date: `- [ ] Complete project due:2024-12-31`
- Estimate: `- [ ] Fix bug est:2h #urgent`
- Tags: `- [ ] Important task #urgent #work`
- Priority: Use keywords like `urgent`, `critical`, `important`, `!!!`, `!!`, or `soon`

### Smart Task Views
Tasks feature intelligent defaults and multiple view modes:

```bash
notes tasks                         # Smart defaults (context-aware or focus mode)
notes tasks --summary              # Overview with counts and top 5 critical tasks
notes tasks --focus                # Show only overdue + today's tasks
notes tasks --full                 # Detailed view of all tasks
notes tasks --all                  # Override smart defaults, show everything
```

### Task Filtering
Combine filters for precise task lists:

```bash
notes tasks --tag urgent            # Filter by tag
notes tasks --priority high         # Filter by priority (high/medium/low)
notes tasks --overdue              # Show only overdue tasks
notes tasks --today                # Show only tasks due today
notes tasks --file daily/          # Filter by file pattern
notes tasks --sort priority        # Sort by priority, due, or file
```

### Examples
```bash
# Show all urgent tasks that are overdue
notes tasks --tag urgent --overdue

# Show high priority tasks sorted by priority
notes tasks --priority high --sort priority

# Show all tasks due today in daily notes
notes tasks --file daily/ --today

# Show work-related medium priority tasks
notes tasks --tag work --priority medium
```

## Time Tracking

Track time spent on tasks with structured markdown logs that remain human-readable.

### Time Commands

```bash
notes time start "Fix authentication bug"     # Start timing a task
notes time pause                              # Pause current timer
notes time resume                             # Resume paused timer
notes time stop                               # Stop timer and save entry
notes time status                             # Show current timer status
notes time report [period]                    # Time reports (coming soon)
```

### Time Log Format

When you work on tasks, time tracking creates structured logs in your markdown:

```markdown
- [ ] Implement user dashboard due:2024-01-25 est:4h #frontend
  Time log:
  • 2024-01-15 09:30-10:45 (1h15m) - Initial component setup
  • 2024-01-15 14:00-15:30 (1h30m) - API integration
  Remaining: ~1h15m

- [x] Fix login validation est:1h #backend
  Time log:
  • 2024-01-14 10:00-11:00 (1h) - Debug and fix
  Total: 1h (on estimate!)
```

### Time Display

Tasks show time tracking information in all views:

```
📋 Focus: Overdue & Today's Tasks
─────────────────────────────────────────────────
├─ 🔴 Fix authentication bug (due today) [1h30m/2h] ~2h (L45)
├─ 🟡 Add error handling [45m worked] ~1h (L67)  
└─ ⚪ Update documentation [2h completed] ~1h30m (L89)
```

### Smart Features

- **Task finding**: Partial text search automatically finds tasks to track
- **Persistent timers**: Timers survive app restarts and system reboots
- **Progress tracking**: Visual indicators show worked time vs estimates
- **Clean integration**: Time logs don't clutter your markdown files

### Time Tracking Examples

```bash
# Start working on a task
$ notes time start "Fix auth"
⏰ Started timer for: Fix authentication bug
Location: projects/auth.md:L45

# Check what you're working on
$ notes time status  
🕐 RUNNING: Fix authentication bug
Elapsed: 1h23m • Location: projects/auth.md:L45

# Take a break
$ notes time pause
⏸️ Paused timer for: Fix authentication bug
Elapsed time: 1h23m

# Finish up
$ notes time stop
⏹️ Stopped timer for: Fix authentication bug
Time logged: 1h45m
```

## Installation

```bash
go build -o notes
```

---

*This POC was built by Claude Code.*