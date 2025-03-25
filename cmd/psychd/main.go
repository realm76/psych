package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/nakabonne/tstorage"
	"github.com/pelletier/go-toml/v2"
)

type Response struct {
	Duration float64
	Status   int
}

type Config struct {
	Sites map[string]SiteConfig
}

type SiteConfig struct {
	Url              string
	Timeout          int
	WarningThreshold float64
	Interval         int
	Key              string
}

type Status int

const (
	StatusDown Status = iota
	StatusUp
	StatusWarn
)

func main() {
	cfg := getConfig()

	client := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	storage, _ := tstorage.NewStorage(
		tstorage.WithTimestampPrecision(tstorage.Seconds),
		tstorage.WithDataPath("./data"),
	)
	defer storage.Close()

	respPool := &sync.Pool{
		New: func() any {
			return &Response{}
		},
	}

	for _, c := range cfg.Sites {
		startSession(c, respPool, storage, client)
	}

}

func createRow(siteConfig SiteConfig, metric string, point tstorage.DataPoint) tstorage.Row {
	return tstorage.Row{
		Metric:    metric,
		DataPoint: point,
		Labels: []tstorage.Label{
			{
				Name:  "url",
				Value: siteConfig.Url,
			},
			{
				Name:  "key",
				Value: siteConfig.Key,
			},
		},
	}
}

func startSession(site SiteConfig, pool *sync.Pool, storage tstorage.Storage, client *http.Client) {
	ticker := time.NewTicker(time.Duration(site.Interval) * time.Millisecond)
	client.Timeout = time.Duration(site.Timeout) * time.Second

	for t := range ticker.C {
		resp := pool.Get().(*Response)
		makeRequest(site, client, t, resp)

		status := StatusDown

		if resp.Status >= 200 && resp.Status < 300 {
			status = StatusUp
		}

		if resp.Duration >= site.WarningThreshold {
			status = StatusWarn
		}

		_ = storage.InsertRows([]tstorage.Row{
			createRow(site, "duration", tstorage.DataPoint{
				Timestamp: t.Unix(),
				Value:     resp.Duration,
			}),

			createRow(site, "statusCode", tstorage.DataPoint{
				Timestamp: t.Unix(),
				Value:     float64(resp.Status),
			}),

			createRow(site, "status", tstorage.DataPoint{
				Timestamp: t.Unix(),
				Value:     float64(status),
			}),
		})

	}
}

func makeRequest(siteConfig SiteConfig, client *http.Client, start time.Time, response *Response) {
	resp, err := client.Get(siteConfig.Url)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	duration := time.Since(start)

	response.Status = resp.StatusCode
	response.Duration = duration.Seconds()

	log.Println(siteConfig.Key, response.Status, response.Duration)
}

func getConfig() Config {
	pwd, _ := os.Getwd()
	filename := filepath.FromSlash(path.Join(pwd, "config.toml"))

	var cfg Config

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		file, err := os.Create(filename)
		if err != nil {
			fmt.Println("Error creating file:", err)
			return cfg
		}
		defer file.Close()

		content := []byte(`
[Sites.Local]
url = "127.0.0.1"
key = "localhost"
timeout = 5
warning_threshold = 2
`)
		if _, err := file.Write(content); err != nil {
			fmt.Println("Error writing to file:", err)
			return cfg
		}
	} else {
		// File exists, read it
		content, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("invalid config file %s\n", filename)
			return cfg
		}

		err = toml.Unmarshal([]byte(content), &cfg)
		if err != nil {
			fmt.Printf("invalid config file %s\n", filename)
			return cfg
		}
	}

	for key, site := range cfg.Sites {
		site.Key = key
	}

	return cfg
}
