package main

import (
	//"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/cryptobioz/prometheus-service-discovery/backends/cattle"
	"github.com/cryptobioz/prometheus-service-discovery/backends/puppetdb"
	"github.com/cryptobioz/prometheus-service-discovery/config"
)

// Backends stores backends configurations
type Backends struct {
	Backends struct {
		PuppetDB []puppetdb.PuppetDB `yaml:"puppetdb,omitempty"`
		Cattle   []cattle.Cattle     `yaml:"cattle,omitempty"`
	} `yaml:"backends,omitempty"`
}

func main() {
	y, err := ioutil.ReadFile("prometheus-service-discovery.yml")
	if err != nil {
		return
	}

	// TODO: Add support for live reload
	cfg, err := config.LoadConfig(y)
	if err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}

	log.SetLevel(log.DebugLevel)

	var b Backends
	err = yaml.Unmarshal(y, &b)
	if err != nil {
		return
	}
	if reflect.DeepEqual(b, (Backends{})) {
		log.Fatalf("no backend provided")
		return
	}

	e := make(map[string][]byte)

	puppetDBData := make(chan interface{})
	cattleData := make(chan interface{})

	if len(b.Backends.PuppetDB) > 0 {
		for _, p := range b.Backends.PuppetDB {
			log.WithFields(log.Fields{
				"backend": "puppetdb",
				"name":    p.Name,
			}).Info("Initializing...")

			err = p.New()
			if err != nil {
				log.Errorf("failed to initialize PuppetBD: %s", err)
				break
			}

			log.WithFields(log.Fields{
				"backend": "puppetdb",
				"name":    p.Name,
			}).Info("Starting...")

			go p.Start(puppetDBData)
		}
	}

	if len(b.Backends.Cattle) > 0 {
		for _, c := range b.Backends.Cattle {
			log.WithFields(log.Fields{
				"backend": "cattle",
				"name":    c.Name,
			}).Info("Initializing...")

			err = c.New()
			if err != nil {
				log.Errorf("failed to initialize Cattle: %s", err)
				continue
			}

			log.WithFields(log.Fields{
				"backend": "cattle",
				"name":    c.Name,
			}).Info("Starting...")

			go c.Start(cattleData)
		}
	}

	var d interface{}
	for {
		select {
		case d = <-puppetDBData:
			e["puppetdb"], err = yaml.Marshal(&d)
			if err != nil {
				log.Errorf("failed to export PuppetDB targets: %s", err)
			}
			log.WithFields(log.Fields{
				"backend": "puppetdb",
			}).Debugf("Exporters list refreshed")
		case d = <-cattleData:
			e["cattle"], err = yaml.Marshal(&d)
			if err != nil {
				log.Errorf("failed to export Cattle targets: %s", err)
			}
			log.WithFields(log.Fields{
				"backend": "cattle",
			}).Debugf("Exporters list refreshed")
		}
		err = writeConfig(cfg, e)
		if err != nil {
			log.Errorf("failed to write config file: %s", err)
		}
	}
}

func writeConfig(cfg config.Config, e map[string][]byte) (err error) {
	var output []string
	for _, v := range e {
		output = append(output, string(v))
	}
	switch cfg.Config.Output.Type {
	case "stdout":
		log.Debugf("%s", strings.Join(output, "\n"))
	case "file":
		os.MkdirAll(filepath.Dir(cfg.Config.Output.Path), 0755)
		err = ioutil.WriteFile(cfg.Config.Output.Path, []byte(strings.Join(output, "\n")), 0644)
		if err != nil {
			log.Errorf("failed to write output: %s", err)
			return
		}
	}
	return
}
