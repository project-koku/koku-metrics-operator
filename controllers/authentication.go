package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/crhchttp"
	"github.com/project-koku/koku-metrics-operator/sources"
)

type previousAuthValidation struct {
	secretName string
	username   string
	password   string
	err        error
	timestamp  metav1.Time
}

type serializedAuthMap struct {
	Auths map[string]serializedAuth `json:"auths"`
}
type serializedAuth struct {
	Auth string `json:"auth"`
}

// getPullSecretToken Obtain the bearer token string from the pull secret in the openshift-config namespace
func getPullSecretToken(r *MetricsConfigReconciler, authConfig *crhchttp.AuthConfig) error {
	ctx := context.Background()
	log := log.WithName("GetPullSecretToken")

	secret, err := r.Clientset.CoreV1().Secrets(openShiftConfigNamespace).Get(ctx, pullSecretName, metav1.GetOptions{})
	if err != nil {
		switch {
		case errors.IsNotFound(err):
			log.Error(err, "pull-secret does not exist")
		case errors.IsForbidden(err):
			log.Error(err, "operator does not have permission to check pull-secret")
		default:
			log.Error(err, "could not check pull-secret")
		}
		return err
	}

	tokenFound := false
	encodedPullSecret := secret.Data[pullSecretDataKey]
	if len(encodedPullSecret) <= 0 {
		return fmt.Errorf("cluster authorization secret did not have data")
	}
	var pullSecret serializedAuthMap
	if err := json.Unmarshal(encodedPullSecret, &pullSecret); err != nil {
		log.Error(err, "unable to unmarshal cluster pull-secret")
		return err
	}
	if auth, ok := pullSecret.Auths[pullSecretAuthKey]; ok {
		token := strings.TrimSpace(auth.Auth)
		if strings.Contains(token, "\n") || strings.Contains(token, "\r") {
			return fmt.Errorf("cluster authorization token is not valid: contains newlines")
		}
		if len(token) > 0 {
			log.Info("found cloud.openshift.com token")
			authConfig.BearerTokenString = token
			tokenFound = true
		} else {
			return fmt.Errorf("cluster authorization token is not found")
		}
	} else {
		return fmt.Errorf("cluster authorization token was not found in secret data")
	}
	if !tokenFound {
		return fmt.Errorf("cluster authorization token is not found")
	}
	return nil
}

// getAuthSecret Obtain the username and password from the authentication secret provided in the current namespace
func getAuthSecret(r *MetricsConfigReconciler, cr *metricscfgv1beta1.MetricsConfig, authConfig *crhchttp.AuthConfig, reqNamespace types.NamespacedName) error {
	ctx := context.Background()
	log := log.WithName("GetAuthSecret")

	if previousValidation == nil || previousValidation.secretName != cr.Status.Authentication.AuthenticationSecretName {
		previousValidation = &previousAuthValidation{secretName: cr.Status.Authentication.AuthenticationSecretName}
	}

	log.Info("secret namespace", "namespace", reqNamespace.Namespace)
	secret := &corev1.Secret{}
	namespace := types.NamespacedName{
		Namespace: reqNamespace.Namespace,
		Name:      cr.Status.Authentication.AuthenticationSecretName}
	err := r.Get(ctx, namespace, secret)
	if err != nil {
		switch {
		case errors.IsNotFound(err):
			log.Error(err, "secret does not exist")
		case errors.IsForbidden(err):
			log.Error(err, "operator does not have permission to check secret")
		default:
			log.Error(err, "could not check secret")
		}
		return err
	}

	keys := make(map[string]string)
	for k, v := range secret.Data {
		keys[strings.ToLower(k)] = string(v)
	}

	for _, k := range []string{authSecretUserKey, authSecretPasswordKey} {
		if len(keys[k]) <= 0 {
			msg := fmt.Sprintf("secret not found with expected %s data", k)
			log.Info(msg)
			return fmt.Errorf(msg)
		}
	}

	authConfig.BasicAuthUser = keys[authSecretUserKey]
	authConfig.BasicAuthPassword = keys[authSecretPasswordKey]

	return nil
}

func setAuthentication(r *MetricsConfigReconciler, authConfig *crhchttp.AuthConfig, cr *metricscfgv1beta1.MetricsConfig, reqNamespace types.NamespacedName) error {
	log := log.WithName("setAuthentication")
	cr.Status.Authentication.AuthenticationCredentialsFound = &trueDef
	if cr.Status.Authentication.AuthType == metricscfgv1beta1.Token {
		cr.Status.Authentication.ValidBasicAuth = nil
		cr.Status.Authentication.AuthErrorMessage = ""
		cr.Status.Authentication.LastVerificationTime = nil
		// Get token from pull secret
		err := getPullSecretToken(r, authConfig)
		if err != nil {
			log.Error(nil, "failed to obtain cluster authentication token")
			cr.Status.Authentication.AuthenticationCredentialsFound = &falseDef
			cr.Status.Authentication.AuthErrorMessage = err.Error()
		}
		return err
	} else if cr.Spec.Authentication.AuthenticationSecretName != "" {
		// Get user and password from auth secret in namespace
		err := getAuthSecret(r, cr, authConfig, reqNamespace)
		if err != nil {
			log.Error(nil, "failed to obtain authentication secret credentials")
			cr.Status.Authentication.AuthenticationCredentialsFound = &falseDef
			cr.Status.Authentication.AuthErrorMessage = err.Error()
			cr.Status.Authentication.ValidBasicAuth = &falseDef
		}
		return err
	} else {
		// No authentication secret name set when using basic auth
		cr.Status.Authentication.AuthenticationCredentialsFound = &falseDef
		err := fmt.Errorf("no authentication secret name set when using basic auth")
		cr.Status.Authentication.AuthErrorMessage = err.Error()
		cr.Status.Authentication.ValidBasicAuth = &falseDef
		return err
	}
}

func validateCredentials(r *MetricsConfigReconciler, handler *sources.SourceHandler, cr *metricscfgv1beta1.MetricsConfig, cycle int64) error {
	log := log.WithName("validateCredentials")

	if cr.Spec.Authentication.AuthType == metricscfgv1beta1.Token {
		// no need to validate token auth
		return nil
	}

	if previousValidation == nil {
		previousValidation = &previousAuthValidation{}
	}

	if previousValidation.password == handler.Auth.BasicAuthPassword &&
		previousValidation.username == handler.Auth.BasicAuthUser &&
		!checkCycle(log, cycle, previousValidation.timestamp, "credential verification") {
		return previousValidation.err
	}

	log.Info("validating credentials")
	client := crhchttp.GetClient(handler.Auth)
	_, err := sources.GetSources(handler, client)

	previousValidation.username = handler.Auth.BasicAuthUser
	previousValidation.password = handler.Auth.BasicAuthPassword
	previousValidation.err = err
	previousValidation.timestamp = metav1.Now()

	cr.Status.Authentication.LastVerificationTime = &previousValidation.timestamp

	if err != nil && strings.Contains(err.Error(), "401") {
		msg := fmt.Sprintf("cloud.redhat.com credentials are invalid. Correct the username/password in `%s`. Updated credentials will be re-verified during the next reconciliation.", cr.Spec.Authentication.AuthenticationSecretName)
		log.Info(msg)
		cr.Status.Authentication.AuthErrorMessage = msg
		cr.Status.Authentication.ValidBasicAuth = &falseDef
		return err
	}
	log.Info("credentials are valid")
	cr.Status.Authentication.AuthErrorMessage = ""
	cr.Status.Authentication.ValidBasicAuth = &trueDef
	return nil
}
