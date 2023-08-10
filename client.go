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
	instance        *lokiClient.Client
	flog            *flog.Flog
	addVuAsTenantID bool
}

func GetClient(url string, randSeed int64, addVuAsTenantID bool) (*Client, error) {
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

	return &Client{instance: instance, flog: flog, addVuAsTenantID: addVuAsTenantID}, nil
}

func clipLine(tc *TestConfig, line string) string {
	if tc.MaxLineSize != 0 {
		return line[:tc.MaxLineSize]
	}

	if tc.RandomLineSizeMin != 0 && tc.RandomLineSizeMax != 0 {
		if len(line) <= tc.RandomLineSizeMin {
			return line
		}
		proposedMax := tc.RandomLineSizeMin + rand.Intn(tc.RandomLineSizeMax-tc.RandomLineSizeMin)
		if len(line) < proposedMax {
			proposedMax = len(line)
		}
		return line[:proposedMax]
	}

	return line
}

func addStream(lbls *model.LabelSet, tc *TestConfig) {
	(*lbls)[model.LabelName("stream")] = model.LabelValue(strconv.Itoa(rand.Intn(tc.Streams)))
}

func (c *Client) send(lbls *model.LabelSet, tc *TestConfig, vuID model.LabelValue, now time.Time, logLine string) {
	if tc.Streams != 0 {
		addStream(lbls, tc)
	}
	if c.addVuAsTenantID {
		(*lbls)[model.LabelName(lokiClient.ReservedLabelTenantID)] = vuID
	}
	if tc.TenantID != "" {
		(*lbls)[model.LabelName(lokiClient.ReservedLabelTenantID)] = model.LabelValue(tc.TenantID)
	}
	c.instance.Handle(lbls.Clone(), now, logLine)
}

func (c *Client) GenerateLogs(tc *TestConfig, state *lib.State, logger logrus.FieldLogger) error {
	lbls := tc.StaticLabels.Clone()
	vuID := model.LabelValue(strconv.Itoa(int(state.VUID)))
	lbls[model.LabelName("vuid")] = vuID
	for churnLabelKey, churnLabelValue := range tc.ChurningLabels {
		quotient := state.GetScenarioVUIter() / uint64(churnLabelValue)
		lbls[model.LabelName(churnLabelKey)] = model.LabelValue(strconv.Itoa(int(quotient)))
	}

	if tc.LinesPerSecond != 0 {
		for i := 0; i < tc.LinesPerSecond; i++ {
			now := time.Now()
			logLine := c.flog.LogLine(tc.LogType, now)
			logLine = clipLine(tc, logLine)
			c.send(&lbls, tc, vuID, now, logLine)
		}
	}

	if tc.BytesPerSecond != 0 {
		currentSize := 0
		for {
			now := time.Now()
			logLine := c.flog.LogLine(tc.LogType, now)
			logLine = clipLine(tc, logLine)
			currentSize += len(logLine)
			if currentSize > tc.BytesPerSecond {
				remainder := len(logLine) - (currentSize - tc.BytesPerSecond)
				logLine = logLine[:remainder]
				c.send(&lbls, tc, vuID, now, logLine)
				break
			}
			c.send(&lbls, tc, vuID, now, logLine)
		}
	}

	return nil
}

func (c *Client) Stop() {
	c.instance.Stop()
}
