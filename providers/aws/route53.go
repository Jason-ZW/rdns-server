package aws

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rancher/rdns-server/keepers"
	"github.com/rancher/rdns-server/keepers/rds"
	"github.com/rancher/rdns-server/types"
	"github.com/rancher/rdns-server/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	dnsLogKey        = "dns"
	providerName     = "route53"
	maxSlugHashTimes = 100
	slugLength       = 6
)

type R53Provider struct {
	client   *route53.Route53
	keeper   keepers.Keeper
	domain   string
	name     string
	ttl      int64
	zoneName string
	zoneID   string
}

// NewR53Provider contains the subset of the AWS Route53 API that we actually use.
// See: https://docs.aws.amazon.com/sdk-for-go/api/service/route53
func NewR53Provider() *R53Provider {
	retryInt, err := strconv.Atoi(os.Getenv("AWS_RETRY"))
	if err != nil {
		logrus.Fatalln(err)
	}

	s, err := session.NewSession()
	if err != nil {
		logrus.Fatalln(err)
	}

	config := &aws.Config{
		Credentials: credentials.NewEnvCredentials(),
		MaxRetries:  aws.Int(retryInt),
	}

	arn := os.Getenv("AWS_ASSUME_ROLE")
	if arn != "" {
		config.WithCredentials(stscreds.NewCredentials(s, arn))
	}

	r53 := route53.New(s, config)

	zone, err := r53.GetHostedZone(&route53.GetHostedZoneInput{
		Id: aws.String(os.Getenv("AWS_HOSTED_ZONE_ID")),
	})
	if err != nil {
		logrus.Fatalln(err)
	}

	zoneName := utils.TrimTrailingDot(aws.StringValue(zone.HostedZone.Name))
	zoneID := aws.StringValue(zone.HostedZone.Id)

	if os.Getenv("DOMAIN") != utils.TrimTrailingDot(zoneName) {
		logrus.Fatalln("domain name not match aws hosted zone name")
	}

	ttl, err := strconv.ParseInt(os.Getenv("TTL"), 10, 64)
	if err != nil {
		logrus.Fatalln("failed to parse ttl")
	}

	// initialize keeper which maintain concept information.
	keeper := rds.NewRDS(zoneName, ttl)
	keepers.SetKeeper(keeper)

	return &R53Provider{
		client:   r53,
		keeper:   keeper,
		domain:   zoneName,
		name:     providerName,
		ttl:      ttl,
		zoneName: zoneName,
		zoneID:   zoneID,
	}
}

// GetProviderName return current provider name.
func (p *R53Provider) GetProviderName() string {
	return p.name
}

// GetZoneName return current zone name.
func (p *R53Provider) GetZoneName() string {
	return utils.TrimTrailingDot(p.zoneName)
}

// List return all supported records, timeouts may occur when the records is extremely large.
func (p *R53Provider) List() ([]types.Domain, error) {
	output := make([]types.Domain, 0)

	f := func(rrs *route53.ListResourceRecordSetsOutput, last bool) (isContinue bool) {
		results := combinedResults(rrs.ResourceRecordSets, "", true)

		// append result sets to output.
		output = append(output, results...)

		// timeouts may occur when the records is extremely large.
		return true
	}

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(p.zoneID),
	}

	if err := p.client.ListResourceRecordSetsPages(params, f); err != nil {
		logrus.Debugln(err)
		return output, err
	}

	return output, nil
}

