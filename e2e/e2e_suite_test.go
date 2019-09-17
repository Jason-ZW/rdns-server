package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	client  *resty.Client
	domains map[string]string
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
		response, err := client.R().SetAuthToken(v).Delete("/v1/domain/" + k)
		if err != nil {
			logrus.Errorf("failed to cleanup e2e record(s): %v", k)
		}

		if response.StatusCode() != http.StatusOK {
			logrus.Errorf("failed to cleanup e2e record(s): %v", response.Request.URL)
		}

		delete(domains, k)
	}
	client = nil
})
