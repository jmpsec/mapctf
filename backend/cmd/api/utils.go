package main

// Template string for the help command and avoid displaying the global options when not needed
var CommandHelpTemplateString = `NAME:
   {{template "helpNameTemplate" .}}

USAGE:
   {{if .Usage}}{{.Usage}}{{end}}

CATEGORY:
   {{.Category}}{{if .Description}}

DESCRIPTION:
   {{template "descriptionTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

OPTIONS:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

OPTIONS:{{template "visibleFlagTemplate" .}}{{end}}
`
