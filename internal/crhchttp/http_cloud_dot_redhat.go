//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package crhchttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/packaging"
)

// Client is an http.Client
var Client HTTPClient
var cacerts = "/etc/ssl/certs/ca-certificates.crt"
var log = logr.Log.WithName("crc_http")

// DefaultTransport is a copy from the golang http package
var DefaultTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var delimeter = strings.Repeat("=", 100)

// HTTPClient gives us a testable interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func scrubAuthorization(b []byte) string {
	str := strings.Split(string(b), "\r\n")
	for i, s := range str {
		if strings.Contains(s, "Authorization") {
			slice := strings.Split(s, " ")
			idx := len(slice) - 1
			slice[idx] = strings.Repeat("*", len(slice[idx]))
			str[i] = strings.Join(slice, " ")
		}
	}
	return strings.Join(str, "\r\n")
}

// GetMultiPartBodyAndHeaders Get multi-part body and headers for upload
func GetMultiPartBodyAndHeaders(filename string) (*bytes.Buffer, string, error) {
	// set the content and content type
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, "file", filename))
	h.Set("Content-Type", "application/vnd.redhat.hccm.tar+tgz")
	fw, err := mw.CreatePart(h)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create part: %v", err)
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()
	_, err = io.Copy(fw, f)
	if err != nil {
		return nil, "", fmt.Errorf("failed to copy file: %v", err)
	}
	return buf, mw.FormDataContentType(), mw.Close()
}

// SetupRequest creates a new request, adds headers to request object for communication to console.redhat.com, and returns the request
func SetupRequest(authConfig *AuthConfig, contentType, method, uri string, body *bytes.Buffer) (*http.Request, error) {
	log := log.WithName("SetupRequest")

	req, err := http.NewRequestWithContext(context.Background(), method, uri, body)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %v", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	log.Info(fmt.Sprintf("request using %s authentication", authConfig.Authentication))
	switch authConfig.Authentication {
	case metricscfgv1beta1.Basic:
		req.SetBasicAuth(authConfig.BasicAuthUser, authConfig.BasicAuthPassword)
	case metricscfgv1beta1.ServiceAccount:
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authConfig.BearerTokenString))
	default:
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authConfig.BearerTokenString))
		req.Header.Set("User-Agent", fmt.Sprintf("cost-mgmt-operator/%s cluster/%s", authConfig.OperatorCommit, authConfig.ClusterID))
	}

	// log the request headers
	byteReq, err := httputil.DumpRequest(req, false)
	if err == nil { // only log if the dump is successful
		log.Info(fmt.Sprintf("request:\n%s", scrubAuthorization(byteReq)))
	}

	return req, nil
}

// GetClient Return client with certificate handling based on configuration
func GetClient(authConfig *AuthConfig) HTTPClient {
	log := log.WithName("GetClient")
	transport := DefaultTransport
	if authConfig.ValidateCert {
		// create the client specifying the ca cert file for transport
		caCert, err := os.ReadFile(cacerts)
		if err != nil {
			log.Error(err, "The following error occurred: ") // TODO fix this error handling
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		transport.TLSClientConfig = &tls.Config{RootCAs: caCertPool}
	}
	// Default the client
	return &http.Client{Timeout: 30 * time.Second, Transport: transport}
}

// ProcessResponse Log response for request and return valid
func ProcessResponse(resp *http.Response) ([]byte, error) {
	log := log.WithName("ProcessResponse")
	log.Info("request response",
		"method", resp.Request.Method,
		"status", resp.StatusCode,
		"URL", resp.Request.URL,
		"x-rh-insights-request-id", resp.Header.Get("x-rh-insights-request-id"))

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, fmt.Errorf("failed to dump response body: %v", err)
	}
	log.Info(fmt.Sprintf("request response:\n%s", dump))

	bodySlice := bytes.SplitN(dump, []byte("\r\n\r\n"), 2)
	if len(bodySlice) != 2 {
		return nil, fmt.Errorf("failed to read response body: DumpResponse split length does not equal 2")
	}
	body := bodySlice[1]

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return nil, fmt.Errorf("status: %d | error response: %s", resp.StatusCode, body)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	}
	return nil, fmt.Errorf("unexpected response: %d", resp.StatusCode)
}

// Upload Send data to console.redhat.com
func Upload(authConfig *AuthConfig, contentType, method, uri string, body *bytes.Buffer, fileInfo packaging.FileInfoManifest, file string) (string, metav1.Time, string, error) {
	log := log.WithName("Upload")
	currentTime := metav1.Now()
	req, err := SetupRequest(authConfig, contentType, method, uri, body)
	if err != nil {
		return "", currentTime, "", fmt.Errorf("could not setup the request: %v", err)
	}

	client := GetClient(authConfig)
	resp, err := client.Do(req)
	if err != nil {
		return "", currentTime, "", fmt.Errorf("could not send the request: %v", err)
	}
	defer resp.Body.Close()

	uploadStatus := fmt.Sprintf("%d ", resp.StatusCode) + string(http.StatusText(resp.StatusCode))
	uploadTime := metav1.Now()

	resBody, err := ProcessResponse(resp)
	log.Info("\n\n" + delimeter +
		"\nmethod: " + resp.Request.Method +
		"\nstatus: " + fmt.Sprint(resp.StatusCode) +
		"\nURL: " + fmt.Sprint(resp.Request.URL) +
		"\nx-rh-insights-request-id: " + resp.Header.Get("x-rh-insights-request-id") +
		"\nPackaged file name: " + file +
		"\nFiles included: " + fmt.Sprint(fileInfo.Files) +
		"\nManifest ID: " + fileInfo.UUID +
		"\nCluster ID: " + fileInfo.ClusterID +
		"\nAccount ID: " + string(resBody) +
		"\n" + delimeter + "\n\n")

	if err != nil {
		return uploadStatus, currentTime, resp.Header.Get("x-rh-insights-request-id"), err
	}

	return uploadStatus, uploadTime, resp.Header.Get("x-rh-insights-request-id"), nil
}
