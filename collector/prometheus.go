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
	"regexp"
	"time"

	"github.com/go-logr/logr"
	promapi "github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
)

var (
	costQuerier PrometheusConfig
	promConn    prom.API

	costMgmtNamespace   = "openshift-cost"
	monitoringNamespace = "openshift-monitoring"
	secretKey           = "token"
	serviceAccountName  = "default"
	thanosRouteName     = "thanos-querier"
	tokenRegex          = "default-token-*"

	certFile = "/var/run/configmaps/trusted-ca-bundle/service-ca.crt"
)

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

func getRuntimeObj(ctx context.Context, r client.Client, obj runtime.Object, key types.NamespacedName, name string) error {
	err := r.Get(ctx, key, obj)
	if err != nil {
		switch {
		case errors.IsNotFound(err):
			return fmt.Errorf("no %s found", name)
		case errors.IsForbidden(err):
			return fmt.Errorf("operator does not have permission to check %s", name)
		default:
			return fmt.Errorf("could not check %s: %v", name, err)
		}
	}
	return nil
}

func getBearerToken(ctx context.Context, r client.Client, cfg *PrometheusConfig) error {
	sa := &corev1.ServiceAccount{}
	objKey := client.ObjectKey{
		Namespace: costMgmtNamespace,
		Name:      serviceAccountName,
	}
	err := getRuntimeObj(ctx, r, sa, objKey, "service account")
	if err != nil {
		return err
	}

	if len(sa.Secrets) <= 0 {
		return fmt.Errorf("getBearerToken: no secrets in service account")
	}

	for _, secret := range sa.Secrets {
		matched, _ := regexp.MatchString(tokenRegex, secret.Name)
		if !matched {
			continue
		}

		s := &corev1.Secret{}
		objKey := client.ObjectKey{
			Namespace: costMgmtNamespace,
			Name:      secret.Name,
		}
		err := getRuntimeObj(ctx, r, s, objKey, "secret")
		if err != nil {
			return err
		}
		encodedSecret, ok := s.Data[secretKey]
		if !ok {
			return fmt.Errorf("getBearerToken: cannot find token in secret")
		}
		if len(encodedSecret) <= 0 {
			return fmt.Errorf("getBearerToken: no data in default secret")
		}
		cfg.BearerToken = config.Secret(encodedSecret)
		return nil
	}
	return fmt.Errorf("getBearerToken: no token found")

}

func getPrometheusConfig(ctx context.Context, r client.Client, cost *costmgmtv1alpha1.CostManagement, log logr.Logger) (*PrometheusConfig, error) {
	cfg := &PrometheusConfig{
		CAFile:  certFile,
		Address: cost.Status.Prometheus.SvcAddress,
		SkipTLS: *cost.Status.Prometheus.SkipTLSVerification,
	}
	if err := getBearerToken(ctx, r, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func statusHelper(cost *costmgmtv1alpha1.CostManagement, status string, err error) {
	switch status {
	case "configuration":
		if err != nil {
			cost.Status.Prometheus.PrometheusConfigured = false
			cost.Status.Prometheus.ConfigError = fmt.Sprintf("%v", err)
		} else {
			cost.Status.Prometheus.PrometheusConfigured = true
			cost.Status.Prometheus.ConfigError = ""
		}
	case "connection":
		if err != nil {
			cost.Status.Prometheus.PrometheusConnected = false
			cost.Status.Prometheus.ConnectionError = fmt.Sprintf("%v", err)
		} else {
			cost.Status.Prometheus.PrometheusConnected = true
			cost.Status.Prometheus.ConnectionError = ""
		}
	}
}

func GetPromConn(ctx context.Context, r client.Client, cost *costmgmtv1alpha1.CostManagement, log logr.Logger) (prom.API, error) {
	log = log.WithValues("costmanagement", "GetPromConn")
	cfg, err := getPrometheusConfig(ctx, r, cost, log)
	statusHelper(cost, "configuration", err)
	if err != nil {
		return nil, fmt.Errorf("cannot get prometheus configuration: %v", err)
	}

	promConn, err = newPrometheusConnFromCfg(*cfg)
	statusHelper(cost, "configuration", err)
	if err != nil {
		return nil, err
	}
	costQuerier = *cfg

	log.Info("testing the ability to query prometheus")
	err = wait.Poll(3*time.Second, 15*time.Second, func() (bool, error) {
		_, _, err := promConn.Query(context.TODO(), "up", time.Now())
		if err != nil {
			return false, err
		}
		return true, err
	})
	statusHelper(cost, "connection", err)
	if err != nil {
		return nil, fmt.Errorf("prometheus test query failed: %v", err)
	}
	log.Info("prometheus test query succeeded")

	return promConn, nil
}

func newPrometheusConnFromCfg(cfg PrometheusConfig) (prom.API, error) {
	if promConn != nil && cfg == costQuerier {
		// reuse the prometheus API
		return promConn, nil
	}
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
	promConn = prom.NewAPI(client)
	return promConn, nil
}

func performMatrixQuery(q collector, query string) (model.Matrix, error) {
	log := q.Log.WithValues("costmanagement", "performMatrixQuery")
	result, warnings, err := q.PrometheusConnection.QueryRange(q.Context, query, q.TimeSeries)
	if err != nil {
		return nil, fmt.Errorf("error querying prometheus: %v", err)
	}
	if len(warnings) > 0 {
		log.Info("query warnings", "Warnings", warnings)
	}
	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("expected a matrix in response to query, got a %v", result.Type())
	}
	return matrix, nil
}
