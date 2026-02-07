package models

type User struct {
	ID           string
	TenantID     string
	Email        string
	PasswordHash string
	Role         string
	Status       string
	FirstName    string
	LastName     string
	Locale       string
}
