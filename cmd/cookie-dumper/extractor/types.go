package extractor

// Cookies holds extracted session cookies from a browser.
// The map key is the cookie name (e.g. "sessionid", "csrftoken", "c_user", "xs").
type Cookies struct {
	Values map[string]string
}

// Extractor is the interface implemented by each browser backend.
type Extractor interface {
	Extract() (*Cookies, error)
}
