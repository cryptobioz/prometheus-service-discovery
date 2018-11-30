package main

import (
	"fmt"
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/cryptobioz/prometheus-service-discovery/backends/puppetdb"
	"github.com/cryptobioz/prometheus-service-discovery/config"
)

type Backends struct {
	Backends struct {
		PuppetDB puppetdb.PuppetDB `yaml:"puppetdb,omitempty"`
	} `yaml:"backends,omitempty"`
}

func main() {
	log.SetLevel(log.DebugLevel)
	y, err := ioutil.ReadFile("prometheus-service-discovery.yml")
	if err != nil {
		return
	}
	// TODO: Add support for live reload
	conf, err := config.LoadConfig(y)
	if err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}
	fmt.Printf("%+v\n", conf)

	var back Backends
	err = yaml.Unmarshal(y, &back)
	if err != nil {
		return
	}

	if back.Backends.PuppetDB != (puppetdb.PuppetDB{}) {
		log.Info("Initializing auto-discovery for PuppetDB...")
		err = back.Backends.PuppetDB.New()
		if err != nil {
			log.Fatalf("failed to initialize PuppetBD: %s", err)
		}
		log.Info("Starting auto-discovery for PuppetDB...")
		go back.Backends.PuppetDB.Start()
	}

	select {}
}
