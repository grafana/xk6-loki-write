package loki

import (
	"errors"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib/netext/httpext"
)

// init is called by the Go runtime at application startup.
func init() {
	modules.Register("k6/x/ngloki", new(LokiRoot))
}

var _ modules.Module = &LokiRoot{}

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
			"Setup": l.setup,
			"Tick":  l.tick,
		},
	}
}

type Config struct{}

func (l *Loki) setup(c goja.ConstructorCall) *goja.Object {
	rt := l.vu.Runtime()

	config := &Config{}
	return rt.ToValue(config).ToObject(rt)
}

func (l *Loki) tick() (httpext.Response, error) {
	state := l.vu.State()
	if state == nil {
		return *httpext.NewResponse(), errors.New("state is nil")
	}
	totalVUs := state.Options.VUs.ValueOrZero()
	totalIterations := state.Options.Iterations.ValueOrZero()

	l.logger.Infof(
		"VUId: %v, VUIDGlobal: %v, Scenario Iter: %v, Scenario Local Iter: %v, Scenario Glocal Iter: %v, total VUs: %v, totalOperations: %v",
		state.VUID,
		state.VUIDGlobal,
		state.GetScenarioVUIter(),
		state.GetScenarioLocalVUIter(),
		state.GetScenarioGlobalVUIter(),
		totalVUs,
		totalIterations,
	)

	// TODO: send the data for the current vu for 1 second

	// TODO: update the response
	return *httpext.NewResponse(), nil
}
