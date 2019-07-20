package aws

import (
	"os"
	"strings"
	"time"

	"github.com/rancher/rdns-server/keepers"
	"github.com/rancher/rdns-server/types"
	"github.com/rancher/rdns-server/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	keeperName  = "route53"
	tokenLength = 32
)

type R53Keeper struct {
	client   *route53.Route53
	domain   string
	name     string
	ttl      int64
	expire   int64
	rotate   int64
	zoneName string
	zoneID   string
}

// NewRout53Keeper returns AWS Route53 Keeper object.
func NewRout53Keeper(client *route53.Route53, zoneName, zoneID string, ttl int64) *R53Keeper {
	rd, err := time.ParseDuration(os.Getenv("ROTATE"))
	if err != nil {
		logrus.WithError(err).Errorln("failed to parse rotate")
		return &R53Keeper{}
	}

	ed, err := time.ParseDuration(os.Getenv("EXPIRE"))
	if err != nil {
		logrus.WithError(err).Errorln("failed to parse expire")
		return &R53Keeper{}
	}

	return &R53Keeper{
		client:   client,
		domain:   zoneName,
		name:     keeperName,
		ttl:      ttl,
		expire:   int64(ed.Seconds()),
		rotate:   int64(rd.Seconds()),
		zoneName: zoneName,
		zoneID:   zoneID,
	}
}

func (k *R53Keeper) NameCanUse(dnsType, dnsName string) (bool, error) {
	rotateName := keepers.DNSNameToRotate(dnsType, dnsName)

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(k.zoneID),
		StartRecordType: aws.String(types.RecordTypeTXT),
		StartRecordName: aws.String(rotateName),
	}

	// returns up to 100 resource record sets at a time in ASCII order.
	// queries outside of more than 100 record sets are not supported.
	rrs, err := k.client.ListResourceRecordSets(params)
	if err != nil {
		return false, errors.Wrapf(err, "name can not be used: %s (%s)\n", rotateName, types.RecordTypeTXT)
	}

	for _, rs := range rrs.ResourceRecordSets {
		if utils.GetDNSName(aws.StringValue(rs.Name)) == rotateName {
			return false, nil
		}
	}

	return true, nil
}

func (k *R53Keeper) SetRotate(dnsType, dnsName string) (keepers.Keep, error) {
	output := keepers.NewTXT(dnsType, dnsName, "", k.rotate, false)

	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(k.zoneID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{},
		},
	}

	change := &route53.Change{
		Action: aws.String(route53.ChangeActionUpsert),
		ResourceRecordSet: &route53.ResourceRecordSet{
			TTL:  aws.Int64(k.ttl),
			Type: aws.String(types.RecordTypeTXT),
			Name: aws.String(output.Domain),
			ResourceRecords: []*route53.ResourceRecord{
				{
					Value: aws.String(keepers.ToDNSValue(output)),
				},
			},
		},
	}

	params.ChangeBatch.Changes = append(params.ChangeBatch.Changes, change)

	_, err := k.client.ChangeResourceRecordSets(params)
	if err != nil {
		return output, errors.Wrapf(err, "set route53 rotate record(s) failed: %s (%s)\n", dnsName, dnsType)
	}

	return output, nil
}

func (k *R53Keeper) GetOwnership(dnsType, dnsName string) (keepers.Keep, error) {
	keeperName := keepers.DNSNameToOwnership(dnsType, dnsName)

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(k.zoneID),
		StartRecordType: aws.String(types.RecordTypeTXT),
		StartRecordName: aws.String(keeperName),
	}

	// returns up to 100 resource record sets at a time in ASCII order.
	// queries outside of more than 100 record sets are not supported.
	rrs, err := k.client.ListResourceRecordSets(params)
	if err != nil {
		return keepers.Keep{}, errors.Wrapf(err, "get route53 ownership record(s) failed: %s (%s)\n", keeperName, types.RecordTypeTXT)
	}

	for _, rs := range rrs.ResourceRecordSets {
		if utils.GetDNSName(aws.StringValue(rs.Name)) == keeperName {
			if len(rs.ResourceRecords) < 1 {
				continue
			}
			ss := strings.Split(utils.TextRemoveQuotes(aws.StringValue(rs.ResourceRecords[0].Value)), ",")
			return keepers.Keep{
				Domain: ss[0],
				Type:   strings.ToLower(ss[1]),
				Token:  ss[2],
				Expire: ss[3],
			}, nil
		}
	}

	return keepers.Keep{}, errors.Errorf("no exist route53 ownership record(s): %s (%s)\n", keeperName, types.RecordTypeTXT)
}

func (k *R53Keeper) SetOwnership(dnsType, dnsName string) (keepers.Keep, error) {
	output := keepers.NewTXT(dnsType, dnsName, utils.RandStringWithAll(tokenLength), k.expire, true)

	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(k.zoneID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{},
		},
	}

	change := &route53.Change{
		Action: aws.String(route53.ChangeActionUpsert),
		ResourceRecordSet: &route53.ResourceRecordSet{
			TTL:  aws.Int64(k.ttl),
			Type: aws.String(types.RecordTypeTXT),
			Name: aws.String(output.Domain),
			ResourceRecords: []*route53.ResourceRecord{
				{
					Value: aws.String(keepers.ToDNSValue(output)),
				},
			},
		},
	}

	params.ChangeBatch.Changes = append(params.ChangeBatch.Changes, change)

	_, err := k.client.ChangeResourceRecordSets(params)
	if err != nil {
		return output, errors.Wrapf(err, "set route53 ownership record(s) failed: %s (%s)\n", dnsName, dnsType)
	}

	return output, nil
}

