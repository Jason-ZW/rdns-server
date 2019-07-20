package aws

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rancher/rdns-server/keepers"
	awsKeeper "github.com/rancher/rdns-server/keepers/aws"
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
	keeper := awsKeeper.NewRout53Keeper(r53, zoneName, zoneID, ttl)
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
		results := assembleResults(rrs.ResourceRecordSets, "", true)

		// append result sets to output.
		output.Records = append(output.Records, results...)

		// timeouts may occur when the records is extremely large.
		return true
	}

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(p.zoneID),
	}

	if err := p.client.ListResourceRecordSetsPages(params, f); err != nil {
		logrus.WithError(err).Debugln("list route53 record(s) failed")
		return output, err
	}

	return output, nil
}

// Get return relevant records based on key information.
func (p *R53Provider) Get(payload types.Payload) (types.Result, error) {
	output := types.Result{
		Domain:   payload.Domain,
		Type:     payload.Type,
		Wildcard: payload.Wildcard,
		Records:  []types.ResultRecord{},
	}

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(p.zoneID),
		StartRecordType: aws.String(payload.Type),
		StartRecordName: aws.String(payload.Domain),
	}

	rrs, err := p.getMatchesRecordSets(params, payload)
	if err != nil {
		logrus.WithError(err).Debugf("get route53 matches record(s) failed: %s (%s)\n", payload.Domain, payload.Type)
		return output, err
	}

	results := assembleResults(rrs.ResourceRecordSets, payload.Domain, false)

	// append result sets to output.
	output.Records = append(output.Records, results...)

	// get token & expire from ownership-keeper TXT record which keep domain ownership concept.
	ownership, err := p.keeper.GetOwnership(payload.Type, payload.Domain)
	if err != nil {
		logrus.Debug(err)
		return output, err
	}

	if len(output.Records) != 1 {
		logrus.WithError(err).Debugf("found multi record(s): %s (%s)\n", payload.Domain, payload.Type)
		return output, errors.New("found multi record(s)")
	}

	output.Records[0].Token = ownership.Token
	output.Records[0].Expire = ownership.Expire

	return output, nil
}

// Post create supported records.
func (p *R53Provider) Post(payload types.Payload) (types.Result, error) {
	output := types.Result{
		Domain:   payload.Domain,
		Type:     payload.Type,
		Wildcard: payload.Wildcard,
		Records:  []types.ResultRecord{},
	}

	// generate random DNS names.
	if payload.Domain == "" {
		for i := 0; i < maxSlugHashTimes; i++ {
			payload.Domain = utils.RandStringWithSmall(slugLength) + "." + p.zoneName
			if payload.Wildcard {
				payload.Domain = "*" + "." + payload.Domain
			}

			// check rotate-keeper record to determine whether the name is available.
			canUse, err := p.keeper.NameCanUse(payload.Type, payload.Domain)
			if !canUse || err != nil {
				continue
			}

			_, err = p.keeper.SetRotate(payload.Type, payload.Domain)
			if err != nil {
				logrus.Debug(err)
				return output, err
			}

			break
		}
	}

	// set ownership-keeper TXT record which keep domain ownership concept.
	ownership, err := p.keeper.SetOwnership(payload.Type, payload.Domain)
	if err != nil {
		logrus.Debug(err)
		return output, err
	}

	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(p.zoneID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{},
		},
	}

	rr := make([]*route53.ResourceRecord, 0)

	// set sub-domain ownership-keeper TXT record which keep sub-domain ownership concept.
	for k, v := range payload.SubDomains {
		sn := strings.ToLower(k) + "." + utils.GetDNSRootName(payload.Domain, payload.Wildcard)
		err := p.keeper.SetSubOwnership(payload.Type, sn, ownership)
		if err != nil {
			logrus.Debug(err)
			return output, err
		}
		// set sub-domain records.
		srr := make([]*route53.ResourceRecord, 0)

		sv := strings.Split(v, ",")
		for _, vv := range sv {
			if payload.Type == types.RecordTypeTXT {
				vv = utils.TextWithQuotes(vv)
			}
			srr = append(srr, &route53.ResourceRecord{Value: aws.String(vv)})
		}

		params = p.batchChanges(params, route53.ChangeActionUpsert, payload.Type, sn, p.ttl, srr)
	}

	// set domain records.
	values := strings.Split(payload.Value, ",")
	for _, v := range values {
		if payload.Type == types.RecordTypeTXT {
			v = utils.TextWithQuotes(v)
		}
		rr = append(rr, &route53.ResourceRecord{Value: aws.String(v)})
	}

	params = p.batchChanges(params, route53.ChangeActionUpsert, payload.Type, payload.Domain, p.ttl, rr)

	_, err = p.client.ChangeResourceRecordSets(params)
	if err != nil {
		logrus.WithError(err).Debugf("set route53 record(s) failed: %s (%s)\n", payload.Domain, payload.Type)
		return output, err
	}

	return p.Get(payload)
}

