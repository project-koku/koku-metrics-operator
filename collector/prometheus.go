//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package collector

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"k8s.io/apimachinery/pkg/util/wait"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
)

var (
	ps *metricscfgv1beta1.PrometheusSpec

	pollingCtxTimeout = 15 * time.Second

	certKey  = "service-ca.crt"
	tokenKey = "token"

	serviceaccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
)

type PrometheusConnection interface {
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

func statusHelper(cr *metricscfgv1beta1.MetricsConfig, status string, err error) {
	switch status {
	case "configuration":
		if err != nil {
			cr.Status.Prometheus.PrometheusConfigured = false
			cr.Status.Prometheus.ConfigError = fmt.Sprintf("%v", err)
		} else {
			cr.Status.Prometheus.PrometheusConfigured = true
			cr.Status.Prometheus.ConfigError = ""
		}
	case "connection":
		if err != nil {
			cr.Status.Prometheus.PrometheusConnected = false
			cr.Status.Prometheus.ConnectionError = fmt.Sprintf("%v", err)
		} else {
			cr.Status.Prometheus.PrometheusConnected = true
			cr.Status.Prometheus.ConnectionError = ""
		}
	}
}

type PrometheusConfigurationSetter func(ps *metricscfgv1beta1.PrometheusSpec, c *PrometheusCollector) error

func SetPrometheusConfig(ps *metricscfgv1beta1.PrometheusSpec, c *PrometheusCollector) error {

	pCfg := &PrometheusConfig{
		Address: ps.SvcAddress,
		CAFile:  filepath.Join(c.serviceaccountPath, certKey),
		SkipTLS: *ps.SkipTLSVerification,
	}

	tokenFile := filepath.Join(c.serviceaccountPath, tokenKey)
	token, err := getBearerToken(tokenFile)
	if err != nil {
		return err
	}
	pCfg.BearerToken = token
	c.PromCfg = pCfg

	return nil
}

type PrometheusConnectionSetter func(c *PrometheusCollector) error

func SetPrometheusConnection(c *PrometheusCollector) error {
	cfg := c.PromCfg
	promconf := config.HTTPClientConfig{
		BearerToken: cfg.BearerToken,
		TLSConfig:   config.TLSConfig{CAFile: cfg.CAFile, InsecureSkipVerify: cfg.SkipTLS},
	}
	roundTripper, err := config.NewRoundTripperFromConfig(promconf, "promconf")
	if err != nil {
		return fmt.Errorf("cannot create roundTripper: %v", err)
	}
	client, err := promapi.NewClient(promapi.Config{
		Address:      cfg.Address,
		RoundTripper: roundTripper,
	})
	if err != nil {
		return fmt.Errorf("cannot create prometheus client: %v", err)
	}
	c.PromConn = promv1.NewAPI(client)
	return nil
}

type PrometheusConnectionTester func(c *PrometheusCollector) error

func TestPrometheusConnection(c *PrometheusCollector) error {
	ctx, cancel := context.WithTimeout(context.Background(), pollingCtxTimeout)
	defer cancel()
	return wait.PollImmediate(1*time.Second, 15*time.Second, func() (bool, error) {
		_, _, err := c.PromConn.Query(ctx, "up", time.Now())
		if err != nil {
			return false, err
		}
		return true, err
	})
}

type PrometheusCollector struct {
	PromConn       PrometheusConnection
	PromCfg        *PrometheusConfig
	TimeSeries     *promv1.Range
	ContextTimeout *int64

	serviceaccountPath string
}

func NewPromCollector(saPath string) *PrometheusCollector {
	if saPath == "" {
		saPath = serviceaccountPath
	}
	return &PrometheusCollector{
		serviceaccountPath: saPath,
	}
}

// GetPromConn returns the prometheus connection
func (c *PrometheusCollector) GetPromConn(
	cr *metricscfgv1beta1.MetricsConfig,
	pcfgs PrometheusConfigurationSetter,
	pcs PrometheusConnectionSetter,
	pct PrometheusConnectionTester,
) error {
	log := log.WithName("GetPromConn")
	var err error

	updated := true
	if ps != nil {
		updated = !reflect.DeepEqual(*ps, cr.Spec.PrometheusConfig)
	}
	ps = cr.Spec.PrometheusConfig.DeepCopy()

	if updated || c.PromCfg == nil || cr.Status.Prometheus.ConfigError != "" {
		log.Info("getting prometheus configuration")
		err = pcfgs(&cr.Spec.PrometheusConfig, c)
		statusHelper(cr, "configuration", err)
		if err != nil {
			return fmt.Errorf("cannot get prometheus configuration: %v", err)
		}
	}

	if updated || c.PromConn == nil || cr.Status.Prometheus.ConnectionError != "" {
		log.Info("getting prometheus connection")
		err = pcs(c)
		statusHelper(cr, "configuration", err)
		if err != nil {
			return err
		}
	}

	log.Info("testing the ability to query prometheus")
	err = pct(c)
	statusHelper(cr, "connection", err)
	if err != nil {
		return fmt.Errorf("prometheus test query failed: %v", err)
	}
	log.Info("prometheus test query succeeded")

	return nil
}

func (c *PrometheusCollector) getQueryResults(queries *querys, results *mappedResults) error {
	log := log.WithName("getQueryResults")
	timeout := int64(120)
	if c.ContextTimeout != nil {
		timeout = *c.ContextTimeout
	}
	log.Info(fmt.Sprintf("prometheus query timeout set to: %d seconds", timeout))
	for _, query := range *queries {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
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
