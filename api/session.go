package api

// Represents a Session. Currently only holds the private_token, used for
// authentication. This can be extended in the future to store more session
// information such as time created, user email, etc.
type Session struct {
	PrivateToken string `json:"private_token"`
}
