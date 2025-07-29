package notes

import (
	"fmt"
	"notes/internal/templates"
)

type NoteType = templates.NoteType

const (
	Daily    = templates.Daily
	Project  = templates.Project
	Meeting  = templates.Meeting
	Design   = templates.Design
	Learning = templates.Learning
)

func ValidateNoteType(n NoteType) error {
	switch n {
	case Daily, Project, Meeting, Design, Learning:
		return nil
	default:
		return fmt.Errorf("invalid note type: %s. Available types: daily, project, meeting, design, learning", n)
	}
}

type TaskFilters struct {
	Tags        []string
	Priority    string
	Overdue     bool
	Today       bool
	FilePattern string
	SortBy      string
	Focus       bool
	All         bool
	Summary     bool
	Full        bool
}