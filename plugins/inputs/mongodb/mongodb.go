//go:generate ../../../tools/readme_config_includer/generator
package mongodb

import (
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/influxdata/telegraf"
	tlsint "github.com/influxdata/telegraf/plugins/common/tls"
	"github.com/influxdata/telegraf/plugins/inputs"
)

// DO NOT REMOVE THE NEXT TWO LINES! This is required to embed the sampleConfig data.
//
//go:embed sample.conf
var sampleConfig string

type MongoDB struct {
	Servers             []string
	GatherClusterStatus bool
	GatherPerdbStats    bool
	GatherColStats      bool
	GatherTopStat       bool
	ColStatsDbs         []string
	tlsint.ClientConfig

	Log telegraf.Logger `toml:"-"`

	clients []*Server
}

func (*MongoDB) SampleConfig() string {
	return sampleConfig
}

func (m *MongoDB) Init() error {
	var tlsConfig *tls.Config
	var err error
	tlsConfig, err = m.ClientConfig.TLSConfig()
	if err != nil {
		return err
	}

	if len(m.Servers) == 0 {
		m.Servers = []string{"mongodb://127.0.0.1:27017"}
	}

	for _, connURL := range m.Servers {
		if !strings.HasPrefix(connURL, "mongodb://") && !strings.HasPrefix(connURL, "mongodb+srv://") {
			// Preserve backwards compatibility for hostnames without a
			// scheme, broken in go 1.8. Remove in Telegraf 2.0
			connURL = "mongodb://" + connURL
			m.Log.Warnf("Using %q as connection URL; please update your configuration to use an URL", connURL)
		}

		u, err := url.Parse(connURL)
		if err != nil {
			return fmt.Errorf("unable to parse connection URL: %q", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel() //nolint:revive

		opts := options.Client().ApplyURI(connURL)
		if tlsConfig != nil {
			opts.TLSConfig = tlsConfig
		}
		if opts.ReadPreference == nil {
			opts.ReadPreference = readpref.Nearest()
		}

		client, err := mongo.Connect(ctx, opts)
		if err != nil {
			return fmt.Errorf("unable to connect to MongoDB: %q", err)
		}

		err = client.Ping(ctx, opts.ReadPreference)
		if err != nil {
			return fmt.Errorf("unable to connect to MongoDB: %s", err)
		}

		server := &Server{
			client:   client,
			hostname: u.Host,
			Log:      m.Log,
		}
		m.clients = append(m.clients, server)
	}

	return nil
}

// Reads stats from all configured servers accumulates stats.
// Returns one of the errors encountered while gather stats (if any).
func (m *MongoDB) Gather(acc telegraf.Accumulator) error {
	var wg sync.WaitGroup
	for _, client := range m.clients {
		wg.Add(1)
		go func(srv *Server) {
			defer wg.Done()
			err := srv.gatherData(acc, m.GatherClusterStatus, m.GatherPerdbStats, m.GatherColStats, m.GatherTopStat, m.ColStatsDbs)
			if err != nil {
				m.Log.Errorf("failed to gather data: %q", err)
			}
		}(client)
	}

	wg.Wait()
	return nil
}

func init() {
	inputs.Add("mongodb", func() telegraf.Input {
		return &MongoDB{
			GatherClusterStatus: true,
			GatherPerdbStats:    false,
			GatherColStats:      false,
			GatherTopStat:       false,
			ColStatsDbs:         []string{"local"},
		}
	})
}
