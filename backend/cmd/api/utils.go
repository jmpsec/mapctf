package main

import "math/rand"

// charset for the random password
var charset = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

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

// GenerateRandomPassword to generate a random password
func GenerateRandomPassword(length int) string {
	password := make([]byte, length)
	for i := range password {
		password[i] = charset[rand.Intn(len(charset))]
	}
	return string(password)
}
