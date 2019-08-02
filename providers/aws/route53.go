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
func (p *R53Provider) List() (types.Result, error) {
	output := types.Result{
		Type:    types.RecordTypeSupported,
		Records: []types.ResultRecord{},
	}

	f := func(rrs *route53.ListResourceRecordSetsOutput, last bool) (isContinue bool) {
		results := combinedResults(rrs.ResourceRecordSets, "", true)

		// append result sets to output.
		output.Records = append(output.Records, results...)

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
func (p *R53Provider) Get(payload types.Payload) (types.Result, error) {
	rrs, err := p.get(payload.Type, payload.Domain)
	if err != nil {
		logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debugln(err)
		return types.Result{}, err
	}

	rs := p.filterRecords(rrs, payload)

	output := types.Result{
		Domain:   payload.Domain,
		Type:     payload.Type,
		Wildcard: payload.Wildcard,
		Records:  combinedResults(rs, payload.Domain, false),
	}

	return output, nil
}

// Post create supported records.
func (p *R53Provider) Post(payload types.Payload) (types.Result, error) {
	//changes := make([]*route53.Change, 0)

	// generate random DNS name.
	if payload.Domain == "" {
		for i := 0; i < maxSlugHashTimes; i++ {
			prefix := utils.RandStringWithSmall(slugLength)

			// check if the name is available.
			canUse, err := p.keeper.PrefixCanBeUsed(prefix)
			if !canUse || err != nil {
				continue
			}

			payload.Domain = prefix + "." + p.zoneName

			// if wildcard, need add a "*" as prefix.
			if payload.Wildcard {
				payload.Domain = "*" + "." + payload.Domain
			}

			break
		}
	}

	//
	//// set rotate record, record this name cannot be used again.
	//changes = append(changes, p.keeper.SetRotate(payload.Type, payload.Domain).(*route53.Change))
	//
	//// processing domain records.
	//rr := stringToResourceRecord(payload.Type, payload.Value)
	//changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, payload.Domain, rr)...)
	//
	//// set ownership record which keep domain ownership concept.
	//ownership, err := p.keeper.SetOwnership(payload.Type, payload.Domain)
	//if err != nil {
	//	logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debug(err)
	//	return types.Result{}, err
	//}
	//
	//// processing sub-domain records.
	//for k, v := range payload.SubDomains {
	//	subName := strings.ToLower(k) + "." + utils.GetDNSRootName(payload.Domain, payload.Wildcard)
	//
	//	// set sub-domain ownership record which keep sub-domain ownership concept.
	//	changes = append(changes, p.keeper.SetSubDomainOwnership(payload.Type, subName, ownership).(*route53.Change))
	//
	//	srr := make([]*route53.ResourceRecord, 0)
	//	srr = append(srr, stringToResourceRecord(payload.Type, v)...)
	//
	//	// set sub-domain record.
	//	changes = append(changes, p.addChange(route53.ChangeActionUpsert, payload.Type, subName, srr)...)
	//}
	//
	//// submit changes to route53.
	//if err := p.submitChanges(changes); err != nil {
	//	logrus.WithField(dnsLogKey, toDNSLogKey(payload)).Debug(err)
	//	return types.Result{}, err
	//}

	return p.Get(payload)
}

// Put update supported records.
func (p *R53Provider) Put(payload types.Payload) (types.Result, error) {
	return types.Result{}, nil
}

// Delete delete supported records.
func (p *R53Provider) Delete(payload types.Payload) (types.Result, error) {
	return types.Result{}, nil
}

// Renew renew supported records, this only renew records which has prefix with txt-keeper-.
func (p *R53Provider) Renew(payload types.Payload) (types.Result, error) {
	return types.Result{}, nil
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
	change := &route53.Change{
		Action: aws.String(action),
		ResourceRecordSet: &route53.ResourceRecordSet{
			TTL:             aws.Int64(p.ttl),
			Type:            aws.String(dnsType),
			Name:            aws.String(utils.WildcardUnescape(dnsName)),
			ResourceRecords: rr,
		},
	}
	changes = append(changes, change)
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

		if !strings.Contains(rsName, payload.Domain) {
			continue
		}

		if rsType == payload.Type {
			if rsName == payload.Domain {
				output = append(output, rs)
				continue
			}

			// TODO: sub-domain logic will be considered.
			// if it is sub-domain then added it to the output.
			//parentType := payload.Type
			//parentName := utils.GetDNSRootName(payload.Domain, payload.Wildcard)

		}
	}

	return output
}

// combinedResults combine []*route53.ResourceRecordSet results to rdns-server preferred.
func combinedResults(rrs []*route53.ResourceRecordSet, parentName string, isList bool) []types.ResultRecord {
	output := make([]types.ResultRecord, 0)
	kvs := make(map[string]types.ResultRecord, 0)

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
		if !isList && dnsName != utils.GetDNSName(parentName) {
			key = fmt.Sprintf("%s-%s", dnsType, parentName)
			key = strings.ToLower(key)
			prefix := strings.Split(dnsName, ".")[0]
			kvs[key].SubDomains[prefix] = valuesToString(rs.ResourceRecords, dnsType)
			continue
		}

		if v, ok := kvs[key]; ok {
			v.Value = v.Value + "," + valuesToString(rs.ResourceRecords, dnsType)
			v.Value = strings.Trim(v.Value, ",")
			kvs[key] = v
			continue
		}

		r := types.ResultRecord{
			Domain:     dnsName,
			Type:       dnsType,
			Value:      valuesToString(rs.ResourceRecords, dnsType),
			SubDomains: map[string]string{},
		}
		kvs[key] = r
	}

	for _, v := range kvs {
		output = append(output, v)
	}

	return output
}

// valuesToString returns dns string value.
// See: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/ResourceRecordTypes.html
func valuesToString(rr []*route53.ResourceRecord, dnsType string) string {
	multiValue := ""

	switch dnsType {
	case types.RecordTypeAAAA, types.RecordTypeA, types.RecordTypeSRV, types.RecordTypeTXT:
		// multiple values separated by the comma. e.g. 192.168.1.10,192.168.1.11
		for _, r := range rr {
			multiValue = multiValue + utils.TextRemoveQuotes(aws.StringValue(r.Value)) + ","
		}
	default:
		return utils.TextRemoveQuotes(aws.StringValue(rr[0].Value))
	}

	return strings.TrimRight(multiValue, ",")
}

// stringToResourceRecord returns []*route53.ResourceRecord.
func stringToResourceRecord(dnsType, value string) []*route53.ResourceRecord {
	rr := make([]*route53.ResourceRecord, 0)
	values := strings.Split(value, ",")
	for _, v := range values {
		if dnsType == types.RecordTypeTXT {
			v = utils.TextWithQuotes(v)
		}
		rr = append(rr, &route53.ResourceRecord{Value: aws.String(v)})
	}
	return rr
}

// toDNSLogKey returns logrus.Field key.
func toDNSLogKey(payload types.Payload) string {
	return fmt.Sprintf("%s (%s)", payload.Domain, payload.Type)
}