func (k *R53Keeper) SetSubOwnership(dnsType, dnsName string, keep keepers.Keep) error {
	keeperName := keepers.DNSNameToOwnership(dnsType, dnsName)

	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(k.zoneID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{},
		},
	}

	change := &route53.Change{
		Action: aws.String(route53.ChangeActionUpsert),
		ResourceRecordSet: &route53.ResourceRecordSet{
			TTL:  aws.Int64(k.ttl),
			Type: aws.String(types.RecordTypeTXT),
			Name: aws.String(keeperName),
			ResourceRecords: []*route53.ResourceRecord{
				{
					Value: aws.String(keepers.ToDNSValue(keep)),
				},
			},
		},
	}

	params.ChangeBatch.Changes = append(params.ChangeBatch.Changes, change)

	_, err := k.client.ChangeResourceRecordSets(params)
	if err != nil {
		return errors.Wrapf(err, "set route53 sub-domain ownership record(s) failed: %s (%s)\n", dnsName, dnsType)
	}

	return nil
}

func (k *R53Keeper) DeleteOwnership(dnsType, dnsName string) error {
	keeperName := keepers.DNSNameToOwnership(dnsType, dnsName)

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(k.zoneID),
		StartRecordType: aws.String(types.RecordTypeTXT),
		StartRecordName: aws.String(keeperName),
	}

	// returns up to 100 resource record sets at a time in ASCII order.
	// queries outside of more than 100 record sets are not supported.
	rrs, err := k.client.ListResourceRecordSets(params)
	if err != nil {
		return errors.Wrapf(err, "delete route53 ownership record(s) failed: %s (%s)\n", keeperName, types.RecordTypeTXT)
	}

	for _, rs := range rrs.ResourceRecordSets {
		if utils.GetDNSName(aws.StringValue(rs.Name)) == keeperName {
			_, err = k.client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
				HostedZoneId: aws.String(k.zoneID),
				ChangeBatch: &route53.ChangeBatch{
					Changes: []*route53.Change{
						{
							Action:            aws.String(route53.ChangeActionDelete),
							ResourceRecordSet: rs,
						},
					},
				},
			})
			if err != nil {
				return errors.Wrapf(err, "delete route53 ownership record(s) failed: %s (%s)\n", keeperName, types.RecordTypeTXT)
			}
		}
	}

	return nil
}

func (k *R53Keeper) RenewOwnership(dnsType, dnsName string, keep keepers.Keep) (keepers.Keep, error) {
	keeperName := keepers.DNSNameToOwnership(dnsType, dnsName)

	renew := keepers.RenewTXT(keep, k.expire)

	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(k.zoneID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{},
		},
	}

	change := &route53.Change{
		Action: aws.String(route53.ChangeActionUpsert),
		ResourceRecordSet: &route53.ResourceRecordSet{
			TTL:  aws.Int64(k.ttl),
			Type: aws.String(types.RecordTypeTXT),
			Name: aws.String(keeperName),
			ResourceRecords: []*route53.ResourceRecord{
				{
					Value: aws.String(keepers.ToDNSValue(renew)),
				},
			},
		},
	}

	params.ChangeBatch.Changes = append(params.ChangeBatch.Changes, change)

	_, err := k.client.ChangeResourceRecordSets(params)
	if err != nil {
		return keepers.Keep{}, errors.Wrapf(err, "renew route53 ownership record(s) failed: %s (%s)\n", dnsName, dnsType)
	}

	return renew, nil
}

func (k *R53Keeper) IsSubDomain(dnsType, dnsName, parentType, parentName string) bool {
	ownershipName := keepers.DNSNameToOwnership(dnsType, dnsName)
	upperOwnershipName := keepers.DNSNameToOwnership(parentType, parentName)

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(k.zoneID),
		StartRecordType: aws.String(types.RecordTypeTXT),
		StartRecordName: aws.String(ownershipName),
	}

	// returns up to 100 resource record sets at a time in ASCII order.
	// queries outside of more than 100 record sets are not supported.
	rrs, err := k.client.ListResourceRecordSets(params)
	if err != nil {
		logrus.WithError(err).Debugf("can not determine whether it is sub-domain: %s (%s)\n", ownershipName, types.RecordTypeTXT)
		return false
	}

	if len(rrs.ResourceRecordSets) > 0 && utils.GetDNSName(aws.StringValue(rrs.ResourceRecordSets[0].Name)) == ownershipName {
		ss := strings.Split(utils.TextRemoveQuotes(aws.StringValue(rrs.ResourceRecordSets[0].ResourceRecords[0].Value)), ",")
		if ss[0] == upperOwnershipName {
			return true
		}
	}

	return false
}
