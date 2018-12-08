package static

import (
	"reflect"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/cryptobioz/prometheus-service-discovery/backends"
)

// Static is a struct which stores the Static configuration parameters
type Static struct {
	backends.JobConfig `yaml:",inline"`
}

// New creates a new Static client
func (cfg *Static) New() (err error) {
	return
}

// Start starts the Static service discovery
func (cfg *Static) Start(d chan backends.BackendData) {
	var data, w backends.BackendData

	for {
		w = backends.BackendData{
			ID:      cfg.JobName,
			Backend: "static",
			Jobs: []backends.JobConfig{
				cfg.JobConfig,
			},
		}

		if !reflect.DeepEqual(w, data) {
			data = w
			d <- data
		}
		log.WithFields(log.Fields{
			"backend": "static",
			"name":    cfg.JobName,
		}).Debugf("Sleeping for %ds", 1000)
		time.Sleep(1000 * time.Second)
	}
}

// GetName returns the backend's name
func (cfg *Static) GetName() string {
	return "static"
}

// GetID returns the target's ID
func (cfg *Static) GetID() string {
	return cfg.JobName
}