// Put update supported records.
func (p *R53Provider) Put(payload types.Payload) (types.Result, error) {
	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(p.zoneID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{},
		},
	}

	origin, err := p.Get(payload)
	if err != nil {
		return types.Result{}, err
	}

	if len(origin.Records) < 1 {
		logrus.Debugf("no exist record(s): %s (%s)\n", payload.Domain, payload.Type)
		return types.Result{}, err
	}

	// add the matched records to changeBatch.
	if origin.Records[0].Value != payload.Value {
		rr := make([]*route53.ResourceRecord, 0)

		values := strings.Split(payload.Value, ",")
		for _, v := range values {
			if payload.Type == types.RecordTypeTXT {
				v = utils.TextWithQuotes(v)
			}
			rr = append(rr, &route53.ResourceRecord{Value: aws.String(v)})
		}

		params = p.batchChanges(params, route53.ChangeActionUpsert, payload.Type, payload.Domain, p.ttl, rr)
	}

	for _, r := range origin.Records {
		if !utils.IsSupportedType(r.Type) || r.Type != payload.Type || r.Domain != payload.Domain {
			continue
		}
		for k, vv := range r.SubDomains {
			if sv, ok := payload.SubDomains[k]; ok {
				if sv == vv {
					// no need to handle.
					continue
				}

				// add the matched sub-domain records which need to be updated to changeBatch.
				srr := make([]*route53.ResourceRecord, 0)

				values := strings.Split(sv, ",")
				for _, v := range values {
					if payload.Type == types.RecordTypeTXT {
						v = utils.TextWithQuotes(v)
					}
					srr = append(srr, &route53.ResourceRecord{Value: aws.String(v)})
				}

				params = p.batchChanges(params, route53.ChangeActionUpsert, payload.Type, k+"."+payload.Domain, p.ttl, srr)
			} else {
				// add the matched sub-domain records which need to be deleted to changeBatch.
				srr := make([]*route53.ResourceRecord, 0)

				values := strings.Split(vv, ",")
				for _, v := range values {
					if payload.Type == types.RecordTypeTXT {
						v = utils.TextWithQuotes(v)
					}
					srr = append(srr, &route53.ResourceRecord{Value: aws.String(v)})
				}

				params = p.batchChanges(params, route53.ChangeActionDelete, payload.Type, k+"."+payload.Domain, p.ttl, srr)

				// delete sub-domain ownership-keeper TXT record which keep ownership concept.
				if err := p.keeper.DeleteOwnership(payload.Type, k+"."+payload.Domain); err != nil {
					logrus.Debug(err)
					return types.Result{}, err
				}
			}
		}
		for k, vv := range payload.SubDomains {
			if _, ok := r.SubDomains[k]; !ok {
				// add sub-domain records to changeBatch.
				srr := make([]*route53.ResourceRecord, 0)

				values := strings.Split(vv, ",")
				for _, v := range values {
					if payload.Type == types.RecordTypeTXT {
						v = utils.TextWithQuotes(v)
					}
					srr = append(srr, &route53.ResourceRecord{Value: aws.String(v)})
				}

				params = p.batchChanges(params, route53.ChangeActionUpsert, payload.Type, k+"."+payload.Domain, p.ttl, srr)

				// set sub-domain ownership-keeper TXT record which keep ownership concept.
				ownership, err := p.keeper.GetOwnership(payload.Type, payload.Domain)
				if err != nil {
					logrus.Debug(err)
					return types.Result{}, err
				}
				if err := p.keeper.SetSubOwnership(payload.Type, k+"."+payload.Domain, ownership); err != nil {
					logrus.Debug(err)
					return types.Result{}, err
				}
			}
		}
	}

	if len(params.ChangeBatch.Changes) > 0 {
		_, err = p.client.ChangeResourceRecordSets(params)
		if err != nil {
			logrus.WithError(err).Debugf("update route53 record(s) failed: %s (%s)\n", payload.Domain, payload.Type)
			return types.Result{}, err
		}
	}

	return p.Get(payload)
}

