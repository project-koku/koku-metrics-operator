//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package crhchttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
)

// AuthConfig provides the data for reconciling the CR with defaults
type AuthConfig struct {
	Client              client.Client
	ClusterID           string
	Authentication      metricscfgv1beta1.AuthenticationType
	BearerTokenString   string
	BasicAuthUser       string
	BasicAuthPassword   string
	ValidateCert        bool
	OperatorCommit      string
	ServiceAccountData  ServiceAccountData
	ServiceAccountToken ServiceAccountToken
}

// ServiceAccountData provides the data for acquiring the service acount token
type ServiceAccountData struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
}
type ServiceAccountToken struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not_before_policy"`
	Scope            string `json:"scope"`
}

const serviceaccount = metricscfgv1beta1.ServiceAccount

func (ac *AuthConfig) GetAccessToken(url string) error {
	if ac.Authentication != serviceaccount {
		return nil
	}

	log := log.WithName("GetAccessToken")

	// Marshal ServiceAccountData into JSON prior to requesting
	serviceAccountJSON, err := json.Marshal(ac.ServiceAccountData)
	if err != nil {
		log.Error(err, "failed to marshal service account data")
		return err
	}

	// Make request with marshalled JSON as the POST body
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(serviceAccountJSON))
	if err != nil {
		log.Error(err, "failed to make HTTP request to acquire token")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "failed to read response body")
		return err
	}

	var result ServiceAccountToken
	err = json.Unmarshal([]byte(body), &result)
	if err != nil {
		log.Error(err, "error unmarshaling data from request.")
		return err
	}

	if result.AccessToken == "" {
		errorMsg := "token response did not contain an access token"
		log.Error(errors.New(errorMsg), errorMsg)
		return errors.New(errorMsg)
	}
	// Save the token to AuthConfig BearerTokenString
	log.Info("successfully retrieved and set access token for subsequent requests")
	ac.BearerTokenString = result.AccessToken
	return nil

}
