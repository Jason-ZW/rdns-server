package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rancher/rdns-server/model"
	"github.com/rancher/rdns-server/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
)

const (
	TTL              = "240h"
	Name             = "etcdv3"
	frozenPath       = "/frozenv3"
	tokenPath        = "/tokenv3"
	tokenLength      = 32
	slugLength       = 6
	maxSlugHashTimes = 100
)

type Backend struct {
	ClientV3 *clientv3.Client
	duration time.Duration
	frozen   string
	path     string
	domain   string
}

func NewBackend(endpoints []string, path, domain, frozen string) (*Backend, error) {
	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	}
	c, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	d, err := time.ParseDuration(TTL)
	if err != nil {
		return nil, err
	}
	return &Backend{c, d, frozen, path, domain}, nil
}

func (b *Backend) Get(opts *model.DomainOptions) (d model.Domain, err error) {
	logrus.Debugf("Get A record for domain options: %s", opts.String())

	path := getPath(b.path, opts.Fqdn)

	// lookup all keys under the path
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := b.ClientV3.Get(ctx, path, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return d, errors.Wrapf(err, "Failed to lookup keys under the path: %s", path)
	}

	var lease int64
	hosts := make([]string, 0)
	for _, kv := range resp.Kvs {
		lease = kv.Lease
		v, err := unmarshalToMap(kv.Value)
		if err != nil {
			return d, err
		}
		hosts = append(hosts, v["host"])
	}

	// get lease from lease ID
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	leaseResp, err := b.ClientV3.TimeToLive(ctx, clientv3.LeaseID(lease))
	cancel()
	if err != nil {
		return d, err
	}

	// TODO: sub-domain logic will be added

	d.Fqdn = opts.Fqdn
	d.Hosts = hosts
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", leaseResp.TTL))
	e := time.Now().Add(duration)
	d.Expiration = &e

	return d, nil
}

func (b *Backend) Set(opts *model.DomainOptions) (d model.Domain, err error) {
	logrus.Debugf("Set A record for domain options: %s", opts.String())

	var path string
	for i := 0; i < maxSlugHashTimes; i++ {
		fqdn := fmt.Sprintf("%s.%s", generateSlug(), b.domain)

		// check whether this fqdn can be used or not
		if b.checkFrozen(fqdn) {
			logrus.Debugf("Failed to use fqdn: %s because it is frozen, will try another", fqdn)
			continue
		}

		path = getPath(b.path, fqdn)

		// check whether this path can be used or not
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resp, err := b.ClientV3.Get(ctx, path)
		cancel()
		if err != nil || resp.Count <= 0 {
			opts.Fqdn = fqdn
			break
		}
	}

	// set A record to etcd
	d, err = b.setRecordA(path, opts, false)
	if err != nil {
		return d, err
	}

	// set frozen for this fqdn, used to check whether fqdn can be issued again later
	if err := b.setFrozen(opts, false); err != nil {
		return d, err
	}

	return b.Get(opts)
}

func (b *Backend) Update(opts *model.DomainOptions) (d model.Domain, err error) {
	logrus.Debugf("Update A record for domain options: %s", opts.String())

	path := getPath(b.path, opts.Fqdn)

	// get A record
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := b.ClientV3.Get(ctx, path, clientv3.WithPrefix())
	cancel()
	if err != nil || resp.Count <= 0 {
		return d, errors.Wrapf(err, "Failed to update A record for %s", path)
	}

	d, err = b.setRecordA(path, opts, true)
	if err != nil {
		return d, errors.Wrapf(err, "Failed to update A record for %s", path)
	}

	return d, b.setFrozen(opts, true)
}

