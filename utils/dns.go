package utils

import (
	"fmt"
	"strings"

	"github.com/rancher/rdns-server/types"
)

// IsSupportedType returns true only for supported record types.
// currently AAAA, A, CNAME, SRV, TXT record types are supported.
func IsSupportedType(dnsType string) bool {
	switch dnsType {
	case types.RecordTypeAAAA, types.RecordTypeA, types.RecordTypeCNAME, types.RecordTypeSRV, types.RecordTypeTXT:
		return true
	default:
		return false
	}
}

// GetDNSName returns dns name which rdns-server preferred.
func GetDNSName(dnsName string) string {
	return TrimTrailingDot(WildcardUnescape(strings.ToLower(dnsName)))
}

// GetDNSRootName get dns root name.
// e.g. *.example.lb.rancher.cloud => example.lb.rancher.cloud
// e.g. a.example.lb.rancher.cloud => a.example.lb.rancher.cloud
func GetDNSRootName(dnsName string, wildcard bool) string {
	dnsName = WildcardUnescape(TrimTrailingDot(dnsName))
	if wildcard {
		return Level1WithZone(dnsName)
	}
	return dnsName
}

// Level1WithZone remove the first one on the left.
// e.g. *.example.lb.rancher.cloud => example.lb.rancher.cloud
// e.g. a.example.lb.rancher.cloud => example.lb.rancher.cloud
func Level1WithZone(dnsName string) string {
	dnsName = WildcardUnescape(TrimTrailingDot(dnsName))
	ss := strings.Split(dnsName, ".")
	return strings.Join(ss[1:], ".")
}

// EnsureTrailingDot ensure trailing dot.
// e.g. *.example.lb.rancher.cloud => *.example.lb.rancher.cloud.
// e.g. a.example.lb.rancher.cloud => a.example.lb.rancher.cloud.
func EnsureTrailingDot(dnsName string) string {
	return strings.TrimSuffix(dnsName, ".") + "."
}

// TrimTrailingDot trim trailing dot.
// e.g. *.example.lb.rancher.cloud. => *.example.lb.rancher.cloud
// e.g. a.example.lb.rancher.cloud. => a.example.lb.rancher.cloud
func TrimTrailingDot(dnsName string) string {
	return strings.TrimSuffix(dnsName, ".")
}

// WildcardEscape converts \\052 to *
// Route53 stores wildcards escaped.
// See: http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/DomainNameFormat.html?shortFooter=true#domain-name-format-asterisk
func WildcardEscape(s string) string {
	if strings.Contains(s, "*") {
		s = strings.Replace(s, "*", "\\052", 1)
	}
	return s
}

// WildcardUnescape converts \\052 back to *
// Route53 stores wildcards escaped.
// See: http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/DomainNameFormat.html?shortFooter=true#domain-name-format-asterisk
func WildcardUnescape(s string) string {
	if strings.Contains(s, "\\052") {
		s = strings.Replace(s, "\\052", "*", 1)
	}
	return s
}

// TextWithQuotes converts string with quotes.
// Route53 stores text with quotes.
// See: https://docs.aws.amazon.com/zh_cn/Route53/latest/DeveloperGuide/ResourceRecordTypes.html#TXTFormat
func TextWithQuotes(s string) string {
	return fmt.Sprintf("\"%s\"", s)
}

// TextRemoveQuotes converts string with quotes.
// Route53 stores text remove quotes.
// See: https://docs.aws.amazon.com/zh_cn/Route53/latest/DeveloperGuide/ResourceRecordTypes.html#TXTFormat
func TextRemoveQuotes(s string) string {
	return strings.Trim(s, "\"")
}
