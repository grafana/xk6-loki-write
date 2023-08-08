package loki

import (
	"errors"
	"fmt"

	"github.com/dop251/goja"
	"github.com/prometheus/tsdb/labels"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib/netext/httpext"
)

// init is called by the Go runtime at application startup.
func init() {
	modules.Register("k6/x/ngloki", new(LokiRoot))
}

var _ modules.Module = &LokiRoot{}
var _ modules.Instance = &Loki{}

type LokiRoot struct{}

func (*LokiRoot) NewModuleInstance(vu modules.VU) modules.Instance {
	logger := vu.InitEnv().Logger.WithField("component", "xk6-ngloki")
	return &Loki{vu: vu, logger: logger}
}

// Loki is the k6 extension that can be imported in the Javascript test file.
type Loki struct {
	vu     modules.VU
	logger logrus.FieldLogger
}

func (l *Loki) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"Tick": l.Tick,
		},
	}
}

type TestConfig struct {
	StaticLabels labels.Labels
	LineSize     int
	BytesPerLine int
	Frequency    int
}

func isNully(v goja.Value) bool {
	return v == nil || goja.IsUndefined(v) || goja.IsNull(v)
}

func (l *Loki) parseTestConfigObject(obj *goja.Object, tc *TestConfig) error {
	rt := l.vu.Runtime()

	if v := obj.Get("lines"); !isNully(v) {
		tc.LineSize = int(v.ToInteger())
	}

	if v := obj.Get("bytes"); !isNully(v) {
		tc.BytesPerLine = int(v.ToInteger())
	}

	if v := obj.Get("frequency"); !isNully(v) {
		tc.Frequency = int(v.ToInteger())
	}

	if v := obj.Get("staticLabels"); !isNully(v) {
		var stringLabels map[string]string
		if err := rt.ExportTo(v, &stringLabels); err != nil {
			return fmt.Errorf("staticLabels should be a map of string to strings: %w", err)
		}

		lbls := labels.FromMap(stringLabels)
		tc.StaticLabels = lbls
	}

	return nil
}

func (l *Loki) Tick(obj *goja.Object) (httpext.Response, error) {
	tc := TestConfig{}
	err := l.parseTestConfigObject(obj, &tc)
	if err != nil {
		return *httpext.NewResponse(), err
	}
	l.logger.Infof("received data: %+v", tc)

	state := l.vu.State()
	if state == nil {
		return *httpext.NewResponse(), errors.New("state is nil")
	}
	/*
		totalVUs := state.Options.VUs.ValueOrZero()
		totalIterations := state.Options.Iterations.ValueOrZero()

		l.logger.Infof(
			"VUId: %v, VUIDGlobal: %v, Scenario Iter: %v, Scenario Local Iter: %v, Scenario Glocal Iter: %v, total VUs: %v, totalOperations: %v",
			state.VUID, // *
			state.VUIDGlobal,
			state.GetScenarioVUIter(), // *
			state.GetScenarioLocalVUIter(),
			state.GetScenarioGlobalVUIter(),
			totalVUs, // *
			totalIterations,
		)
	*/

	/*
		// TODO: send the data for the current vu for 1 second, waiting if needed
		currentVu := specs.Vus[state.VUID]
		l.logger.Infof("vuSpec: %+v", currentVu)
	*/

	// TODO: update the response
	return *httpext.NewResponse(), nil
}
