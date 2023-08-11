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
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/metrics"
)

var once sync.Once
var instance *lokiClient.Client

type Client struct {
	vu              modules.VU
	metrics         lokiMetrics
	instance        *lokiClient.Client
	flog            *flog.Flog
	addVuAsTenantID bool
}

func GetClient(url string, vu modules.VU, m lokiMetrics, randSeed int64, addVuAsTenantID bool) (*Client, error) {
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

	return &Client{instance: instance, flog: flog, vu: vu, metrics: m, addVuAsTenantID: addVuAsTenantID}, nil
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

	var bytes int // TODO: make this unicode aware
	var lines int

	state.Tags.Modify(func(tagsAndMeta *metrics.TagsAndMeta) {
		tagsAndMeta.SetTag("vuid", string(vuID))
	})

	if tc.LinesPerSecond != 0 {
		for i := 0; i < tc.LinesPerSecond; i++ {
			now := time.Now()
			logLine := c.flog.LogLine(tc.LogType, now)
			logLine = clipLine(tc, logLine)
			bytes += len(logLine)
			c.send(&lbls, tc, vuID, now, logLine)
		}
		lines = tc.LinesPerSecond
	}

	if tc.BytesPerSecond != 0 {
		for {
			now := time.Now()
			logLine := c.flog.LogLine(tc.LogType, now)
			logLine = clipLine(tc, logLine)
			bytes += len(logLine)
			if bytes > tc.BytesPerSecond {
				remainder := len(logLine) - (bytes - tc.BytesPerSecond)
				logLine = logLine[:remainder]
				bytes += len(logLine)
				c.send(&lbls, tc, vuID, now, logLine)
				lines += 1
				break
			}
			c.send(&lbls, tc, vuID, now, logLine)
			lines += 1
		}
	}

	c.reportMetricsFromBatch(bytes, lines)

	return nil
}

func (c *Client) Stop() {
	c.instance.Stop()
}

func (c *Client) reportMetricsFromBatch(bytes, lines int) {
	now := time.Now()
	ctx := c.vu.Context()
	ctm := c.vu.State().Tags.GetCurrentValues()

	metrics.PushIfNotDone(ctx, c.vu.State().Samples, metrics.ConnectedSamples{
		Samples: []metrics.Sample{
			{
				TimeSeries: metrics.TimeSeries{
					Metric: c.metrics.ClientUncompressedBytes,
					Tags:   ctm.Tags,
				},
				Metadata: ctm.Metadata,
				Value:    float64(bytes),
				Time:     now,
			},
			{
				TimeSeries: metrics.TimeSeries{
					Metric: c.metrics.ClientLines,
					Tags:   ctm.Tags,
				},
				Metadata: ctm.Metadata,
				Value:    float64(lines),
				Time:     now,
			},
		},
	})
}
