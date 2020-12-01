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
	"reflect"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
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
	promSpec *costmgmtv1alpha1.PrometheusSpec

	costMgmtNamespace  = "openshift-cost"
	secretKey          = "token"
	serviceAccountName = "default"
	tokenRegex         = "default-token-*"

	certFile = "/var/run/configmaps/trusted-ca-bundle/service-ca.crt"
)

type PromCollector struct {
	Client     client.Client
	PromConn   prometheusConnection
	PromCfg    *PrometheusConfig
	TimeSeries *promv1.Range
	Log        logr.Logger
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

func getRuntimeObj(ctx context.Context, r client.Client, obj runtime.Object, key types.NamespacedName, name string) error {
	if err := r.Get(ctx, key, obj); err != nil {
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

func getBearerToken(clt client.Client) (config.Secret, error) {
	ctx := context.Background()
	sa := &corev1.ServiceAccount{}
	objKey := client.ObjectKey{
		Namespace: costMgmtNamespace,
		Name:      serviceAccountName,
	}
	if err := getRuntimeObj(ctx, clt, sa, objKey, "service account"); err != nil {
		return "", err
	}

	if len(sa.Secrets) <= 0 {
		return "", fmt.Errorf("getBearerToken: no secrets in service account")
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
		if err := getRuntimeObj(ctx, clt, s, objKey, "secret"); err != nil {
			return "", err
		}
		encodedSecret, ok := s.Data[secretKey]
		if !ok {
			return "", fmt.Errorf("getBearerToken: cannot find token in secret")
		}
		if len(encodedSecret) <= 0 {
			return "", fmt.Errorf("getBearerToken: no data in default secret")
		}
		return config.Secret(encodedSecret), nil
	}
	return "", fmt.Errorf("getBearerToken: no token found")

}

func getPrometheusConfig(cost *costmgmtv1alpha1.PrometheusSpec, clt client.Client) (*PrometheusConfig, error) {
	promCfg := &PrometheusConfig{
		CAFile:  certFile,
		Address: cost.SvcAddress,
		SkipTLS: *cost.SkipTLSVerification,
	}
	token, err := getBearerToken(clt)
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
func (c *PromCollector) GetPromConn(cost *costmgmtv1alpha1.CostManagement) error {
	log := c.Log.WithValues("costmanagement", "GetPromConn")
	var err error

	updated := true
	if promSpec != nil {
		updated = !reflect.DeepEqual(*promSpec, cost.Spec.PrometheusConfig)
	}
	promSpec = cost.Spec.PrometheusConfig.DeepCopy()

	if updated || c.PromCfg == nil || cost.Status.Prometheus.ConfigError != "" {
		log.Info("getting prometheus configuration")
		c.PromCfg, err = getPrometheusConfig(&cost.Spec.PrometheusConfig, c.Client)
		statusHelper(cost, "configuration", err)
		if err != nil {
			return fmt.Errorf("cannot get prometheus configuration: %v", err)
		}
	}

	if updated || c.PromConn == nil || cost.Status.Prometheus.ConnectionError != "" {
		log.Info("getting prometheus connection")
		c.PromConn, err = getPrometheusConnFromCfg(c.PromCfg)
		statusHelper(cost, "configuration", err)
		if err != nil {
			return err
		}
	}

	log.Info("testing the ability to query prometheus")
	err = testPrometheusConnection(c.PromConn)
	statusHelper(cost, "connection", err)
	if err != nil {
		return fmt.Errorf("prometheus test query failed: %v", err)
	}
	log.Info("prometheus test query succeeded")

	return nil
}

func (c *PromCollector) getQueryResults(queries *querys, results *mappedResults) error {
	log := c.Log.WithValues("costmanagement", "getQueryResults")
	for _, query := range *queries {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		queryResult, warnings, err := c.PromConn.QueryRange(ctx, query.QueryString, *c.TimeSeries)
		if err != nil {
			return fmt.Errorf("error querying prometheus: %v", err)
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
