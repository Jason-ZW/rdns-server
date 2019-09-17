package types

import "time"

type Response struct {
	Status  int    `json:"status"`
	Message string `json:"msg"`
	Data    Domain `json:"data,omitempty"`
	Token   string `json:"token"`
}

type ResponseList struct {
	Status  int      `json:"status"`
	Message string   `json:"msg"`
	Datum   []Domain `json:"datum,omitempty"`
	Type    string   `json:"type"`
}

type Domain struct {
	Fqdn       string              `json:"fqdn,omitempty"`
	Hosts      []string            `json:"hosts,omitempty"`
	SubDomain  map[string][]string `json:"subdomain,omitempty"`
	Text       string              `json:"text,omitempty"`
	CNAME      string              `json:"cname,omitempty"`
	Type       string              `json:"type"`
	Token      string              `json:"token"`
	Expiration *time.Time          `json:"expiration,omitempty"`
}