func (b *Backend) Delete(opts *model.DomainOptions) error {
	logrus.Debugf("Delete A record for domain options: %s", opts.String())

	path := getPath(b.path, opts.Fqdn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, err := b.ClientV3.Delete(ctx, path, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return errors.Wrapf(err, "Failed to delete A record for %s", path)
	}

	// TODO: sub-domain logic will be deleted

	return nil
}

func (b *Backend) Renew(opts *model.DomainOptions) (d model.Domain, err error) {
	logrus.Debugf("Renew for domain options: %s", opts.String())

	path := getPath(b.path, opts.Fqdn)

	leaseID, leaseTTL, err := b.setToken(opts, true)
	if err != nil {
		return d, errors.Wrapf(err, "Failed to set token for %s", path)
	}

	// keep-alive once for lease's ID
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	keepalive, err := b.ClientV3.KeepAliveOnce(ctx, clientv3.LeaseID(leaseID))
	cancel()
	if err != nil {
		return d, errors.Wrapf(err, "Failed to keep-alive-once for lease: %s", leaseID)
	}

	leaseTTL = keepalive.TTL

	// lookup all keys under the path
	hosts, err := b.lookupKeys(path)
	if err != nil {
		return d, err
	}

	// TODO: sub-domain logic will be added

	// TODO: acme-text logic will be added

	d.Fqdn = opts.Fqdn
	d.Hosts = hosts
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", leaseTTL))
	e := time.Now().Add(duration)
	d.Expiration = &e

	return d, b.setFrozen(opts, true)
}

func (b *Backend) SetText(opts *model.DomainOptions) (d model.Domain, err error) {
	return model.Domain{}, nil
}

func (b *Backend) GetText(opts *model.DomainOptions) (d model.Domain, err error) {
	return model.Domain{}, nil
}

func (b *Backend) UpdateText(opts *model.DomainOptions) (d model.Domain, err error) {
	return model.Domain{}, nil
}

func (b *Backend) DeleteText(opts *model.DomainOptions) error {
	return nil
}

func (b *Backend) GetToken(fqdn string) (string, error) {
	logrus.Debugf("Get token for fqdn: %s", fqdn)

	path := getTokenPath(fqdn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := b.ClientV3.Get(ctx, path)
	cancel()
	if err != nil || resp.Count <= 0 {
		return "", errors.Wrapf(err, "Not found key: %s", path)
	}

	return string(resp.Kvs[0].Value), nil
}

func (b *Backend) setRecordA(path string, opts *model.DomainOptions, exist bool) (d model.Domain, err error) {
	leaseID, leaseTTL, err := b.setToken(opts, exist)
	if err != nil {
		return d, err
	}

	// lookup all keys under the path
	hosts, err := b.lookupKeys(path)
	if err != nil {
		return d, err
	}

	// sync records
	if err := b.syncRecords(opts.Hosts, hosts, path, clientv3.LeaseID(leaseID)); err != nil {
		return d, errors.Wrapf(err, "Failed to sync keys under the path: %s", path)
	}

	d.Fqdn = opts.Fqdn
	d.Hosts = opts.Hosts
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", leaseTTL))
	e := time.Now().Add(duration)
	d.Expiration = &e

	// TODO: sub-domain logic will be added

	return d, nil
}

// Used to lookup all keys under the path
func (b *Backend) lookupKeys(path string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := b.ClientV3.Get(ctx, path, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return []string{}, errors.Wrapf(err, "Failed to lookup keys under the path: %s", path)
	}

	hosts := make([]string, 0)
	for _, kv := range resp.Kvs {
		v, err := unmarshalToMap(kv.Value)
		if err != nil {
			return []string{}, err
		}
		hosts = append(hosts, v["host"])
	}

	return hosts, nil
}

// Used to sync to new records and remove useless keys
func (b *Backend) syncRecords(new, old []string, path string, leaseID clientv3.LeaseID) error {
	left := sliceToMap(new)
	right := sliceToMap(old)

	for r := range right {
		if _, ok := left[r]; !ok {
			key := fmt.Sprintf("%s/%s", path, formatKey(r))
			// delete useless key
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err := b.ClientV3.Delete(ctx, key)
			cancel()
			if err != nil {
				return err
			}
		}
	}

	for l := range left {
		if _, ok := right[l]; !ok {
			key := fmt.Sprintf("%s/%s", path, formatKey(l))
			// set new key
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err := b.ClientV3.Put(ctx, key, formatValue(l), clientv3.WithLease(leaseID))
			cancel()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *Backend) setToken(opts *model.DomainOptions, exist bool) (int64, int64, error) {
	logrus.Debugf("Set token for fqdn: %s", opts.Fqdn)

	var token string
	var leaseID int64
	var leaseTTL int64

	path := getTokenPath(opts.Fqdn)

	if exist {
		// get token and lease's ID
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resp, err := b.ClientV3.Get(ctx, path)
		cancel()
		if err != nil || resp.Count <= 0 {
			return 0, -1, errors.Wrapf(err, "Not found previous key: %s", path)
		}

		token = string(resp.Kvs[0].Value)
		l := resp.Kvs[0].Lease

		// get lease with leaseID
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		lease, err := b.ClientV3.TimeToLive(ctx, clientv3.LeaseID(l))
		cancel()
		if err != nil {
			return 0, -1, errors.Wrapf(err, "Failed to get lease for %s", path)
		}
		leaseID = int64(lease.ID)
		leaseTTL = lease.TTL
	} else {
		token = util.RandStringWithAll(tokenLength)

		// create lease with duration
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		lease, err := b.ClientV3.Grant(ctx, int64(b.duration.Seconds()))
		cancel()
		if err != nil {
			return 0, -1, errors.Wrapf(err, "Failed to grant lease: %s", path)
		}
		leaseID = int64(lease.ID)
		leaseTTL = lease.TTL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, err := b.ClientV3.Put(ctx, path, token, clientv3.WithLease(clientv3.LeaseID(leaseID)))
	cancel()

	return leaseID, leaseTTL, errors.Wrapf(err, "Failed to set key: %s", path)
}

// Used to set frozen record which will determine whether fqdn can be issued again
// e.g. sample.lb.rancher.cloud => /frozenv3/sample
func (b *Backend) setFrozen(opts *model.DomainOptions, exist bool) error {
	logrus.Debugf("Set frozen for fqdn: %s", opts.Fqdn)

	duration, err := time.ParseDuration(b.frozen)
	if err != nil {
		return err
	}

	ss := strings.SplitN(opts.Fqdn, ".", 2)
	path := fmt.Sprintf("%s%s/%s", b.path, frozenPath, ss[0])

	var leaseID int64

	if exist {
		// get leaseID
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resp, err := b.ClientV3.Get(ctx, path, clientv3.WithPrefix())
		cancel()
		if err != nil {
			return errors.Wrapf(err, "Failed to lookup keys under the path: %s", path)
		}
		for _, kv := range resp.Kvs {
			leaseID = kv.Lease
		}
		// get lease with ID
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		lease, err := b.ClientV3.TimeToLive(ctx, clientv3.LeaseID(leaseID))
		cancel()
		if err != nil {
			return errors.Wrapf(err, "Failed to get lease for %s", path)
		}
		leaseID = int64(lease.ID)
	} else {
		// create lease with frozen time
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		lease, err := b.ClientV3.Grant(ctx, int64(duration.Seconds()))
		cancel()
		if err != nil {
			return errors.Wrapf(err, "Failed to set lease for %s", path)
		}
		leaseID = int64(lease.ID)
	}

	// create frozen path with lease
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = b.ClientV3.Put(ctx, path, "", clientv3.WithLease(clientv3.LeaseID(leaseID)))
	cancel()
	if err != nil {
		return errors.Wrapf(err, "Failed to set key %s with lease %d", path, clientv3.LeaseID(leaseID))
	}

	return nil
}

// Used to check whether fqdn can be used.
// e.g. sample.lb.rancher.cloud => /frozenv3/sample
// e.g. if /frozenv3/sample is exist that fqdn can not be used
func (b *Backend) checkFrozen(fqdn string) bool {
	ss := strings.SplitN(fqdn, ".", 2)
	path := fmt.Sprintf("%s%s/%s", b.path, frozenPath, ss[0])

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := b.ClientV3.Get(ctx, path)
	cancel()
	if err != nil || resp.Count <= 0 {
		return false
	}
	return true
}

// Used to get a path as etcd preferred
// e.g. sample.lb.rancher.cloud => /rdnsv3/cloud/rancher/lb/sample
func getPath(path, domain string) string {
	return path + convertToPath(domain)
}

// Used to get a token path as etcd preferred
// e.g. sample.lb.rancher.cloud => /tokenv3/sample_lb_rancher_cloud
func getTokenPath(domain string) string {
	return fmt.Sprintf("%s/%s", tokenPath, formatKey(domain))
}

// Used to format a key as etcd preferred
// e.g. 1.1.1.1 => 1_1_1_1
// e.g. sample.lb.rancher.cloud => sample_lb_rancher_cloud
func formatKey(key string) string {
	return strings.Replace(key, ".", "_", -1)
}

// Used to format a value as dns preferred
// e.g. 1.1.1.1 => {"host": "1.1.1.1"}
func formatValue(value string) string {
	return fmt.Sprintf("{\"host\":\"%s\"}", value)
}

// Used to generate a random slug
func generateSlug() string {
	return util.RandStringWithSmall(slugLength)
}

// Used to convert domain to a path as etcd preferred
// e.g. sample.lb.rancher.cloud => /cloud/rancher/lb/sample
func convertToPath(domain string) string {
	ss := strings.Split(domain, ".")
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
	return "/" + strings.Join(ss, "/")
}

func unmarshalToMap(b []byte) (map[string]string, error) {
	var v map[string]string
	err := json.Unmarshal(b, &v)
	return v, err
}

func sliceToMap(ss []string) map[string]bool {
	m := make(map[string]bool)
	for _, s := range ss {
		m[s] = true
	}
	return m
}
