package handlers

// LoginTemplateData for passing data to the login template
type LoginTemplateData struct {
	Title         string
	LoginType     string
	LoginMsg      string
	LoginURL      string
	UUID          string
	TeamLogin     bool
	Authenticated bool
	Admin         bool
}

// IndexTemplateData for passing data to the index template
type IndexTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
}

// CountdownTemplateData for passing data to the countdown template
type CountdownTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
}

// RulesTemplateData for passing data to the rules template
type RulesTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
}

// GameboardTemplateData for passing data to the gameboard template
type GameboardTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
}

// RegistrationTemplateData for passing data to the registration template
type RegistrationTemplateData struct {
	Title            string
	RegistrationType string
	RegistrationMsg  string
	RegisterURL      string
	UUID             string
	OpenRegistration bool
	Authenticated    bool
	Admin            bool
}

// AdminTemplateData for passing data to the admin template
type AdminTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
}

// ErrorTemplateData for passing data to the error template
type ErrorTemplateData struct {
	Title  string
	Error  string
	Status string
	Header string
}
