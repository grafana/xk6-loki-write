package loki

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/dop251/goja"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib/netext/httpext"
	"go.k6.io/k6/metrics"
)

// init is called by the Go runtime at application startup.
func init() {
	modules.Register("k6/x/ngloki", new(LokiRoot))
}

var _ modules.Module = &LokiRoot{}
var _ modules.Instance = &Loki{}

type lokiMetrics struct {
	ClientUncompressedBytes *metrics.Metric
	ClientLines             *metrics.Metric
}

type LokiRoot struct{}

func (*LokiRoot) NewModuleInstance(vu modules.VU) modules.Instance {
	m, err := registerMetrics(vu)
	if err != nil {
		common.Throw(vu.Runtime(), err)
	}

	logger := vu.InitEnv().Logger.WithField("component", "xk6-ngloki")
	return &Loki{vu: vu, metrics: m, logger: logger}
}

func registerMetrics(vu modules.VU) (lokiMetrics, error) {
	var err error
	registry := vu.InitEnv().Registry
	m := lokiMetrics{}

	m.ClientUncompressedBytes, err = registry.NewMetric("loki_client_uncompressed_bytes", metrics.Counter, metrics.Data)
	if err != nil {
		return m, err
	}

	m.ClientLines, err = registry.NewMetric("loki_client_lines", metrics.Counter, metrics.Default)
	if err != nil {
		return m, err
	}

	return m, nil
}

// Loki is the k6 extension that can be imported in the Javascript test file.
type Loki struct {
	vu              modules.VU
	metrics         lokiMetrics
	logger          logrus.FieldLogger
	url             string
	randSeed        int64
	addVuAsTenantID bool
}

func (l *Loki) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"Tick":         l.tick,
			"CreateClient": l.createClient,
			"Stop":         l.stop,
		},
	}
}

func (l *Loki) createClient(obj *goja.Object) {
	rt := l.vu.Runtime()

	if v := obj.Get("randSeed"); !isNully(v) {
		l.randSeed = v.ToInteger()
	}

	if v := obj.Get("url"); !isNully(v) {
		l.url = v.String()
		u, err := url.Parse(l.url)
		if err != nil {
			common.Throw(rt, fmt.Errorf("error parsing url: %v", err))
		}

		_, err = GetClient(u.String(), l.vu, l.metrics, l.randSeed, l.addVuAsTenantID)
		if err != nil {
			common.Throw(rt, fmt.Errorf("error creating client: %v", err))
		}
	}

	if v := obj.Get("addVuAsTenantID"); !isNully(v) {
		l.addVuAsTenantID = v.ToBoolean()
	}
}

type TestConfig struct {
	StaticLabels      model.LabelSet
	ChurningLabels    map[string]int // Churn the string label every int ticks
	Streams           int
	LinesPerSecond    int
	BytesPerSecond    int
	MaxLineSize       int
	RandomLineSizeMin int
	RandomLineSizeMax int
	LogType           string
	TenantID          string
}

func isNully(v goja.Value) bool {
	return v == nil || goja.IsUndefined(v) || goja.IsNull(v)
}

func (l *Loki) parseTestConfigObject(obj *goja.Object, tc *TestConfig) error {
	rt := l.vu.Runtime()

	if v := obj.Get("staticLabels"); !isNully(v) {
		var stringLabels map[string]string
		if err := rt.ExportTo(v, &stringLabels); err != nil {
			return fmt.Errorf("staticLabels should be a map of string to strings: %w", err)
		}

		ls := model.LabelSet{}
		for k, v := range stringLabels {
			ls[model.LabelName(k)] = model.LabelValue(v)
		}
		err := ls.Validate()
		if err != nil {
			return fmt.Errorf("invalid labelset: %w", err)
		}

		tc.StaticLabels = ls
	}

	if v := obj.Get("churningLabels"); !isNully(v) {
		if err := rt.ExportTo(v, &tc.ChurningLabels); err != nil {
			return fmt.Errorf("churningLabels could not be parsed: %w", err)
		}
	}

	if v := obj.Get("streams"); !isNully(v) {
		tc.Streams = int(v.ToInteger())
	}

	if v := obj.Get("linesPerSec"); !isNully(v) {
		tc.LinesPerSecond = int(v.ToInteger())
	}

	if v := obj.Get("bytesPerSec"); !isNully(v) {
		tc.BytesPerSecond = int(v.ToInteger())
	}

	if tc.LinesPerSecond != 0 && tc.BytesPerSecond != 0 {
		return fmt.Errorf("only one of linesPerSec and bytesPerSec can be given")
	}
	if tc.LinesPerSecond == 0 && tc.BytesPerSecond == 0 {
		return fmt.Errorf("one of linesPerSec and bytesPerSec has to be given")
	}

	if v := obj.Get("maxLineSize"); !isNully(v) {
		tc.MaxLineSize = int(v.ToInteger())
	}

	if v := obj.Get("randomLineSizeMin"); !isNully(v) {
		tc.RandomLineSizeMin = int(v.ToInteger())
	}

	if v := obj.Get("randomLineSizeMax"); !isNully(v) {
		tc.RandomLineSizeMax = int(v.ToInteger())
	}

	if v := obj.Get("logType"); !isNully(v) {
		tc.LogType = v.String()

		switch tc.LogType {
		case "apache_common", "apache_combined", "apache_error", "rfc3164", "rfc5424", "common_log", "json", "logfmt":
		default:
			return fmt.Errorf("invalid logtype %v", tc.LogType)
		}
	} else {
		tc.LogType = "logfmt"
	}

	if v := obj.Get("tenantID"); !isNully(v) {
		tc.TenantID = v.String()
	}

	return nil
}

func (l *Loki) tick(obj *goja.Object) (httpext.Response, error) {
	started := time.Now()
	oneSecAfterStarting := started.Add(time.Second)
	tc := TestConfig{}
	err := l.parseTestConfigObject(obj, &tc)
	if err != nil {
		return *httpext.NewResponse(), err
	}
	l.logger.Debugf("received data: %+v", tc)

	state := l.vu.State()
	if state == nil {
		return *httpext.NewResponse(), errors.New("state is nil")
	}

	client, err := GetClient(l.url, l.vu, l.metrics, l.randSeed, l.addVuAsTenantID)
	if err != nil {
		return *httpext.NewResponse(), err
	}

	err = client.GenerateLogs(&tc, state, l.logger)
	if err != nil {
		return *httpext.NewResponse(), err
	}

	// Wait the remainder of the 1 second we can take
	timeLeft := oneSecAfterStarting.Sub(started)
	if timeLeft < 0 {
		return *httpext.NewResponse(), nil
	}

	t := time.NewTimer(timeLeft)
	defer t.Stop()
	<-t.C
	return *httpext.NewResponse(), nil
}

func (l *Loki) stop() (httpext.Response, error) {
	client, err := GetClient(l.url, l.vu, l.metrics, l.randSeed, l.addVuAsTenantID)
	if err != nil {
		return *httpext.NewResponse(), err
	}

	client.Stop()

	return *httpext.NewResponse(), nil
}
