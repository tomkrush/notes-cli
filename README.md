# notes

A CLI tool for organized note-taking with templates and task management. Designed for developers who prefer markdown files and git-based workflows over heavy note-taking apps.

## Features

- Template-based note creation for different contexts
- Task extraction and due date tracking across all notes
- Full-text search with tag filtering
- Git integration for version control
- Lightweight and fast - just markdown files

## Usage

```bash
notes init                          # Initialize folder structure
notes create <type> [title]         # Create a new note
notes list                          # List existing notes
notes tasks                         # Show incomplete tasks with due dates
notes search <query> [#tag ...]     # Search notes by content and tags
notes save [message]                # Commit all changes to git
```

## Note Types

- `daily` - Daily notes (auto-dated)
- `project` - Project documentation
- `meeting` - Meeting notes
- `design` - Technical design documents
- `learning` - Learning notes and tutorials

## Installation

```bash
go build -o notes
```

---

*This POC was built by Claude Code.*