# notes

A CLI tool for organized note-taking with templates and task management. Designed for developers who prefer markdown files and git-based workflows over heavy note-taking apps.

## Features

- Template-based note creation for different contexts
- Task extraction and due date tracking across all notes
- Advanced task filtering by tag, priority, due date, and file
- Full-text search with tag filtering
- Git integration for version control
- Lightweight and fast - just markdown files

## Usage

```bash
notes init                          # Initialize folder structure
notes create <type> [title]         # Create a new note
notes list                          # List existing notes
notes tasks [filters]               # Show incomplete tasks with due dates
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
- Tags: `- [ ] Important task #urgent #work`
- Priority: Use keywords like `urgent`, `critical`, `important`, `!!!`, `!!`, or `soon`

### Task Filtering
Filter tasks using command-line options:

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

## Installation

```bash
go build -o notes
```

---

*This POC was built by Claude Code.*