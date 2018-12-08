package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/cryptobioz/prometheus-service-discovery/backends"
	"github.com/cryptobioz/prometheus-service-discovery/backends/cattle"
	"github.com/cryptobioz/prometheus-service-discovery/backends/puppetdb"
	"github.com/cryptobioz/prometheus-service-discovery/backends/static"
	"github.com/cryptobioz/prometheus-service-discovery/config"
)

// Backends stores backends configurations
type Backends struct {
	Backends map[string][]interface{} `yaml:"backends,omitempty"`
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

	chanData := make(chan backends.BackendData)
	var back backends.BackendInterface

	// TODO: optimize
	for k, v := range b.Backends {
		for _, target := range v {
			switch k {
			case "puppetdb":
				c := &puppetdb.PuppetDB{}
				rawTarget, _ := yaml.Marshal(target)
				err = yaml.Unmarshal(rawTarget, &c)
				back = c
			case "cattle":
				c := &cattle.Cattle{}
				rawTarget, _ := yaml.Marshal(target)
				err = yaml.Unmarshal(rawTarget, &c)
				back = c
			case "static":
				c := &static.Static{}
				rawTarget, _ := yaml.Marshal(target)
				err = yaml.Unmarshal(rawTarget, &c)
				back = c
			}

			log.WithFields(log.Fields{
				"backend": back.GetName(),
				"id":      back.GetID(),
			}).Info("Initializing backend...")
			err = back.New()
			if err != nil {
				log.WithFields(log.Fields{
					"backend": back.GetName(),
					"id":      back.GetID(),
				}).Errorf("failed to initialize backend: %s", err)
			}

			go back.Start(chanData)
		}
	}

	e := make(map[string][]byte)
	var d backends.BackendData
	for {
		select {
		case d = <-chanData:
			i := d
			e[fmt.Sprintf("%s_%s", i.Backend, i.ID)], err = yaml.Marshal(&d.Jobs)
			if err != nil {
				log.WithFields(log.Fields{
					"backend": i.Backend,
					"id":      i.ID,
				}).Errorf("failed to export targets: %s", err)
			}
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
