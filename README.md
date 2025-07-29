# notes

A CLI tool for organized note-taking with enhanced markdown tasks and time tracking. Designed for developers who prefer markdown files and git-based workflows.

## Features

- **Enhanced markdown tasks** with due dates, estimates, tags, and priorities
- **Smart time tracking** with structured logs written to your markdown files
- Template-based note creation for different contexts
- Advanced task filtering and smart views (summary, focus modes)
- Full-text search with tag filtering
- Git integration for version control
- Lightweight and fast - just markdown files

## Quick Start

```bash
# Get started in 30 seconds
notes init                    # Set up your notes folder
notes create daily            # Create today's daily note
notes tasks                   # View your tasks
notes time start "task text"  # Start tracking time on a task

# Get help when you need it
notes --help                  # Basic help and getting started
notes help tasks              # Detailed help for task management
notes help time               # Detailed help for time tracking
```

## All Commands

```bash
notes init                         # Initialize folder structure
notes create <type> [title]       # Create a new note
notes list                         # List existing notes
notes tasks [options]              # Show tasks with filters
notes status                       # Show changed notes and todos
notes time <command>               # Time tracking (start/stop/status)
notes search <query> [#tags]       # Search notes by content/tags
notes save [message]               # Commit changes to git
```

## Note Types

- `daily` - Daily notes (auto-dated)
- `project` - Project documentation
- `meeting` - Meeting notes
- `design` - Technical design documents
- `learning` - Learning notes and tutorials

## Enhanced Markdown Tasks

Standard markdown tasks work normally, but this tool adds powerful enhancements:

### Task Syntax
```markdown
- [ ] Basic task
- [ ] Task with due date due:2024-12-31
- [ ] Task with estimate est:2h #urgent  
- [ ] Task with tags #urgent #work #backend
- [ ] URGENT high priority task !!!
```

**Priority keywords**: `urgent`, `critical`, `important` = high priority

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
notes time report [period]                    # Time reports (today, week, month)
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

## Help System

The notes CLI features a progressive help system that shows you information when you need it:

### Basic Help
```bash
notes --help              # Overview and quick start
```
Shows essential commands and getting started steps without overwhelming details.

### Command-Specific Help
```bash
notes help create         # Note types and creation
notes help tasks          # Task views and filters  
notes help time           # Time tracking system
notes help markdown       # Enhanced markdown syntax
notes help search         # Search and filtering
```
Get detailed help for specific features when you need to dive deeper.

### Contextual Help
Error messages include relevant help suggestions, and commands show usage when called incorrectly.

## Installation

```bash
go build -o notes
```

---

*This POC was built by Claude Code.*
