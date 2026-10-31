package project

import (
	"wsdash/internal/layout"
)

type Source int

const (
	SourceConfig Source = iota
	SourceScan
)

type Project struct {
	Name        string
	Path        string
	Description string
	Tags        []string
	Backend     string
	StartWS     string
	Launch      string // shell command used when the project has no layout
	Workspaces  map[string]layout.Workspace
	Source      Source
}

type Status struct {
	Branch      string
	Dirty       bool
	AheadBehind string
	LastCommit  string
	Err         error
}
