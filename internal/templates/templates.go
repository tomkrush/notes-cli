package templates

import (
	"strings"
	"time"
)

type NoteType string

const (
	Daily    NoteType = "daily"
	Project  NoteType = "project"
	Meeting  NoteType = "meeting"
	Design   NoteType = "design"
	Learning NoteType = "learning"
)

var templates = map[NoteType]string{
	Daily: `# {{.Date}}

## Tasks
- [ ] 

## Notes


## Follow-ups

`,
	Project: `# {{.Title}}

## Overview


## Goals


## Status


## Actions
- [ ] 

## Notes


## Decisions


## Risks
`,
	Meeting: `# {{.Title}}

**Date:** {{.Date}}
**Attendees:** 

## Agenda


## Discussion


## Decisions


## Action Items
- [ ] 

## Follow-up`,
	Design: `# {{.Title}}

## Problem Statement


## Solution Overview


## Options Considered

### Option A: 

#### Summary

#### Pros
- 

#### Cons
- 

## Recommended Approach


## Implementation Plan


## Risks & Mitigations`,
	Learning: `# {{.Title}}

**Date:** {{.Date}}
**Source:** 

## Key Concepts


## Summary


## Examples


## Code/Commands
` + "```" + `

` + "```" + `

## Notes & Insights


## Action Items
- [ ] 

## Related Topics
- 

## References
- `,
}

type TemplateData struct {
	Title string
	Date  string
}

func Render(noteType NoteType, data TemplateData) string {
	template := templates[noteType]
	
	result := strings.ReplaceAll(template, "{{.Title}}", data.Title)
	result = strings.ReplaceAll(result, "{{.Date}}", data.Date)
	
	return result
}

func GetTemplateData(title string) TemplateData {
	now := time.Now()
	
	return TemplateData{
		Title: title,
		Date:  now.Format("2006-01-02"),
	}
}