// Delete delete supported records.
func (p *R53Provider) Delete(payload types.Payload) (types.Result, error) {
	output := types.Result{
		Domain:   payload.Domain,
		Type:     payload.Type,
		Wildcard: payload.Wildcard,
	}

	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(p.zoneID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{},
		},
	}

	rrs, err := p.getMatchesRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(p.zoneID),
		StartRecordType: aws.String(payload.Type),
		StartRecordName: aws.String(payload.Domain),
	}, payload)
	if err != nil {
		logrus.WithError(err).Debugf("get route53 matches record(s) failed: %s (%s)\n", payload.Domain, payload.Type)
		return output, err
	}

	rr := make([]*route53.ResourceRecord, 0)

	for _, rs := range rrs.ResourceRecordSets {
		dnsType := aws.StringValue(rs.Type)
		dnsName := utils.GetDNSName(aws.StringValue(rs.Name))

		if !utils.IsSupportedType(dnsType) {
			continue
		}

		if dnsType == payload.Type {
			if dnsName == payload.Domain {
				// add matched records to changeBatch.
				rr = append(rr, rs.ResourceRecords...)
				// delete ownership-keeper TXT record which keep ownership concept.
				// just print error, no need to handle.
				if err := p.keeper.DeleteOwnership(dnsType, dnsName); err != nil {
					logrus.Debug(err)
				}
				continue
			}

			if keepers.GetKeeper().IsSubDomain(dnsType, dnsName, payload.Type, utils.GetDNSRootName(payload.Domain, payload.Wildcard)) {
				// add matched sub-domain records to changeBatch.
				params = p.batchChanges(params, route53.ChangeActionDelete, payload.Type, dnsName, p.ttl, rs.ResourceRecords)
				// delete ownership-keeper TXT record which keep ownership concept.
				// just print error, no need to handle.
				if err := p.keeper.DeleteOwnership(dnsType, dnsName); err != nil {
					logrus.Debug(err)
				}
			}
		}
	}

	params = p.batchChanges(params, route53.ChangeActionDelete, payload.Type, payload.Domain, p.ttl, rr)

	_, err = p.client.ChangeResourceRecordSets(params)
	if err != nil {
		logrus.WithError(err).Debugf("delete route53 record(s) failed: %s (%s)\n", payload.Domain, payload.Type)
		return output, err
	}

	return output, nil
}

// Renew renew supported records, this only renew records which has prefix with txt-keeper-.
func (p *R53Provider) Renew(payload types.Payload) (types.Result, error) {
	ownership, err := p.keeper.GetOwnership(payload.Type, payload.Domain)
	if err != nil {
		logrus.Debug(err)
		return types.Result{}, err
	}

	if ownership.Domain != keepers.DNSNameToOwnership(payload.Type, payload.Domain) {
		logrus.Debugf("sub-domain can not be renewed: %s (%s)\n", payload.Domain, payload.Type)
		return types.Result{}, err
	}

	records, err := p.Get(payload)
	if err != nil {
		return types.Result{}, err
	}

	for _, rs := range records.Records {
		// renew matched records.
		renew, err := p.keeper.RenewOwnership(rs.Type, rs.Domain, ownership)
		if err != nil {
			logrus.Debug(err)
			return types.Result{}, err
		}

		// renew matched sub-domain records.
		for k := range rs.SubDomains {
			if err := p.keeper.SetSubOwnership(rs.Type, k+"."+payload.Domain, renew); err != nil {
				logrus.Debug(err)
				return types.Result{}, err
			}
		}
	}

	return p.Get(payload)
}

