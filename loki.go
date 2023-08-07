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
var _ modules.Instance = &Loki{}

type LokiRoot struct {
	vus []vuSpec
}

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
			"Tick":  l.Tick,
		},
	}
}

func (l *Loki) setup(c goja.ConstructorCall) *goja.Object {
	rt := l.vu.Runtime()

	state := l.vu.State()
	if state == nil {
		l.logger.Errorf("unable to get state in Setup")
		return nil
	}
	totalVUs := state.Options.VUs.ValueOrZero()
	if totalVUs == 0 {
		l.logger.Errorf("no vus")
		return nil
	}
	vus := make([]vuSpec, totalVUs)

	// Hardcoded for now, assume percentages are numbers
	// TODO: replace with config from Javascript
	testConfigs := parseTestConfig()
	l.logger.Infof("testConfigs: %v", testConfigs)

	var currentVu int
	for _, tc := range testConfigs {
		for i := 0; i < tc.PercentOfVUs; i++ {
			lc, err := createLokiConfig(currentVu)
			if err != nil {
				l.logger.Errorf("can't create loki client config")
				return nil
			}

			newOne := *newVu(currentVu, lc, tc, l.logger)
			vus[currentVu] = newOne
			l.logger.Infof("Adding vu %+v", newOne)
			currentVu++
		}
	}

	vSpec := vuSpecs{
		vus: vus,
	}

	v := rt.ToValue(vSpec)
	l.logger.Infof("ToValue %+v", v)
	t := v.ToObject(rt)
	// l.logger.Infof("ToObject %+v", t.Export())
	bts, err := t.MarshalJSON()
	if err != nil {
		l.logger.Errorf("ToObject as json error: %v", err)
	} else {
		l.logger.Infof("ToObject as json: %v", string(bts))
	}

	return t
}

func (l *Loki) Tick(data interface{}) (httpext.Response, error) {
	l.logger.Infof("received data: %+v", data)

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
				state.GetScenarioVUIter(),
				state.GetScenarioLocalVUIter(),
				state.GetScenarioGlobalVUIter(),
				totalVUs, // *
				totalIterations,
			)
	*/

	// TODO: send the data for the current vu for 1 second, waiting if needed
	// currentVu := vus[state.VUID]
	// l.logger.Infof("vuSpec: %+v", currentVu)

	// TODO: update the response
	return *httpext.NewResponse(), nil
}
