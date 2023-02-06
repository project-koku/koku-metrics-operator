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

	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
)

var (
	promSpec *kokumetricscfgv1beta1.PrometheusSpec

	certKey  = "service-ca.crt"
	tokenKey = "token"

	serviceaccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
)

type PrometheusConnectionTest func(promconn PrometheusConnection) error
type PrometheusConnectionSetter func(promcoll *PrometheusCollector) error

type PrometheusCollector struct {
	PromConn       PrometheusConnection
	PromCfg        *PrometheusConfig
	TimeSeries     *promv1.Range
	ContextTimeout *int64

	serviceaccountPath string
}

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

func NewPromCollector(saPath string) *PrometheusCollector {
	if saPath == "" {
		saPath = serviceaccountPath
	}
	return &PrometheusCollector{
		serviceaccountPath: saPath,
	}
}

func getBearerToken(tokenFile string) (config.Secret, error) {
	encodedSecret, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("getBearerToken: failed to get token: %v", err)
	}
	return config.Secret(encodedSecret), nil
}

func setPrometheusConfig(kmCfg *kokumetricscfgv1beta1.PrometheusSpec, c *PrometheusCollector) error {

	promCfg := &PrometheusConfig{
		Address: kmCfg.SvcAddress,
		CAFile:  filepath.Join(c.serviceaccountPath, certKey),
		SkipTLS: *kmCfg.SkipTLSVerification,
	}

	tokenFile := filepath.Join(c.serviceaccountPath, tokenKey)
	token, err := getBearerToken(tokenFile)
	if err != nil {
		return err
	}
	promCfg.BearerToken = token
	c.PromCfg = promCfg

	return nil
}

func SetPrometheusConnection(promcoll *PrometheusCollector) error {
	cfg := promcoll.PromCfg
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
	promcoll.PromConn = promv1.NewAPI(client)
	return nil
}

func statusHelper(kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig, status string, err error) {
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

func TestPrometheusConnection(promConn PrometheusConnection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return wait.PollImmediate(1*time.Second, 15*time.Second, func() (bool, error) {
		_, _, err := promConn.Query(ctx, "up", time.Now())
		if err != nil {
			return false, err
		}
		return true, err
	})
}

// GetPromConn returns the prometheus connection
func (c *PrometheusCollector) GetPromConn(kmCfg *kokumetricscfgv1beta1.KokuMetricsConfig, setter PrometheusConnectionSetter, tester PrometheusConnectionTest) error {
	log := log.WithName("GetPromConn")
	var err error

	updated := true
	if promSpec != nil {
		updated = !reflect.DeepEqual(*promSpec, kmCfg.Spec.PrometheusConfig)
	}
	promSpec = kmCfg.Spec.PrometheusConfig.DeepCopy()

	if updated || c.PromCfg == nil || kmCfg.Status.Prometheus.ConfigError != "" {
		log.Info("getting prometheus configuration")
		err = setPrometheusConfig(&kmCfg.Spec.PrometheusConfig, c)
		statusHelper(kmCfg, "configuration", err)
		if err != nil {
			return fmt.Errorf("cannot get prometheus configuration: %v", err)
		}
	}

	if updated || c.PromConn == nil || kmCfg.Status.Prometheus.ConnectionError != "" {
		log.Info("getting prometheus connection")
		err = setter(c)
		statusHelper(kmCfg, "configuration", err)
		if err != nil {
			return err
		}
	}

	log.Info("testing the ability to query prometheus")
	err = tester(c.PromConn)
	statusHelper(kmCfg, "connection", err)
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
