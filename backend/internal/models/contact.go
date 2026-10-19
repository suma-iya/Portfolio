package models

// ContactRequest is the payload submitted from the frontend contact form.
type ContactRequest struct {
	Name    string `json:"name" binding:"required,min=2,max=100"`
	Email   string `json:"email" binding:"required,email,max=255"`
	Subject string `json:"subject" binding:"max=200"`
	Message string `json:"message" binding:"required,min=10,max=5000"`
}
