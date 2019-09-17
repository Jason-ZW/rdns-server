package types

type Payload struct {
	Fqdn      string              `json:"fqdn,omitempty"`
	Hosts     []string            `json:"hosts,omitempty"`
	SubDomain map[string][]string `json:"subdomain,omitempty"`
	Text      string              `json:"text,omitempty"`
	CNAME     string              `json:"cname,omitempty"`
	Type      string              `json:"type,omitempty"`
	Wildcard  bool                `json:"wildcard,omitempty"`
	Cleanup   string              `json:"cleanup,omitempty"`
}