// Get return relevant records based on key information.
func (p *R53Provider) Get(payload types.Payload) (types.Domain, error) {
	rrs, err := p.get(payload.Type, payload.Fqdn)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	rs := p.filterRecords(rrs, payload)

	if len(rs) <= 0 {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln("no record(s) found")
		return types.Domain{}, errors.New("no record(s) found")
	}

	// get key information which contains token and expire field.
	keep, err := p.keeper.GetKeep(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	output := combinedResults(rs, payload.Fqdn, false)[0]
	output.Expiration = keep.Expire

	return output, nil
}

// Post create A/AAAA records.
func (p *R53Provider) Post(payload types.Payload) (types.Domain, error) {
	// generate random DNS name.
	if payload.Fqdn == "" {
		for i := 0; i < maxSlugHashTimes; i++ {
			prefix := utils.RandStringWithSmall(slugLength)

			// check if the name is available.
			canUse, err := p.keeper.PrefixCanBeUsed(prefix)
			if !canUse || err != nil {
				continue
			}

			payload.Fqdn = prefix + "." + p.zoneName

			// if wildcard, need add a "*" as prefix.
			if payload.Wildcard {
				payload.Fqdn = "*" + "." + payload.Fqdn
			}

			break
		}
	}

	if payload.Fqdn == "" {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln("domain name can not be empty")
		return types.Domain{}, errors.New("domain name can not be empty")
	}

	// processing keeper records, this is database operation.
	keep, err := p.keeper.SetKeeps(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	// processing domain records.
	rr := stringToResourceRecord(payload.Hosts)

	changes := make([]*route53.Change, 0)
	changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, payload.Fqdn, rr)...)

	// processing wildcard records.
	if !payload.Wildcard {
		changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, "*."+payload.Fqdn, rr)...)
	}

	// processing sub-domain records.
	if !payload.Wildcard {
		for k, v := range payload.SubDomain {
			subName := strings.ToLower(k) + "." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

			// set sub-domain record.
			srr := stringToResourceRecord(v)
			changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, subName, srr)...)
		}
	}

	// submit changes to route53.
	if err := p.submitChanges(changes); err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	return outputFromPayload(payload, keep)
}

// Put update A/AAAA records.
func (p *R53Provider) Put(payload types.Payload) (types.Domain, error) {
	// obtain existing records
	existing, err := p.Get(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	// processing keeper records, this is database operation.
	keep, err := p.keeper.PutKeeps(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	if existing.Fqdn == "" {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln("no exist record(s)")
		return types.Domain{}, errors.New("no exist record(s)")
	}

	changes := make([]*route53.Change, 0)

	// processing domain records.
	rr := stringToResourceRecord(payload.Hosts)
	if len(payload.Hosts) > 0 {
		changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, payload.Fqdn, rr)...)
	} else {
		rr := stringToResourceRecord(existing.Hosts)
		changes = append(changes, p.addChange(route53.ChangeActionDelete, payload.Type, payload.Fqdn, rr)...)
	}

	// processing wildcard records.
	if !payload.Wildcard {
		if len(payload.Hosts) > 0 {
			changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, "*."+payload.Fqdn, rr)...)
		} else {
			rr := stringToResourceRecord(existing.Hosts)
			changes = append(changes, p.addChange(route53.ChangeActionDelete, payload.Type, "*."+payload.Fqdn, rr)...)
		}
	}

	// processing sub-domain records.
	if !payload.Wildcard {
		for k, v := range payload.SubDomain {
			subName := strings.ToLower(k) + "." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

			// set sub-domain record.
			srr := stringToResourceRecord(v)
			changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, subName, srr)...)
		}
	}

	// processing useless records.
	if !payload.Wildcard {
		for k, v := range existing.SubDomain {
			if _, ok := payload.SubDomain[k]; !ok {
				subName := strings.ToLower(k) + "." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

				// del useless records.
				srr := stringToResourceRecord(v)
				changes = append(changes, p.addChange(route53.ChangeActionDelete, payload.Type, subName, srr)...)
			}
		}
	}

	// submit changes to route53.
	if err := p.submitChanges(changes); err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	keep.Token = ""

	return outputFromPayload(payload, keep)
}

