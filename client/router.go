package client

// SchemeName is an alias for string (for readability)
type SchemeName = string

// DomainName is an alias for string (for readability)
type DomainName = string

// TerminalConfig indicates destination
type TerminalConfig struct {
	// The localhost port to which to forward
	Port int
	// Whether or not to unwap the TLS
	TerminateTLS bool
	//Hostname     string
	XForward bool
	// ... create react app...
}

// RouteMap is a map of scheme to domain to port
type RouteMap = map[SchemeName]map[DomainName]*TerminalConfig
