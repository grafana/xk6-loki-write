package loki

import (
	"strconv"

	lokiClient "github.com/grafana/loki-client-go/loki"
	"github.com/sirupsen/logrus"
)

type vuSpecs struct {
	vus []vuSpec
}

type vuSpec struct {
	// clientConfig *lokiClient.Config
	vuNr int
	tc   testConfig
	// logger       logrus.FieldLogger
}

func newVu(vuNr int, clientConfig *lokiClient.Config, testConfig testConfig, logger logrus.FieldLogger) *vuSpec {
	return &vuSpec{
		vuNr: vuNr,
		// clientConfig: clientConfig,
		tc: testConfig,
		// logger:       logger,
	}
}

func createLokiConfig(tenantID int) (*lokiClient.Config, error) {
	lc, err := lokiClient.NewDefaultConfig("")
	if err != nil {
		return nil, err
	}

	lc.TenantID = "tenant-" + strconv.Itoa(tenantID)
	return &lc, nil
}
