//go:generate ../../../tools/readme_config_includer/generator
package influxdb

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/common/tls"
	"github.com/influxdata/telegraf/plugins/inputs"
)

//go:embed sample.conf
var sampleConfig string

const (
	maxErrorResponseBodyLength = 1024
)

type InfluxDB struct {
	URLs     []string        `toml:"urls"`
	Username string          `toml:"username"`
	Password string          `toml:"password"`
	Timeout  config.Duration `toml:"timeout"`
	tls.ClientConfig

	client *http.Client
}

type apiError struct {
	StatusCode  int
	Reason      string
	Description string `json:"error"`
}

func (e *apiError) Error() string {
	if e.Description != "" {
		return e.Reason + ": " + e.Description
	}
	return e.Reason
}

func (*InfluxDB) SampleConfig() string {
	return sampleConfig
}

func (i *InfluxDB) Gather(acc telegraf.Accumulator) error {
	if len(i.URLs) == 0 {
		i.URLs = []string{"http://localhost:8086/debug/vars"}
	}

	if i.client == nil {
		tlsCfg, err := i.ClientConfig.TLSConfig()
		if err != nil {
			return err
		}
		i.client = &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: time.Duration(i.Timeout),
				TLSClientConfig:       tlsCfg,
			},
			Timeout: time.Duration(i.Timeout),
		}
	}

	var wg sync.WaitGroup
	for _, u := range i.URLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			if err := i.gatherURL(acc, url); err != nil {
				acc.AddError(err)
			}
		}(u)
	}

	wg.Wait()

	return nil
}

// Gathers data from a particular URL
// Parameters:
//
//	acc    : The telegraf Accumulator to use
//	url    : endpoint to send request to
//
// Returns:
//
//	error: Any error that may have occurred
func (i *InfluxDB) gatherURL(acc telegraf.Accumulator, url string) error {
	shardCounter := 0
	now := time.Now()

	// Get the data
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if i.Username != "" || i.Password != "" {
		req.SetBasicAuth(i.Username, i.Password)
	}

	req.Header.Set("User-Agent", "Telegraf/"+internal.Version)

	resp, err := i.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return readResponseError(resp)
	}

	// It would be nice to be able to decode into a map[string]point, but
	// we'll get a decoder error like:
	// `json: cannot unmarshal array into Go value of type influxdb.point`
	// if any of the values aren't objects.
	// To avoid that error, we decode by hand.
	dec := json.NewDecoder(resp.Body)

	// Parse beginning of object
	if t, err := dec.Token(); err != nil {
		return err
	} else if t != json.Delim('{') {
		return errors.New("document root must be a JSON object")
	}

	// Loop through rest of object
	for dec.More() {
		// Read in a string key. We don't do anything with the top-level keys,
		// so it's discarded.
		rawKey, err := dec.Token()
		if err != nil {
			return err
		}

		// All variables should be keyed
		key, ok := rawKey.(string)
		if !ok {
			continue
		}

		// Try to decode known special structs
		switch key {
		case "system":
			var p system
			if err := dec.Decode(&p); err != nil {
				continue
			}

			acc.AddFields("influxdb_system",
				map[string]interface{}{
					"current_time": p.CurrentTime,
					"started":      p.Started,
					"uptime":       p.Uptime,
				},
				map[string]string{"url": url},
				now,
			)
			continue
		case "memstats":
			var m memstats
			if err := dec.Decode(&m); err != nil {
				continue
			}
			acc.AddFields("influxdb_memstats",
				map[string]interface{}{
					"alloc":           m.Alloc,
					"total_alloc":     m.TotalAlloc,
					"sys":             m.Sys,
					"lookups":         m.Lookups,
					"mallocs":         m.Mallocs,
					"frees":           m.Frees,
					"heap_alloc":      m.HeapAlloc,
					"heap_sys":        m.HeapSys,
					"heap_idle":       m.HeapIdle,
					"heap_inuse":      m.HeapInuse,
					"heap_released":   m.HeapReleased,
					"heap_objects":    m.HeapObjects,
					"stack_inuse":     m.StackInuse,
					"stack_sys":       m.StackSys,
					"mspan_inuse":     m.MSpanInuse,
					"mspan_sys":       m.MSpanSys,
					"mcache_inuse":    m.MCacheInuse,
					"mcache_sys":      m.MCacheSys,
					"buck_hash_sys":   m.BuckHashSys,
					"gc_sys":          m.GCSys,
					"other_sys":       m.OtherSys,
					"next_gc":         m.NextGC,
					"last_gc":         m.LastGC,
					"pause_total_ns":  m.PauseTotalNs,
					"pause_ns":        m.PauseNs[(m.NumGC+255)%256],
					"num_gc":          m.NumGC,
					"gc_cpu_fraction": m.GCCPUFraction,
				},
				map[string]string{"url": url},
				now,
			)
		case "build":
			var d build
			if err := dec.Decode(&d); err != nil {
				continue
			}
			acc.AddFields("influxdb_build",
				map[string]interface{}{
					"branch":     d.Branch,
					"build_time": d.BuildTime,
					"commit":     d.Commit,
					"version":    d.Version,
				},
				map[string]string{"url": url},
				now,
			)
		case "cmdline":
			var d []string
			if err := dec.Decode(&d); err != nil {
				continue
			}
			acc.AddFields("influxdb_cmdline",
				map[string]interface{}{"value": strings.Join(d, " ")},
				map[string]string{"url": url},
				now,
			)
		case "crypto":
			var d crypto
			if err := dec.Decode(&d); err != nil {
				continue
			}
			acc.AddFields("influxdb_crypto",
				map[string]interface{}{
					"fips":           d.FIPS,
					"ensure_fips":    d.EnsureFIPS,
					"implementation": d.Implementation,
					"password_hash":  d.PasswordHash,
				},
				map[string]string{"url": url},
				now,
			)
		default:
			// Attempt to parse all other entires as an object into a point.
			// Ignore all non-object entries (like a string or array) or
			// entries not conforming to a "point" structure and move on.
			var p point
			if err := dec.Decode(&p); err != nil {
				continue
			}

			// If the object was a point, but was not fully initialized,
			// ignore it and move on.
			if p.Name == "" || p.Values == nil || len(p.Values) == 0 {
				continue
			}

			if p.Name == "shard" {
				shardCounter++
			}

			// Add a tag to indicate the source of the data.
			if p.Tags == nil {
				p.Tags = make(map[string]string)
			}
			p.Tags["url"] = url

			acc.AddFields("influxdb_"+p.Name, p.Values, p.Tags, now)
		}
	}

	// Add a metric for the number of shards
	acc.AddFields("influxdb", map[string]interface{}{"n_shards": shardCounter}, nil, now)

	return nil
}

func readResponseError(resp *http.Response) error {
	apiErr := &apiError{
		StatusCode: resp.StatusCode,
		Reason:     resp.Status,
	}

	var buf bytes.Buffer
	r := io.LimitReader(resp.Body, maxErrorResponseBodyLength)
	_, err := buf.ReadFrom(r)
	if err != nil {
		return apiErr
	}

	err = json.Unmarshal(buf.Bytes(), apiErr)
	if err != nil {
		return apiErr
	}

	return apiErr
}

func init() {
	inputs.Add("influxdb", func() telegraf.Input {
		return &InfluxDB{
			Timeout: config.Duration(time.Second * 5),
		}
	})
}
