package types

const (
	RecordTypeA         string = "A"
	RecordTypeAAAA      string = "AAAA"
	RecordTypeCNAME     string = "CNAME"
	RecordTypeTXT       string = "TXT"
	RecordTypeSRV       string = "SRV"
	RecordTypeNS        string = "NS"
	RecordTypeMX        string = "MX"
	RecordTypePTR       string = "PTR"
	RecordTypeCAA       string = "CAA"
	RecordTypeSupported string = "AAAA, A, CNAME, SRV, TXT"
	RecordTypeNone      string = "NONE"
)

type Record struct {
	Domain   string `json:"domain,omitempty"`
	Group    string `json:"group,omitempty"`
	Host     string `json:"host,omitempty"`
	Mail     bool   `json:"mail,omitempty"`
	Port     int    `json:"port,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Strip    int    `json:"strip,omitempty"`
	Target   int    `json:"target,omitempty"`
	Text     string `json:"text,omitempty"`
	Token    string `json:"token,omitempty"`
	TTL      int64  `json:"ttl,omitempty"`
	Type     string `json:"type,omitempty"`
	Weight   int    `json:"weight,omitempty"`
	Expire   int    `json:"expire,omitempty"`
}
