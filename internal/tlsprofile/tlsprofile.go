//
// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package tlsprofile

import (
	"context"
	"crypto/tls"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logr.Log.WithName("tlsprofile")

// openSSLToGoCipher maps OpenSSL-style cipher names (as used in OpenShift TLS profiles)
// to Go crypto/tls constants. Unsupported ciphers (e.g. DHE-RSA variants not in Go's
// standard library) are omitted. TLS 1.3 ciphers are omitted because Go always enables
// them when TLS 1.3 is negotiated.
var openSSLToGoCipher = map[string]uint16{
	"ECDHE-ECDSA-AES128-GCM-SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"ECDHE-RSA-AES128-GCM-SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"ECDHE-ECDSA-AES256-GCM-SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	"ECDHE-RSA-AES256-GCM-SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"ECDHE-ECDSA-CHACHA20-POLY1305": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	"ECDHE-RSA-CHACHA20-POLY1305":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	"ECDHE-ECDSA-AES128-SHA256":     tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
	"ECDHE-RSA-AES128-SHA256":       tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	"ECDHE-ECDSA-AES128-SHA":        tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	"ECDHE-RSA-AES128-SHA":          tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	"ECDHE-ECDSA-AES256-SHA":        tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	"ECDHE-RSA-AES256-SHA":          tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	"AES128-GCM-SHA256":             tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	"AES256-GCM-SHA384":             tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	"AES128-SHA256":                 tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
	"AES128-SHA":                    tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	"AES256-SHA":                    tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	"DES-CBC3-SHA":                  tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
}

var tlsVersionMap = map[configv1.TLSProtocolVersion]uint16{
	configv1.VersionTLS10: tls.VersionTLS10,
	configv1.VersionTLS11: tls.VersionTLS11,
	configv1.VersionTLS12: tls.VersionTLS12,
	configv1.VersionTLS13: tls.VersionTLS13,
}

// FetchAPIServerTLSConfig reads the cluster APIServer resource, resolves the
// TLS security profile, and returns a *tls.Config ready for use with outbound
// HTTP clients. Returns nil if the profile cannot be fetched (caller should
// fall back to Go defaults).
func FetchAPIServerTLSConfig(ctx context.Context, c client.Client) *tls.Config {
	log := log.WithName("FetchAPIServerTLSConfig")

	apiServer := &configv1.APIServer{}
	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, apiServer); err != nil {
		log.Info("failed to get APIServer config, using Go defaults", "error", err)
		return nil
	}

	profileSpec, err := resolveProfileSpec(apiServer.Spec.TLSSecurityProfile)
	if err != nil {
		log.Info("failed to resolve TLS profile, using Go defaults", "error", err)
		return nil
	}

	log.Info("fetched cluster TLS security profile",
		"minTLSVersion", profileSpec.MinTLSVersion,
		"cipherCount", len(profileSpec.Ciphers))

	cfg := buildTLSConfig(profileSpec)

	log.Info("applied TLS security profile to outbound client config",
		"minVersion", cfg.MinVersion,
		"cipherSuiteCount", len(cfg.CipherSuites))

	return cfg
}

func resolveProfileSpec(profile *configv1.TLSSecurityProfile) (*configv1.TLSProfileSpec, error) {
	if profile == nil || profile.Type == "" {
		return configv1.TLSProfiles[configv1.TLSProfileIntermediateType], nil
	}

	switch profile.Type {
	case configv1.TLSProfileOldType,
		configv1.TLSProfileIntermediateType,
		configv1.TLSProfileModernType:
		spec, ok := configv1.TLSProfiles[profile.Type]
		if !ok {
			return nil, fmt.Errorf("unknown TLS profile type: %s", profile.Type)
		}
		return spec, nil
	case configv1.TLSProfileCustomType:
		if profile.Custom == nil {
			return nil, fmt.Errorf("custom TLS profile specified but Custom field is nil")
		}
		return &profile.Custom.TLSProfileSpec, nil
	default:
		return configv1.TLSProfiles[configv1.TLSProfileIntermediateType], nil
	}
}

func buildTLSConfig(profileSpec *configv1.TLSProfileSpec) *tls.Config {
	minVersion, ok := tlsVersionMap[profileSpec.MinTLSVersion]
	if !ok {
		log.Info("unknown MinTLSVersion, defaulting to TLS 1.2", "version", profileSpec.MinTLSVersion)
		minVersion = tls.VersionTLS12
	}

	cfg := &tls.Config{
		MinVersion: minVersion,
	}

	// TLS 1.3 cipher suites are not configurable in Go; they are always used.
	// Only set CipherSuites when TLS 1.2 connections are possible.
	if minVersion < tls.VersionTLS13 {
		var suites []uint16
		for _, name := range profileSpec.Ciphers {
			if id, ok := openSSLToGoCipher[name]; ok {
				suites = append(suites, id)
			}
		}
		if len(suites) > 0 {
			cfg.CipherSuites = suites
		}
	}

	return cfg
}
