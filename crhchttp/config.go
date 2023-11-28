//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package crhchttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

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

// ServiceAccountToken holds the structure of the token response.
// The fields within the struct provide information about the access token that is used to authenticate service account requests.
type ServiceAccountToken struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	Scope            string `json:"scope"`
}

// AuthError represents a client error returned when authenticating client credentials.
type AuthError struct {
	StatusCode  int
	ErrorType   string `json:"error"`
	Description string `json:"error_description"`
}

func (a *AuthError) Error() string {
	return fmt.Sprintf("status: %d, error: %s, description: %s", a.StatusCode, a.ErrorType, a.Description)
}

const serviceaccount = metricscfgv1beta1.ServiceAccount

func (ac *AuthConfig) GetAccessToken(cxt context.Context, tokenURL string) error {
	if ac.Authentication != serviceaccount {
		return nil
	}

	log := log.WithName("GetAccessToken")

	// Prepare the POST data
	data := url.Values{}
	data.Set("client_id", ac.ServiceAccountData.ClientID)
	data.Set("client_secret", ac.ServiceAccountData.ClientSecret)
	data.Set("grant_type", ac.ServiceAccountData.GrantType)

	// // Making the HTTP POST request
	cxt, cancel := context.WithTimeout(cxt, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cxt, http.MethodPost, tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to construct HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	log.Info("requesting service-account access token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request to acquire token: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errResponse := &AuthError{StatusCode: resp.StatusCode}
		if err := json.NewDecoder(resp.Body).Decode(errResponse); err != nil {
			return fmt.Errorf("status: %d, failed to unmarshal error response: %w", resp.StatusCode, err)
		}
		return errResponse
	}

	var result ServiceAccountToken
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Save the token to AuthConfig BearerTokenString
	ac.BearerTokenString = result.AccessToken
	log.Info("successfully retrieved and set access token for subsequent requests")
	return nil
}
