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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"k8s.io/apimachinery/pkg/util/wait"

	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
)

var (
	promSpec *kokumetricscfgv1beta1.PrometheusSpec

	certKey  = "service-ca.crt"
	tokenKey = "token"

	serviceaccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
)

type PromCollector struct {
	PromConn   prometheusConnection
	PromCfg    *PrometheusConfig
	TimeSeries *promv1.Range
	Log        logr.Logger
	InCluster  bool
}

type prometheusConnection interface {
	QueryRange(ctx context.Context, query string, r promv1.Range) (model.Value, promv1.Warnings, error)
	Query(ctx context.Context, query string, ts time.Time) (model.Value, promv1.Warnings, error)
}

// PrometheusConfig provides the configuration options to set up a Prometheus connections from a URL.
type PrometheusConfig struct {
	// Address is the URL to reach Prometheus.
	Address string
	// BearerToken is the user auth token
	BearerToken config.Secret
	// CAFile is the ca file
	CAFile string
	// SkipTLS skips cert verification
	SkipTLS bool
}

func getBearerToken(tokenFile string) (config.Secret, error) {
	encodedSecret, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("getBearerToken: failed to get token: %v", err)
	}
	return config.Secret(encodedSecret), nil
}

func getPrometheusConfig(kmCfg *kokumetricscfgv1beta1.PrometheusSpec, inCluster bool) (*PrometheusConfig, error) {
	if !inCluster {
		val, ok := os.LookupEnv("SECRET_ABSPATH")
		if ok {
			serviceaccountPath = val
		}
	}
	promCfg := &PrometheusConfig{
		Address: kmCfg.SvcAddress,
		CAFile:  filepath.Join(serviceaccountPath, certKey),
		SkipTLS: *kmCfg.SkipTLSVerification,
	}

	tokenFile := filepath.Join(serviceaccountPath, tokenKey)
	token, err := getBearerToken(tokenFile)
	if err != nil {
		return nil, err
	}
	promCfg.BearerToken = token

	return promCfg, nil
}

func getPrometheusConnFromCfg(cfg *PrometheusConfig) (promv1.API, error) {
	promconf := config.HTTPClientConfig{
		BearerToken: cfg.BearerToken,
		TLSConfig:   config.TLSConfig{CAFile: cfg.CAFile, InsecureSkipVerify: cfg.SkipTLS},
	}
	roundTripper, err := config.NewRoundTripperFromConfig(promconf, "promconf", false, false)
	if err != nil {
		return nil, fmt.Errorf("cannot create roundTripper: %v", err)
	}
	client, err := promapi.NewClient(promapi.Config{
		Address:      cfg.Address,
		RoundTripper: roundTripper,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create prometheus client: %v", err)
	}
	return promv1.NewAPI(client), nil
}

func statusHelper(kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig, status string, err error) {
	switch status {
	case "configuration":
		if err != nil {
			kmCfg.Status.Prometheus.PrometheusConfigured = false
			kmCfg.Status.Prometheus.ConfigError = fmt.Sprintf("%v", err)
		} else {
			kmCfg.Status.Prometheus.PrometheusConfigured = true
			kmCfg.Status.Prometheus.ConfigError = ""
		}
	case "connection":
		if err != nil {
			kmCfg.Status.Prometheus.PrometheusConnected = false
			kmCfg.Status.Prometheus.ConnectionError = fmt.Sprintf("%v", err)
		} else {
			kmCfg.Status.Prometheus.PrometheusConnected = true
			kmCfg.Status.Prometheus.ConnectionError = ""
		}
	}
}

func testPrometheusConnection(promConn prometheusConnection) error {
	return wait.Poll(1*time.Second, 15*time.Second, func() (bool, error) {
		_, _, err := promConn.Query(context.TODO(), "up", time.Now())
		if err != nil {
			return false, err
		}
		return true, err
	})
}

// GetPromConn returns the prometheus connection
func (c *PromCollector) GetPromConn(kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig) error {
	log := c.Log.WithValues("costmanagementmetricsconfig", "GetPromConn")
	var err error

	updated := true
	if promSpec != nil {
		updated = !reflect.DeepEqual(*promSpec, kmCfg.Spec.PrometheusConfig)
	}
	promSpec = kmCfg.Spec.PrometheusConfig.DeepCopy()

	if updated || c.PromCfg == nil || kmCfg.Status.Prometheus.ConfigError != "" {
		log.Info("getting prometheus configuration")
		c.PromCfg, err = getPrometheusConfig(&kmCfg.Spec.PrometheusConfig, c.InCluster)
		statusHelper(kmCfg, "configuration", err)
		if err != nil {
			return fmt.Errorf("cannot get prometheus configuration: %v", err)
		}
	}

	if updated || c.PromConn == nil || kmCfg.Status.Prometheus.ConnectionError != "" {
		log.Info("getting prometheus connection")
		c.PromConn, err = getPrometheusConnFromCfg(c.PromCfg)
		statusHelper(kmCfg, "configuration", err)
		if err != nil {
			return err
		}
	}

	log.Info("testing the ability to query prometheus")
	err = testPrometheusConnection(c.PromConn)
	statusHelper(kmCfg, "connection", err)
	if err != nil {
		return fmt.Errorf("prometheus test query failed: %v", err)
	}
	log.Info("prometheus test query succeeded")

	return nil
}

func (c *PromCollector) getQueryResults(queries *querys, results *mappedResults) error {
	log := c.Log.WithValues("costmanagementmetricsconfig", "getQueryResults")
	for _, query := range *queries {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		queryResult, warnings, err := c.PromConn.QueryRange(ctx, query.QueryString, *c.TimeSeries)
		if err != nil {
			return fmt.Errorf("query: %s: error querying prometheus: %v", query.QueryString, err)
		}
		if len(warnings) > 0 {
			log.Info("query warnings", "Warnings", warnings)
		}
		matrix, ok := queryResult.(model.Matrix)
		if !ok {
			return fmt.Errorf("expected a matrix in response to query, got a %v", queryResult.Type())
		}

		results.iterateMatrix(matrix, query)
	}
	return nil
}
