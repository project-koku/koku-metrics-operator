//
// Copyright 2024 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package crhchttp

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"
)

func TestGetClient(t *testing.T) {
	getClientTests := []struct {
		name               string
		validateCert       bool
		tlsCfg             *tls.Config
		insecureSkipVerify bool
		expectedMinVersion uint16
	}{
		{
			name:               "no validate cert without TLS profile",
			validateCert:       false,
			tlsCfg:             nil,
			insecureSkipVerify: true,
		},
		{
			name:               "validate cert without TLS profile",
			validateCert:       true,
			tlsCfg:             nil,
			insecureSkipVerify: false,
		},
		{
			name:               "with TLS profile sets min version",
			validateCert:       true,
			tlsCfg:             &tls.Config{MinVersion: tls.VersionTLS13},
			insecureSkipVerify: false,
			expectedMinVersion: tls.VersionTLS13,
		},
		{
			name:               "TLS profile with insecure skip verify",
			validateCert:       false,
			tlsCfg:             &tls.Config{MinVersion: tls.VersionTLS12},
			insecureSkipVerify: true,
			expectedMinVersion: tls.VersionTLS12,
		},
	}
	for _, tt := range getClientTests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetClient(tt.validateCert, tt.tlsCfg)
			client, ok := result.(*http.Client)
			if !ok {
				t.Errorf("'%s' expected client to be http.Client type, got %T", tt.name, result)
			}
			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Errorf("'%s' expected transport to be http.Transport type, got %T", tt.name, client.Transport)
			}
			if tt.insecureSkipVerify != transport.TLSClientConfig.InsecureSkipVerify {
				t.Errorf("'%s' expected insecureSkipVerify to be %v, got %v", tt.name, tt.insecureSkipVerify, transport.TLSClientConfig.InsecureSkipVerify)
			}
			if tt.expectedMinVersion != 0 && transport.TLSClientConfig.MinVersion != tt.expectedMinVersion {
				t.Errorf("'%s' expected MinVersion to be %v, got %v", tt.name, tt.expectedMinVersion, transport.TLSClientConfig.MinVersion)
			}
		})
	}
}

func TestGetMultiPartBodyAndHeaders(t *testing.T) {
	getMultiPartBodyAndHeadersTests := []struct {
		name                string
		filename            string
		expectedBufNotNil   bool
		expectedErrNotNil   bool
		expectedContentType string
	}{
		{
			name:                "valid file returns correct things",
			filename:            "config.go",
			expectedBufNotNil:   true,
			expectedErrNotNil:   false,
			expectedContentType: "multipart/form-data",
		},
		{
			name:                "invalid file raises error",
			filename:            "file-does-not-exist.go",
			expectedBufNotNil:   false,
			expectedErrNotNil:   true,
			expectedContentType: "",
		},
	}
	for _, tt := range getMultiPartBodyAndHeadersTests {
		t.Run(tt.name, func(t *testing.T) {
			buf, s, err := GetMultiPartBodyAndHeaders(tt.filename)
			if tt.expectedBufNotNil != (buf != nil) {
				t.Errorf("'%s' test expected not-nil buffer, got %v", tt.name, buf)
			}
			if tt.expectedErrNotNil != (err != nil) {
				t.Errorf("'%s' test expected error, got %v", tt.name, err)
			}
			if tt.expectedContentType != "" && !strings.Contains(s, tt.expectedContentType) {
				t.Errorf("'%s' test expected content-type %s, got %v", tt.name, tt.expectedContentType, s)
			} else if tt.expectedContentType == "" && s != tt.expectedContentType {
				t.Errorf("'%s' test expected empty content-type, got %v", tt.name, s)
			}
		})
	}

}
