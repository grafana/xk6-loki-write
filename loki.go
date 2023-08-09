package loki

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/dop251/goja"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/common"
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
	vu       modules.VU
	logger   logrus.FieldLogger
	url      string
	randSeed int64
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

		_, err = GetClient(u.String(), l.randSeed)
		if err != nil {
			common.Throw(rt, fmt.Errorf("error creating client: %v", err))
		}

	}
}

type TestConfig struct {
	StaticLabels model.LabelSet
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
	/*
		totalVUs := state.Options.VUs.ValueOrZero()
		totalIterations := state.Options.Iterations.ValueOrZero()

		l.logger.Infof(
			"VUId: %v, VUIDGlobal: %v, Scenario Iter: %v, Scenario Local Iter: %v, Scenario Glocal Iter: %v, total VUs: %v, totalOperations: %v",
			state.VUID, // * Use this
			state.VUIDGlobal,
			state.GetScenarioVUIter(), // *
			state.GetScenarioLocalVUIter(),
			state.GetScenarioGlobalVUIter(),
			totalVUs, // *
			totalIterations,
		)
	*/

	client, err := GetClient(l.url, l.randSeed)
	if err != nil {
		return *httpext.NewResponse(), err
	}

	// Add a vuid label
	tc.StaticLabels[model.LabelName("vuid")] = model.LabelValue(strconv.Itoa(int(state.VUID)))

	err = client.GenerateLogs(&tc)
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
	client, err := GetClient(l.url, l.randSeed)
	if err != nil {
		return *httpext.NewResponse(), err
	}

	client.Stop()

	return *httpext.NewResponse(), nil
}
