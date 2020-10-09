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
)

var (
	costQuerier PrometheusConfig
	promConn    prom.API

	costMgmtNamespace  = "openshift-cost"
	secretKey          = "token"
	serviceAccountName = "default"
	tokenRegex         = "default-token-*"

	prometheusAddress = "https://thanos-querier-openshift-monitoring.apps.cluster-9071.9071.sandbox1249.opentlc.com"
	certFile          = "/var/run/configmaps/trusted-ca-bundle/ca-bundle.crt"
)

// PrometheusConfig provides the configuration options to set up a Prometheus connections from a URL.
type PrometheusConfig struct {
	// Address is the URL to reach Prometheus.
	Address string
	// BearerToken is the user auth token
	BearerToken config.Secret
	// CAFile is the ca file
	CAFile string
}

func getCoreObj(ctx context.Context, r client.Client, obj runtime.Object, key types.NamespacedName, name string) error {
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
	err := getCoreObj(ctx, r, sa, objKey, "service account")
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
		err := getCoreObj(ctx, r, s, objKey, "secret")
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

func getPrometheusConfig(ctx context.Context, r client.Client, log logr.Logger) (*PrometheusConfig, error) {
	cfg := &PrometheusConfig{
		Address: prometheusAddress,
		CAFile:  certFile,
	}
	if err := getBearerToken(ctx, r, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func GetPromConn(ctx context.Context, r client.Client, log logr.Logger) (prom.API, error) {
	log = log.WithValues("costmanagement", "GetPromConn")
	cfg, err := getPrometheusConfig(ctx, r, log)
	if err != nil {
		return nil, fmt.Errorf("GetPromConn: cannot get prometheus configuration: %v", err)
	}

	promConn, err := newPrometheusConnFromCfg(*cfg)
	if err != nil {
		return nil, fmt.Errorf("GetPromConn: cannot connect to prometheus: %v", err)
	}
	costQuerier = *cfg

	log.Info("testing the ability to query prometheus")

	err = wait.Poll(3*time.Second, 15*time.Second, func() (bool, error) {
		_, _, err := promConn.Query(context.TODO(), "up", time.Now())
		if err != nil {
			return false, err
		}
		log.Info("prometheus test query succeeded")
		return true, err
	})
	if err != nil {
		return nil, fmt.Errorf("prometheus test query failed: %v", err)
	}

	return promConn, nil
}

func newPrometheusConnFromCfg(cfg PrometheusConfig) (prom.API, error) {
	if promConn != nil && cfg == costQuerier {
		// reuse the prometheus API
		return promConn, nil
	}
	promconf := config.HTTPClientConfig{
		BearerToken: cfg.BearerToken,
		TLSConfig:   config.TLSConfig{CAFile: cfg.CAFile, InsecureSkipVerify: true},
	}
	roundTripper, err := config.NewRoundTripperFromConfig(promconf, "promconf", false, false)
	if err != nil {
		return nil, fmt.Errorf("can't create roundTripper: %v", err)
	}
	client, err := promapi.NewClient(promapi.Config{
		Address:      cfg.Address,
		RoundTripper: roundTripper,
	})
	if err != nil {
		return nil, fmt.Errorf("can't connect to prometheus: %v", err)
	}
	promConn = prom.NewAPI(client)
	return promConn, nil
}

func performTheQuery(ctx context.Context, promconn prom.API, query string, ts time.Time, log logr.Logger) (model.Vector, error) {
	log = log.WithValues("costmanagement", "performTheQuery")
	result, warnings, err := promconn.Query(ctx, query, ts)
	if err != nil {
		return nil, fmt.Errorf("error querying prometheus: %v", err)
	}
	if len(warnings) > 0 {
		log.Info("query warnings", "Warnings", warnings)
	}
	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("expected a vector in response to query, got a %v", result.Type())
	}
	return vector, nil
}
