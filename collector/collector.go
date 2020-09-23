/*


Copyright 2020 Red Hat, Inc.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	promapi "github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	defaultPromHost = "http://thanos-querier-openshift-monitoring.svc:9091/"
	address         = defaultPromHost // the URL string for connecting to Prometheus
)

// PrometheusConfig provides the configuration options to set up a Prometheus connections from a URL.
type PrometheusConfig struct {
	// Address is the URL to reach Prometheus.
	Address string
}

func Run(ctx context.Context) error {
	var log logr.Logger

	client, err := promapi.NewClient(promapi.Config{Address: defaultPromHost})
	if err != nil {
		return fmt.Errorf("can't connect to prometheus: %v", err)
	}
	promConn := prom.NewAPI(client)

	log.Info("testing the ability to query from Prometheus")

	err = wait.Poll(3*time.Second, 15*time.Second, func() (bool, error) {
		_, _, err := promConn.Query(context.TODO(), "up", time.Now())
		if err != nil {
			log.Error(err, "failed to succesfully query Prometheus: %v", err)
			return false, nil
		}
		log.Info("queries from Prometheus are succeeding")
		return true, nil
	})
	if err != nil {
		log.Error(err, "queries from Prometheus are failing: %v", err)
	}

	return nil
}
