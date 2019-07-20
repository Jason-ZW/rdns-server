package types

type Payload struct {
	Domain     string            `json:"domain,omitempty"`
	Type       string            `json:"type"`
	Wildcard   bool              `json:"wildcard"`
	Value      string            `json:"value"`
	SubDomains map[string]string `json:"subdomains,omitempty"`
}
