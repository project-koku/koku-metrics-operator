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

package crhchttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"os"
	"time"

	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Client is an http.Client
var Client HTTPClient

// HTTPClient gives us a testable interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
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
		return nil, "", fmt.Errorf("Failed to create part: %v", err)
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to open file: %v", err)
	}
	defer f.Close()
	_, err = io.Copy(fw, f)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to copy file: %v", err)
	}
	return buf, mw.FormDataContentType(), mw.Close()
}

// SetupRequest creates a new request, adds headers to request object for communication to cloud.redhat.com, and returns the request
func SetupRequest(costConfig *CostManagementConfig, contentType, method, uri string, body *bytes.Buffer) (*http.Request, error) {
	log := costConfig.Log.WithValues("costmanagement", "SetupRequest")

	req, err := http.NewRequestWithContext(context.Background(), method, uri, body)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %v", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	switch costConfig.Authentication {
	case "basic":
		log.Info("Request using basic authentication!")
		req.SetBasicAuth(costConfig.BasicAuthUser, costConfig.BasicAuthPassword)
	default:
		log.Info("Request using token authentication")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", costConfig.BearerTokenString))
		req.Header.Set("User-Agent", fmt.Sprintf("cost-mgmt-operator/%s cluster/%s", costConfig.OperatorCommit, costConfig.ClusterID))
	}

	return req, nil
}

// GetClient Return client with certificate handling based on configuration
func GetClient(costConfig *CostManagementConfig) HTTPClient {
	log := costConfig.Log.WithValues("costmanagement", "GetClient")
	if costConfig.ValidateCert {
		// create the client specifying the ca cert file for transport
		caCert, err := ioutil.ReadFile("/var/run/configmaps/trusted-ca-bundle/ca-bundle.crt")
		if err != nil {
			log.Error(err, "The following error occurred: ") // TODO fix this error handling
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: caCertPool}}
		return &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	}
	log.Info("Configured to not using the certificate for this request!")
	// Default the client
	return &http.Client{Timeout: 30 * time.Second}
}

// ProcessResponse Log response for request and return valid
func ProcessResponse(logger logr.Logger, resp *http.Response) ([]byte, error) {
	log := logger.WithValues("costmanagement", "ProcessResponse")
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

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return nil, fmt.Errorf("error response: %s", body)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	}
	return nil, fmt.Errorf("Unexpected Response")
}

// Upload Send data to cloud.redhat.com
func Upload(costConfig *CostManagementConfig, contentType, method, uri string, body *bytes.Buffer) (string, metav1.Time, error) {
	log := costConfig.Log.WithValues("costmanagement", "Upload")
	currentTime := metav1.Now()
	req, err := SetupRequest(costConfig, contentType, method, uri, body)
	if err != nil {
		return "", currentTime, err
	}

	client := GetClient(costConfig)
	resp, err := client.Do(req)
	if err != nil {
		return "", currentTime, fmt.Errorf("could not send the request: %v", err)
	}
	defer resp.Body.Close()

	uploadStatus := fmt.Sprintf("%d ", resp.StatusCode) + string(http.StatusText(resp.StatusCode))
	uploadTime := metav1.Now()

	_, err = ProcessResponse(log, resp)
	if err != nil {
		return "", currentTime, fmt.Errorf("failed to process the response: %v", err)
	}

	return uploadStatus, uploadTime, nil
}
