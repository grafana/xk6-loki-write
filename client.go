package loki

import (
	"math/rand"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	lokiClient "github.com/grafana/loki-client-go/loki"
	"github.com/grafana/xk6-loki/flog"
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

func (c *Client) GenerateLogs(tc *TestConfig) error {
	for i := 0; i < tc.LineSize; i++ {
		now := time.Now()
		c.instance.Handle(tc.StaticLabels, now, c.flog.LogLine("logfmt", now))
	}
	return nil
}

func (c *Client) Stop() {
	c.instance.Stop()
}
