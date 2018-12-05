package cattle

import (
	"fmt"
	//"os"
	//"path/filepath"
	"reflect"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
)

// Cattle is a struct which stores the Cattle configuration parameters
type Cattle struct {
	Name            string        `yaml:"name"`
	Endpoint        string        `yaml:"endpoint"`
	AccessKey       string        `yaml:"access_key"`
	SecretKey       string        `yaml:"secret_key"`
	Timeout         time.Duration `yaml:"timeout,omitempty"`
	RefreshInterval time.Duration `yaml:"refresh_interval,omitempty"`
	client          *client.RancherClient
}

type jobConfig struct {
	JobName       string            `yaml:"job_name,omitempty"`
	HonorLabels   bool              `yaml:"honor_labels,omitempty"`
	MetricsPath   string            `yaml:"metrics_path,omitempty"`
	Params        string            `yaml:"params,omitempty"`
	StaticConfigs []staticConfig    `yaml:"static_configs,omitempty"`
	Scheme        string            `yaml:"scheme,omitempty"`
	BasicAuth     map[string]string `yaml:"basic_auth,omitempty"`
}

type staticConfig struct {
	Targets []string          `yaml:"targets,omitempty"`
	Labels  map[string]string `yaml:"labels,omitempty"`
}

type prometheusServer struct {
	name     string
	host     string
	port     string
	username string
	password string
	scheme   string
}

// New creates a new Cattle client
func (cfg *Cattle) New() (err error) {
	err = cfg.setupConfig()
	if err != nil {
		return
	}

	cfg.client, err = client.NewRancherClient(&client.ClientOpts{
		Url:       cfg.Endpoint,
		AccessKey: cfg.AccessKey,
		SecretKey: cfg.SecretKey,
		Timeout:   cfg.Timeout * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create a new Rancher client: %s", err)
	}
	return
}

// Start starts the Cattle service discovery
func (cfg *Cattle) Start(cattleData chan interface{}) {
	var data interface{}
	for {
		log.WithFields(log.Fields{
			"backend": "cattle",
		}).Debugf("Sleeping for %ds", cfg.RefreshInterval)
		time.Sleep(cfg.RefreshInterval * time.Second)

		targets, err := cfg.getTargets()
		if err != nil {
			log.Errorf("failed to retrieve Prometheus servers: %s", err)
			continue
		}

		output, _ := cfg.formatTargets(targets)

		if !reflect.DeepEqual(output, data) {
			data = output
			cattleData <- data
		}
	}
}

func (cfg *Cattle) setupConfig() error {
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 5
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 30
	}

	if cfg.Endpoint == "" {
		return fmt.Errorf("field `endpoint` is required")
	}

	if cfg.Name == "" {
		return fmt.Errorf("field `name` is required")
	}

	if cfg.AccessKey == "" {
		return fmt.Errorf("field `access_key` is required")
	}

	if cfg.SecretKey == "" {
		return fmt.Errorf("field `secret_key` is required")
	}
	return nil
}

func (cfg *Cattle) getTargets() (targets []prometheusServer, err error) {
	stacks, err := cfg.client.Stack.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"limit": -2,
			"all":   true,
		},
	})
	if err != nil {
		log.Errorf("failed to list stacks: %s", err)
		return
	}

	targets = make([]prometheusServer, 0)
	for _, stack := range stacks.Data {
		if stack.Environment["PROMETHEUS_FQDN"] != nil {
			p := prometheusServer{}

			project, err := cfg.client.Project.ById(stack.AccountId)
			if err != nil {
				log.Errorf("failed to retrieve project `%s`: %s", stack.AccountId, err)
			}

			p.name = fmt.Sprintf("%s_%s_%s", cfg.Name, project.Name, project.Id)

			p.host = stack.Environment["PROMETHEUS_FQDN"].(string)

			if stack.Environment["PROMETHEUS_PORT"] != nil {
				p.port = stack.Environment["PROMETHEUS_PORT"].(string)
			} else {
				p.port = "9443"
			}

			if stack.Environment["PROMETHEUS_USERNAME"] != nil {
				p.username = stack.Environment["PROMETHEUS_USERNAME"].(string)
			}

			if stack.Environment["PROMETHEUS_PASSWORD"] != nil {
				p.password = stack.Environment["PROMETHEUS_PASSWORD"].(string)
			}

			if stack.Environment["PROMETHEUS_SCHEME"] != nil {
				p.scheme = stack.Environment["PROMETHEUS_SCHEME"].(string)
			} else {
				p.scheme = "https"
			}

			targets = append(targets, p)
		}
	}

	return
}

func (cfg *Cattle) formatTargets(targets []prometheusServer) (interface{}, error) {
	jobs := []jobConfig{}

	for _, target := range targets {
		job := jobConfig{
			JobName:     target.name,
			HonorLabels: true,
			MetricsPath: "/federate",
			Scheme:      target.scheme,
			StaticConfigs: []staticConfig{
				staticConfig{
					Targets: []string{
						fmt.Sprintf("%s:%s", target.host, target.port),
					},
					Labels: map[string]string{
						"rancher_url":  "foo",
						"rancher_site": "bar",
					},
				},
			},
		}

		if target.username != "" && target.password != "" {
			job.BasicAuth = map[string]string{
				"username": target.username,
				"password": target.password,
			}
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}
