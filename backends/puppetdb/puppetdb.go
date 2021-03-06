package puppetdb

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/cryptobioz/prometheus-service-discovery/backends"
)

// PuppetDB is a struct which stores the PuppetDB configuration parameters
type PuppetDB struct {
	Name            string        `yaml:"name"`
	URL             string        `yaml:"url"`
	CertFile        string        `yaml:"certfile,omitempty"`
	KeyFile         string        `yaml:"keyfile,omitempty"`
	CACertFile      string        `yaml:"cacert,omitempty"`
	SSLSkipVerify   bool          `yaml:"ssl_skip_verify,omitempty"`
	Query           string        `yaml:"query"`
	Output          string        `yaml:"output"`
	OutputFile      string        `yaml:"output_file"`
	Timeout         int           `yaml:"timeout,omitempty"`
	RefreshInterval time.Duration `yaml:"refresh_interval,omitempty"`
	client          *http.Client
}

type node struct {
	Certname  string                `json:"certname"`
	Exporters map[string][]exporter `json:"value"`
}

type exporter struct {
	URL    string            `json:"url"`
	Labels map[string]string `json:"labels,omitempty"`
}

// GetName returns the backend's name
func (cfg *PuppetDB) GetName() string {
	return "puppetdb"
}

// GetID returns the target's ID
func (cfg *PuppetDB) GetID() string {
	return cfg.Name
}

// New creates a new PuppetDB client
func (cfg *PuppetDB) New() (err error) {
	puppetDBUrl, err := url.Parse(cfg.URL)
	if err != nil {
		return
	}

	if puppetDBUrl.Scheme != "http" && puppetDBUrl.Scheme != "https" {
		return fmt.Errorf("%s is not a valid http scheme", puppetDBUrl.Scheme)
	}

	var transport *http.Transport
	if puppetDBUrl.Scheme == "https" {
		// Load client cert
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return err
		}

		// Load CA cert
		caCert, err := ioutil.ReadFile(cfg.CACertFile)
		if err != nil {
			return err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// Setup HTTPS client
		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: cfg.SSLSkipVerify,
		}
		tlsConfig.BuildNameToCertificate()
		transport = &http.Transport{TLSClientConfig: tlsConfig}
	} else {
		transport = &http.Transport{}
	}

	cfg.client = &http.Client{Transport: transport}

	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 5
	}

	if cfg.Query == "" {
		return fmt.Errorf("field `query` is required")
	}

	if cfg.URL == "" {
		return fmt.Errorf("field `url` is required")
	}

	if cfg.OutputFile != "" {
		cfg.Output = "file"
	}

	if cfg.Output == "" {
		cfg.Output = "stdout"
	}
	return
}

// Start starts the PuppetDB service discovery
func (cfg *PuppetDB) Start(puppetDBData chan backends.BackendData) {
	var data backends.BackendData
	for {
		log.WithFields(log.Fields{
			"backend": "puppetdb",
		}).Debugf("Sleeping for %ds", cfg.RefreshInterval)
		time.Sleep(cfg.RefreshInterval * time.Second)

		jobs, err := cfg.getTargets()
		if err != nil {
			log.Errorf("failed to get exporters: %s", err)
			continue
		}
		output := backends.BackendData{
			ID:      cfg.Name,
			Backend: "puppetdb",
			Jobs:    jobs.([]backends.JobConfig),
		}

		if !reflect.DeepEqual(output, data) {
			data = output
			puppetDBData <- data
		}
	}
}

func (cfg *PuppetDB) getNodes() (nodes []node, err error) {
	form := strings.NewReader(fmt.Sprintf("{\"query\":\"%s\"}", cfg.Query))
	puppetDBUrl := fmt.Sprintf("%s/pdb/query/v4", cfg.URL)
	req, err := http.NewRequest("POST", puppetDBUrl, form)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := cfg.client.Do(req)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &nodes)
	return
}

func (cfg *PuppetDB) getTargets() (interface{}, error) {
	fileSdConfig := []backends.StaticConfig{}

	nodes, err := cfg.getNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %s", err)
	}

	for _, n := range nodes {
		for jobName, targets := range n.Exporters {
			for _, exp := range targets {
				u, err := url.Parse(exp.URL)
				if err != nil {
					return nil, err
				}

				labels := map[string]string{
					"certname":     n.Certname,
					"metrics_path": u.Path,
					"job":          jobName,
					"scheme":       u.Scheme,
				}

				for k, v := range u.Query() {
					labels[fmt.Sprintf("__param_%s", k)] = v[0]
					labels[k] = v[0]
				}

				for k, v := range exp.Labels {
					labels[k] = v
				}

				staticConfig := backends.StaticConfig{
					Targets: []string{u.Host},
					Labels:  labels,
				}
				fileSdConfig = append(fileSdConfig, staticConfig)
			}
		}
	}

	job := []backends.JobConfig{
		backends.JobConfig{
			JobName:       cfg.Name,
			HonorLabels:   true,
			MetricsPath:   "/metrics",
			StaticConfigs: fileSdConfig,
			Scheme:        "http",
		},
	}
	return job, nil
}
