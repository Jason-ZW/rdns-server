package types

type Payload struct {
	Fqdn      string              `json:"fqdn,omitempty"`
	Hosts     []string            `json:"hosts"`
	SubDomain map[string][]string `json:"subdomain"`
	Text      string              `json:"text"`
	CNAME     string              `json:"cname"`
	Type      string              `json:"type"`
	Wildcard  bool                `json:"wildcard"`
}