// batchChanges append route53.Change to ChangeResourceRecordSetsInput.ChangeBatch.Changes.
func (p *R53Provider) batchChanges(params *route53.ChangeResourceRecordSetsInput, action, dnsType, dnsName string, ttl int64, rr []*route53.ResourceRecord) *route53.ChangeResourceRecordSetsInput {
	change := &route53.Change{
		Action: aws.String(action),
		ResourceRecordSet: &route53.ResourceRecordSet{
			TTL:             aws.Int64(ttl),
			Type:            aws.String(dnsType),
			Name:            aws.String(utils.WildcardUnescape(dnsName)),
			ResourceRecords: rr,
		},
	}

	params.ChangeBatch.Changes = append(params.ChangeBatch.Changes, change)

	return params
}

// getMatchesRecordSets returns all matches records.
func (p *R53Provider) getMatchesRecordSets(params *route53.ListResourceRecordSetsInput, payload types.Payload) (*route53.ListResourceRecordSetsOutput, error) {
	// returns up to 100 resource record sets at a time in ASCII order.
	// queries outside of more than 100 record sets are not supported.
	rrs, err := p.client.ListResourceRecordSets(params)
	if err != nil {
		logrus.WithError(err).Errorf("get route53 record(s) failed: %s (%s)\n", payload.Domain, payload.Type)
		return rrs, errors.New("get route53 record(s) failed")
	}

	rrs.ResourceRecordSets = filter(rrs.ResourceRecordSets, payload)

	return rrs, nil
}

// assembleResults assemble and convert output format to rdns-server preferred.
func assembleResults(rrs []*route53.ResourceRecordSet, parentName string, isList bool) []types.ResultRecord {
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
			kvs[key].SubDomains[prefix] = valueToString(rs.ResourceRecords, dnsType)
			continue
		}

		if v, ok := kvs[key]; ok {
			v.Value = v.Value + "," + valueToString(rs.ResourceRecords, dnsType)
			v.Value = strings.Trim(v.Value, ",")
			kvs[key] = v
			continue
		}

		r := types.ResultRecord{
			Domain:     dnsName,
			Type:       dnsType,
			Value:      valueToString(rs.ResourceRecords, dnsType),
			SubDomains: map[string]string{},
		}
		kvs[key] = r
	}

	for _, v := range kvs {
		output = append(output, v)
	}

	return output
}

// valueToString returns dns string value.
// See: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/ResourceRecordTypes.html
func valueToString(rr []*route53.ResourceRecord, dnsType string) string {
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

// filter returns the set of records that meet the criteria.
func filter(rrs []*route53.ResourceRecordSet, payload types.Payload) []*route53.ResourceRecordSet {
	output := make([]*route53.ResourceRecordSet, 0)

	for _, rs := range rrs {
		dnsType := aws.StringValue(rs.Type)
		dnsName := utils.GetDNSName(aws.StringValue(rs.Name))

		if strings.HasPrefix(dnsName, keepers.RotatePrefix) || strings.HasPrefix(dnsName, keepers.OwnershipPrefix) || !strings.Contains(dnsName, payload.Domain) {
			continue
		}

		if dnsType == payload.Type {
			if dnsName == payload.Domain {
				output = append(output, rs)
				continue
			}
			// add sub-domain record if exist and valid.
			if keepers.GetKeeper().IsSubDomain(dnsType, dnsName, payload.Type, utils.GetDNSRootName(payload.Domain, payload.Wildcard)) {
				output = append(output, rs)
			}
		}
	}

	return output
}
