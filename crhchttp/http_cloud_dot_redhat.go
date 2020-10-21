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
	"net/textproto"
	"os"
	"time"

	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetMultiPartBodyAndHeaders Get multi-part body and headers for upload
func GetMultiPartBodyAndHeaders(logger logr.Logger, filename string) (*bytes.Buffer, *multipart.Writer) {
	log := logger.WithValues("costmanagement", "GetBodyAndHeaders")
	// set the content and content type
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, "file", filename))
	h.Set("Content-Type", "application/vnd.redhat.hccm.tar+tgz")
	fw, err := mw.CreatePart(h)
	f, err := os.Open(filename)
	if err != nil {
		log.Info("error opening file", err)
	}
	defer f.Close()
	_, err = io.Copy(fw, f)
	if err != nil {
		log.Error(err, "The following error occurred")
	}
	mw.Close()
	return buf, mw
}

// SetupRequest Adds headers to request object for communication to cloud.redhat.com
func SetupRequest(logger logr.Logger, costConfig *CostManagementConfig, method string, uri string, body *bytes.Buffer, contentType string) (*http.Request, error) {
	ctx := context.Background()
	log := logger.WithValues("costmanagement", "SetupRequest")
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		log.Error(err, "Could not create request")
		return nil, err
	}
	// Create the header
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", contentType)
	if costConfig.Authentication == "basic" {
		log.Info("Request using basic authentication!")
		req.SetBasicAuth(costConfig.BasicAuthUser, costConfig.BasicAuthPassword)
	} else {
		log.Info("Request using token authentication")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", costConfig.BearerTokenString))
		req.Header.Set("User-Agent", fmt.Sprintf("cost-mgmt-operator/%s cluster/%s", costConfig.OperatorCommit, costConfig.ClusterID))
	}
	// Log the headers - probably remove this later
	log.Info("Request Headers:")
	for key, val := range req.Header {
		fmt.Println(key, val)
	}
	return req, nil
}

// GetClient Return client with certificate handling based on configuration
func GetClient(logger logr.Logger, validateCert bool) http.Client {
	log := logger.WithValues("costmanagement", "GetClient")
	if validateCert {
		// create the client specifying the ca cert file for transport
		caCert, err := ioutil.ReadFile("/var/run/configmaps/trusted-ca-bundle/ca-bundle.crt")
		if err != nil {
			log.Error(err, "The following error occurred: ")
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		client := http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		}
		return client
	} else {
		log.Info("Configured to not using the certificate for this request!")
		// Default the client
		client := http.Client{Timeout: 30 * time.Second}
		return client
	}
}

// ProcessResponse Log response for request and return valid
func ProcessResponse(logger logr.Logger, resp *http.Response) ([]byte, error) {
	log := logger.WithValues("costmanagement", "ProcessResponse")
	// Add error handling and logging here
	requestID := resp.Header.Get("x-rh-insights-request-id")

	log.Info(fmt.Sprintf("gateway server %s - %s returned %d, x-rh-insights-request-id=%s", resp.Request.Method, resp.Request.URL, resp.StatusCode, requestID))
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode >= 300 || resp.StatusCode < 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		if len(body) > 1024 {
			body = body[:1024]
		}
		log.Info(fmt.Sprintf("Error Response Body: %s", string(body)))
		return nil, fmt.Errorf(string(body))
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Info(fmt.Sprintf("Successfully request x-rh-insights-request-id=%s", requestID))
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error(err, "Error occurred reading the response body.")
			return nil, err
		}
		return bodyBytes, nil
	}
	return nil, fmt.Errorf("Unexpected Response")
}

// Upload Send data to cloud.redhat.com
func Upload(logger logr.Logger, costConfig *CostManagementConfig, method string, uri string, body *bytes.Buffer, mw *multipart.Writer) (string, metav1.Time, error) {
	log := logger.WithValues("costmanagement", "Upload")
	req, err := SetupRequest(logger, costConfig, method, uri, body, mw.FormDataContentType())
	currentTime := metav1.Now()
	if err != nil {
		return "", currentTime, err
	}

	client := GetClient(logger, costConfig.ValidateCert)
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "Could not send request")
		return "", currentTime, err
	}
	defer resp.Body.Close()

	fmt.Println("HTTP Response Status:", resp.StatusCode, http.StatusText(resp.StatusCode))
	uploadStatus := fmt.Sprintf("%d ", resp.StatusCode) + string(http.StatusText(resp.StatusCode))
	uploadTime := metav1.Now()

	bodyBytes, err := ProcessResponse(logger, resp)
	if err != nil {
		log.Error(err, "The following error occurred")
	}
	bodyString := string(bodyBytes)
	log.Info("Response body: ")
	log.Info(bodyString)

	return uploadStatus, uploadTime, err
}
