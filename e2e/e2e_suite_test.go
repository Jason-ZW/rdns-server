package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/rancher/rdns-server/types"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	client  *resty.Client
	domains map[string]map[string]string
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	client = resty.New()
	client.
		// set retry count to non zero to enable retries.
		SetRetryCount(3).
		// default is 100 milliseconds.
		SetRetryWaitTime(5 * time.Second).
		// default is 2 seconds.
		SetRetryMaxWaitTime(20 * time.Second).
		// headers for all request
		SetHeaders(map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		}).
		// set callback to calculate wait time between retries.
		SetRetryAfter(func(client *resty.Client, resp *resty.Response) (time.Duration, error) {
			return 0, errors.New("retry exceeded")
		})
})

var _ = AfterSuite(func() {
	// clean up all e2e records.
	for k, v := range domains {
		for kk, vv := range v {
			url := "/v1/domain/" + kk
			switch k {
			case types.RecordTypeA:
				url = url + "?cleanup=true"
			case types.RecordTypeAAAA:
				url = url + "?cleanup=true&type=AAAA"
			case types.RecordTypeCNAME:
				url = url + "/cname?cleanup=true"
			case types.RecordTypeTXT:
				url = url + "/txt?cleanup=true"
			default:
				logrus.Errorln("unsupported dns types")
			}
			response, err := client.R().SetAuthToken(vv).Delete(url)
			if err != nil {
				logrus.Errorf("failed to cleanup e2e record(s): %v", kk)
			}
			if response.StatusCode() != http.StatusOK {
				logrus.Errorf("failed to cleanup e2e record(s): %v", response.Request.URL)
			}
		}
	}
	client = nil
})
