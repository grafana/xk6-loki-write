package loki

import (
	"github.com/prometheus/common/model"
)

type testConfig struct {
	StaticLabels model.LabelSet
	PercentOfVUs int
	LineSize     int
	BytesPerLine int
}

func parseTestConfig() []testConfig {
	// TODO: check percentages match up
	return []testConfig{
		{
			StaticLabels: model.LabelSet{"namespace": "loki-prod-001", "source": "kafka"},
			PercentOfVUs: 1,
			LineSize:     100,
			BytesPerLine: 2000,
		},
		{
			StaticLabels: model.LabelSet{"namespace": "loki-prod-001", "container": "distributor"},
			PercentOfVUs: 1,
			LineSize:     100,
			BytesPerLine: 2000,
		},
	}
}
