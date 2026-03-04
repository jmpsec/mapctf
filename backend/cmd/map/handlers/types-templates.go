package handlers

// LoginTemplateData for passing data to the login template
type LoginTemplateData struct {
	Title     string
	LoginType string
	LoginMsg  string
	LoginURL  string
	UUID      string
}

// IndexTemplateData for passing data to the index template
type IndexTemplateData struct {
	Title string
	UUID  string
}

// CountdownTemplateData for passing data to the countdown template
type CountdownTemplateData struct {
	Title string
	UUID  string
}

// RulesTemplateData for passing data to the rules template
type RulesTemplateData struct {
	Title string
	UUID  string
}

// GameboardTemplateData for passing data to the gameboard template
type GameboardTemplateData struct {
	Title string
	UUID  string
}

// RegistrationTemplateData for passing data to the registration template
type RegistrationTemplateData struct {
	Title            string
	RegistrationType string
	RegisterURL      string
	UUID             string
}

// ErrorTemplateData for passing data to the error template
type ErrorTemplateData struct {
	Title string
	Error string
}
