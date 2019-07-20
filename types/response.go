package types

type Response struct {
	Status  int    `json:"status"`
	Message string `json:"msg,omitempty"`
	Datum   Result `json:"datum,omitempty"`
}

type Result struct {
	Domain   string         `json:"domain,omitempty"`
	Ignores  []ResultRecord `json:"ignores,omitempty"`
	Records  []ResultRecord `json:"records"`
	Type     string         `json:"type"`
	Wildcard bool           `json:"wildcard"`
}

type ResultRecord struct {
	Domain     string            `json:"domain,omitempty"`
	Type       string            `json:"type"`
	Value      string            `json:"value"`
	SubDomains map[string]string `json:"subdomains,omitempty"`
	Token      string            `json:"token"`
	Expire     string            `json:"expire"`
}
