package loki

import (
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	lokiClient "github.com/grafana/loki-client-go/loki"
	"github.com/grafana/xk6-loki/flog"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/lib"
)

var once sync.Once
var instance *lokiClient.Client

type Client struct {
	instance *lokiClient.Client
	flog     *flog.Flog
}

func GetClient(url string, randSeed int64) (*Client, error) {
	var err error
	once.Do(func() {
		instance, err = lokiClient.NewWithDefault(url)
	})
	if err != nil {
		return nil, err
	}

	rand := rand.New(rand.NewSource(randSeed))
	faker := gofakeit.NewCustom(rand)

	flog := flog.New(rand, faker)

	return &Client{instance: instance, flog: flog}, nil
}

func (c *Client) GenerateLogs(tc *TestConfig, state *lib.State, logger logrus.FieldLogger) error {
	lbls := tc.StaticLabels.Clone()
	lbls[model.LabelName("vuid")] = model.LabelValue(strconv.Itoa(int(state.VUID)))
	for churnLabelKey, churnLabelValue := range tc.ChurningLabels {
		quotient := state.GetScenarioVUIter() / uint64(churnLabelValue)
		lbls[model.LabelName(churnLabelKey)] = model.LabelValue(strconv.Itoa(int(quotient)))
	}

	if tc.LinesPerSecond != 0 {
		for i := 0; i < tc.LinesPerSecond; i++ {
			now := time.Now()
			logLine := c.flog.LogLine(tc.LogType, now)
			if tc.MaxLineSize != 0 {
				logLine = logLine[:tc.MaxLineSize]
			}
			c.instance.Handle(lbls, now, logLine)
		}
	}

	if tc.BytesPerSecond != 0 {
		currentSize := 0
		for {
			now := time.Now()
			logLine := c.flog.LogLine(tc.LogType, now)
			if tc.MaxLineSize != 0 {
				logLine = logLine[:tc.MaxLineSize]
			}
			currentSize += len(logLine)
			if currentSize > tc.BytesPerSecond {
				remainder := len(logLine) - (currentSize - tc.BytesPerSecond)
				logLine = logLine[:remainder]
				c.instance.Handle(lbls, now, logLine)
				break
			}
			c.instance.Handle(lbls, now, logLine)
		}
	}

	return nil
}

func (c *Client) Stop() {
	c.instance.Stop()
}