// Delete delete A/AAAA records.
func (p *R53Provider) Delete(payload types.Payload) (types.Domain, error) {
	// obtain existing records
	existing, err := p.Get(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	// processing keeper records, this is database operation.
	_, err = p.keeper.DeleteKeeps(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	if existing.Fqdn == "" {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln("no exist record(s)")
		return types.Domain{}, errors.New("no exist record(s)")
	}

	changes := make([]*route53.Change, 0)

	// processing domain records.
	rr := stringToResourceRecord(existing.Hosts)
	if len(existing.Hosts) > 0 {
		changes = append(changes, p.addChange(route53.ChangeActionDelete, payload.Type, payload.Fqdn, rr)...)
	}

	// processing wildcard records.
	if !payload.Wildcard && len(existing.Hosts) > 0 {
		changes = append(changes, p.addChange(route53.ChangeActionDelete, payload.Type, "*."+payload.Fqdn, rr)...)
	}

	// processing sub-domain records.
	if !payload.Wildcard {
		for k, v := range existing.SubDomain {
			subName := strings.ToLower(k) + "." + utils.GetDNSRootName(payload.Fqdn, payload.Wildcard)

			// delete sub-domain record.
			srr := stringToResourceRecord(v)
			changes = append(changes, p.addChange(route53.ChangeActionDelete, payload.Type, subName, srr)...)
		}
	}

	// submit changes to route53.
	if err := p.submitChanges(changes); err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	return types.Domain{}, nil
}

// Renew renew supported records.
func (p *R53Provider) Renew(payload types.Payload) (types.Domain, error) {
	// processing keeper records, this is database operation.
	err := p.keeper.RenewKeeps(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	keep, err := p.keeper.GetKeep(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	return types.Domain{
		Fqdn:       keep.Domain,
		Expiration: keep.Expire,
	}, nil
}

// GetTXT return relevant TXT records based on key information.
func (p *R53Provider) GetTXT(payload types.Payload) (types.Domain, error) {
	rrs, err := p.get(payload.Type, payload.Fqdn)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	rs := p.filterRecords(rrs, payload)

	if len(rs) <= 0 {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln("no record(s) found")
		return types.Domain{}, errors.New("no record(s) found")
	}

	// get key information which contains token and expire field.
	keep, err := p.keeper.GetKeep(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	output := combinedResults(rs, payload.Fqdn, false)[0]
	output.Expiration = keep.Expire

	return output, nil
}

// PostTXT create TXT records.
func (p *R53Provider) PostTXT(payload types.Payload) (types.Domain, error) {
	// generate random DNS name.
	if payload.Fqdn == "" {
		for i := 0; i < maxSlugHashTimes; i++ {
			prefix := utils.RandStringWithSmall(slugLength)

			// check if the name is available.
			canUse, err := p.keeper.PrefixCanBeUsed(prefix)
			if !canUse || err != nil {
				continue
			}

			payload.Fqdn = prefix + "." + p.zoneName

			// if wildcard, need add a "*" as prefix.
			if payload.Wildcard {
				payload.Fqdn = "*" + "." + payload.Fqdn
			}

			break
		}
	}

	if payload.Fqdn == "" {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln("domain name can not be empty")
		return types.Domain{}, errors.New("domain name can not be empty")
	}

	// processing keeper records, this is database operation.
	keep, err := p.keeper.SetKeeps(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	// processing domain records.
	rr := stringToResourceRecord([]string{utils.TextWithQuotes(payload.Text)})

	changes := make([]*route53.Change, 0)
	changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, payload.Fqdn, rr)...)

	// submit changes to route53.
	if err := p.submitChanges(changes); err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	return outputFromPayload(payload, keep)
}

// PutTXT update TXT records.
func (p *R53Provider) PutTXT(payload types.Payload) (types.Domain, error) {
	// processing keeper records, this is database operation.
	keep, err := p.keeper.PutKeeps(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	changes := make([]*route53.Change, 0)

	// processing domain records.
	rr := stringToResourceRecord([]string{utils.TextWithQuotes(payload.Text)})
	changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, payload.Fqdn, rr)...)

	// submit changes to route53.
	if err := p.submitChanges(changes); err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	keep.Token = ""

	return outputFromPayload(payload, keep)
}

// DeleteTXT delete TXT records.
func (p *R53Provider) DeleteTXT(payload types.Payload) (types.Domain, error) {
	// obtain existing records
	existing, err := p.GetTXT(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	// processing keeper records, this is database operation.
	_, err = p.keeper.DeleteKeeps(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	if existing.Fqdn == "" {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln("no exist record(s)")
		return types.Domain{}, errors.New("no exist record(s)")
	}

	changes := make([]*route53.Change, 0)

	// processing domain records.
	rr := stringToResourceRecord([]string{utils.TextWithQuotes(existing.Text)})
	changes = append(changes, p.addChange(route53.ChangeActionDelete, payload.Type, payload.Fqdn, rr)...)

	// submit changes to route53.
	if err := p.submitChanges(changes); err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	return types.Domain{}, nil
}

// GetCNAME return relevant CNAME records based on key information.
func (p *R53Provider) GetCNAME(payload types.Payload) (types.Domain, error) {
	rrs, err := p.get(payload.Type, payload.Fqdn)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	rs := p.filterRecords(rrs, payload)

	if len(rs) <= 0 {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln("no record(s) found")
		return types.Domain{}, errors.New("no record(s) found")
	}

	// get key information which contains token and expire field.
	keep, err := p.keeper.GetKeep(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	output := combinedResults(rs, payload.Fqdn, false)[0]
	output.Expiration = keep.Expire

	return output, nil
}

// PostCNAME create CNAME records.
func (p *R53Provider) PostCNAME(payload types.Payload) (types.Domain, error) {
	// generate random DNS name.
	if payload.Fqdn == "" {
		for i := 0; i < maxSlugHashTimes; i++ {
			prefix := utils.RandStringWithSmall(slugLength)

			// check if the name is available.
			canUse, err := p.keeper.PrefixCanBeUsed(prefix)
			if !canUse || err != nil {
				continue
			}

			payload.Fqdn = prefix + "." + p.zoneName

			// if wildcard, need add a "*" as prefix.
			if payload.Wildcard {
				payload.Fqdn = "*" + "." + payload.Fqdn
			}

			break
		}
	}

	if payload.Fqdn == "" {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln("domain name can not be empty")
		return types.Domain{}, errors.New("domain name can not be empty")
	}

	// processing keeper records, this is database operation.
	keep, err := p.keeper.SetKeeps(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	// processing domain records.
	rr := stringToResourceRecord([]string{payload.CNAME})

	changes := make([]*route53.Change, 0)
	changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, payload.Fqdn, rr)...)

	// submit changes to route53.
	if err := p.submitChanges(changes); err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	return outputFromPayload(payload, keep)
}

// PutCNAME update CNAME records.
func (p *R53Provider) PutCNAME(payload types.Payload) (types.Domain, error) {
	// processing keeper records, this is database operation.
	keep, err := p.keeper.PutKeeps(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	changes := make([]*route53.Change, 0)

	// processing domain records.
	rr := stringToResourceRecord([]string{payload.CNAME})
	changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, payload.Fqdn, rr)...)

	// submit changes to route53.
	if err := p.submitChanges(changes); err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	keep.Token = ""

	return outputFromPayload(payload, keep)
}

// DeleteCNAME delete CNAME records.
func (p *R53Provider) DeleteCNAME(payload types.Payload) (types.Domain, error) {
	// obtain existing records
	existing, err := p.GetCNAME(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	// processing keeper records, this is database operation.
	_, err = p.keeper.DeleteKeeps(payload)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	if existing.Fqdn == "" {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln("no exist record(s)")
		return types.Domain{}, errors.New("no exist record(s)")
	}

	changes := make([]*route53.Change, 0)

	// processing domain records.
	rr := stringToResourceRecord([]string{existing.CNAME})
	changes = append(changes, p.addChange(route53.ChangeActionDelete, payload.Type, payload.Fqdn, rr)...)

	// submit changes to route53.
	if err := p.submitChanges(changes); err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Domain{}, err
	}

	return types.Domain{}, nil
}

// submitChanges submit changes to route53.
func (p *R53Provider) submitChanges(changes []*route53.Change) error {
	_, err := p.client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(p.zoneID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

// addChange add *route53.Change to []*route53.Change.
func (p *R53Provider) addChange(action, dnsType, dnsName string, rr []*route53.ResourceRecord) []*route53.Change {
	changes := make([]*route53.Change, 0)
	filter := make([]*route53.ResourceRecord, 0)

	for _, r := range rr {
		if aws.StringValue(r.Value) == "" {
			continue
		}
		filter = append(filter, r)
	}

	change := &route53.Change{
		Action: aws.String(action),
		ResourceRecordSet: &route53.ResourceRecordSet{
			TTL:             aws.Int64(p.ttl),
			Type:            aws.String(dnsType),
			Name:            aws.String(utils.WildcardUnescape(dnsName)),
			ResourceRecords: filter,
		},
	}
	if len(change.ResourceRecordSet.ResourceRecords) > 0 {
		changes = append(changes, change)
	}

	return changes
}

// get returns route53.ListResourceRecordSetsOutput by dnsName & dnsType
func (p *R53Provider) get(dnsType, dnsName string) (*route53.ListResourceRecordSetsOutput, error) {
	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(p.zoneID),
		StartRecordType: aws.String(dnsType),
		StartRecordName: aws.String(dnsName),
	}

	// returns up to 100 resource record sets at a time in ASCII order.
	// queries outside of more than 100 record sets are not supported.
	return p.client.ListResourceRecordSets(params)
}

// filterRecords returns valid records and sub-domain records.
func (p *R53Provider) filterRecords(rrs *route53.ListResourceRecordSetsOutput, payload types.Payload) []*route53.ResourceRecordSet {
	output := make([]*route53.ResourceRecordSet, 0)

	for _, rs := range rrs.ResourceRecordSets {
		rsType := aws.StringValue(rs.Type)
		rsName := utils.GetDNSName(aws.StringValue(rs.Name))

		if !strings.Contains(rsName, payload.Fqdn) {
			continue
		}

		if rsType == payload.Type {
			if rsName == payload.Fqdn {
				output = append(output, rs)
				continue
			}

			// if it is sub-domain then added it to the output.
			if keepers.GetKeeper().IsSubDomain(rsName) {
				output = append(output, rs)
			}
		}
	}

	return output
}

// combinedResults combine []*route53.ResourceRecordSet results to rdns-server preferred.
func combinedResults(rrs []*route53.ResourceRecordSet, parentName string, isList bool) []types.Domain {
	output := make([]types.Domain, 0)
	kvs := make(map[string]types.Domain, 0)

	for _, rs := range rrs {
		dnsType := aws.StringValue(rs.Type)
		dnsName := utils.GetDNSName(aws.StringValue(rs.Name))

		// ignore illegal dns record.
		if !utils.IsSupportedType(dnsType) {
			continue
		}

		key := fmt.Sprintf("%s-%s", dnsType, dnsName)
		key = strings.ToLower(key)

		// deal with sub-domain records
		if !isList && dnsName != utils.GetDNSName(parentName) && utils.HasSubDomain(dnsType) {
			key = fmt.Sprintf("%s-%s", dnsType, parentName)
			key = strings.ToLower(key)
			// if no root domain, we need to hold empty value record.
			if _, ok := kvs[key]; !ok {
				kvs[key] = types.Domain{
					Fqdn:      parentName,
					Type:      dnsType,
					SubDomain: map[string][]string{},
				}
			}
			prefix := strings.Split(dnsName, ".")[0]
			kvs[key].SubDomain[prefix] = valuesToStrings(rs.ResourceRecords)
			continue
		}

		if v, ok := kvs[key]; ok {
			if v.Type == types.RecordTypeCNAME || v.Type == types.RecordTypeTXT {
				continue
			}

			v.Hosts = valuesToStrings(rs.ResourceRecords)
			kvs[key] = v
			continue
		}

		r := types.Domain{
			Fqdn:      dnsName,
			Type:      dnsType,
			SubDomain: map[string][]string{},
		}

		if dnsType == types.RecordTypeTXT {
			r.Text = valuesToStrings(rs.ResourceRecords)[0]
		} else if dnsType == types.RecordTypeCNAME {
			r.CNAME = valuesToStrings(rs.ResourceRecords)[0]
		} else {
			r.Hosts = valuesToStrings(rs.ResourceRecords)
		}
		kvs[key] = r
	}

	for _, v := range kvs {
		output = append(output, v)
	}

	return output
}

// valuesToStrings convert dns values to []string.
// See: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/ResourceRecordTypes.html
func valuesToStrings(rr []*route53.ResourceRecord) []string {
	ss := make([]string, 0)
	for _, r := range rr {
		ss = append(ss, utils.TextRemoveQuotes(aws.StringValue(r.Value)))
	}
	return ss
}

// stringToResourceRecord returns []*route53.ResourceRecord.
func stringToResourceRecord(values []string) []*route53.ResourceRecord {
	rr := make([]*route53.ResourceRecord, 0)
	for _, v := range values {
		rr = append(rr, &route53.ResourceRecord{Value: aws.String(v)})
	}
	return rr
}

// toDNSLogKey returns logrus.Field key.
func toDNSLogKey(payload types.Payload) string {
	return fmt.Sprintf("%s (%s)", payload.Fqdn, payload.Type)
}

// outputFromPayload assemble output from payload.
func outputFromPayload(payload types.Payload, keep keepers.Keep) (types.Domain, error) {
	output := types.Domain{}

	output.Expiration = keep.Expire
	output.Token = keep.Token
	output.Fqdn = payload.Fqdn
	output.Type = payload.Type
	output.Text = payload.Text
	output.Hosts = payload.Hosts
	output.CNAME = payload.CNAME
	output.SubDomain = payload.SubDomain

	return output, nil
}
