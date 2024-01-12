//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package sources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/crhchttp"
)

const (
	// SourceTypesEndpoint The endpoint for retrieving source types.
	SourceTypesEndpoint string = "source_types"

	// OpenShiftSourceType The value to query to find the source type ID.
	OpenShiftSourceType string = "openshift"

	// ApplicationTypesEndpoint The endpoint for retrieving application types.
	ApplicationTypesEndpoint string = "application_types"

	// CostManagementAppType The value to query to find the application ID
	CostManagementAppType string = "/insights/platform/cost-management"

	// SourcesEndpoint The endpoint for retrieving and creating sources.
	SourcesEndpoint string = "sources"

	// NameFilterQueryParam The keyword for filtering by name via query parameter.
	NameFilterQueryParam string = "filter[name]"

	// SourceTypeIDFilterQueryParam The keyword for filtering by source_type_id via query parameter.
	SourceTypeIDFilterQueryParam string = "filter[source_type_id]"

	// SourceRefFilterQueryParam The keyword for filtering by source_ref via query parameter.
	SourceRefFilterQueryParam string = "filter[source_ref]"

	// ApplicationsEndpoint The endpoint for associating a source with an application.
	ApplicationsEndpoint string = "applications"
)

var log = logr.Log.WithName("source_handler")

// GenericMeta A data structure for the meta data in a paginated response
type GenericMeta struct {
	Count int
}

// SourceTypeDataItem A data structure for the source type item
type SourceTypeDataItem struct {
	ID   string
	Name string
}

// SourceTypeResponse A data structure for the paginated source type response
type SourceTypeResponse struct {
	Meta GenericMeta
	Data []SourceTypeDataItem
}

// SourceItem A data structure for the source type item
type SourceItem struct {
	ID           string
	Name         string
	SourceTypeID string `json:"source_type_id"`
	SourceRef    string `json:"source_ref"`
}

// SourceResponse A data structure for the paginated source response
type SourceResponse struct {
	Meta GenericMeta
	Data []SourceItem
}

// ApplicationTypeDataItem A data structure for the application type item
type ApplicationTypeDataItem struct {
	ID   string
	Name string
}

// ApplicationTypeResponse A data structure for the paginated application type response
type ApplicationTypeResponse struct {
	Meta GenericMeta
	Data []ApplicationTypeDataItem
}

type sourceGetReq struct {
	client   crhchttp.HTTPClient
	root     string
	endpoint string
	queries  map[string]string
	errKey   string
}

type sourcePostReq struct {
	client   crhchttp.HTTPClient
	root     string
	endpoint string
	values   map[string]string
	errKey   string
}

type SourceHandler struct {
	APIURL string
	Auth   *crhchttp.AuthConfig
	Spec   metricscfgv1beta1.CloudDotRedHatSourceStatus
}

func (s *sourceGetReq) getRequest(handler *SourceHandler) ([]byte, error) {
	log := log.WithName("getRequest")
	uri := s.root + s.endpoint
	req, err := crhchttp.SetupRequest(handler.Auth, "", "GET", uri, &bytes.Buffer{})
	if err != nil {
		return nil, fmt.Errorf("Failed to construct request for %s from Sources API: %v.", s.errKey, err)
	}

	q := req.URL.Query()
	for k, v := range s.queries {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	log.Info("GET Request - " + req.URL.Path)
	return doRequest(handler, req, s.client, s.errKey)
}

func (s *sourcePostReq) jsonRequest(handler *SourceHandler) ([]byte, error) {
	log := log.WithName("jsonRequest")
	uri := s.root + s.endpoint
	jsonValue, err := json.Marshal(s.values)
	if err != nil {
		return nil, fmt.Errorf("Failed to construct body for call to create a Source with the Sources API: %v.", err)
	}
	req, err := crhchttp.SetupRequest(handler.Auth, "application/json", "POST", uri, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("Failed to construct request for %s from Sources API: %v.", s.errKey, err)
	}

	log.Info("POST Request - " + req.URL.Path)
	return doRequest(handler, req, s.client, s.errKey)
}

func doRequest(handler *SourceHandler, r *http.Request, client crhchttp.HTTPClient, errKey string) ([]byte, error) {
	log := log.WithName("doRequest")
	resp, err := client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("Failed request to Sources API (%s) for %s: %v.", r.URL.Path, errKey, err)
	}
	defer resp.Body.Close()

	respMsg := fmt.Sprintf("HTTP Response Status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	log.Info(respMsg)

	byteBody, err := crhchttp.ProcessResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("Failed to process the response for %s: %v.", errKey, err)
	}

	return byteBody, nil
}

// GetSourceTypeID Request the source type ID for OpenShift
func GetSourceTypeID(handler *SourceHandler, client crhchttp.HTTPClient) (string, error) {
	log := log.WithName("GetSourceTypeID")
	request := &sourceGetReq{
		client:   client,
		root:     handler.APIURL + handler.Spec.SourcesAPIPath,
		endpoint: SourceTypesEndpoint,
		queries:  map[string]string{NameFilterQueryParam: OpenShiftSourceType},
		errKey:   "OpenShift source type lookup",
	}

	// Get Source Type ID
	// https://console.redhat.com/api/sources/v1.0/source_types?filter[name]=openshift
	bodyBytes, err := request.getRequest(handler)
	if err != nil {
		return "", err
	}

	var data SourceTypeResponse
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		log.Error(err, "could not parse output of response")
		return "", fmt.Errorf("Failed to parse OpenShift source type response from Sources API: %v.", err)
	}

	if data.Meta.Count != 1 {
		err = fmt.Errorf("the openshift source type was not found, response count was %d", data.Meta.Count)
		log.Error(err, "unexpected response from source type API")
		return "", fmt.Errorf("Failed to obtain the source type ID for OpenShift: %v.", err)
	}

	return data.Data[0].ID, nil
}

// GetSources does a basic get request to the sources endpoint
func GetSources(handler *SourceHandler, client crhchttp.HTTPClient) ([]byte, error) {
	request := &sourceGetReq{
		client:   client,
		root:     handler.APIURL + handler.Spec.SourcesAPIPath,
		endpoint: SourcesEndpoint,
		errKey:   "validating auth credentials",
	}

	// https://console.redhat.com/api/sources/v1.0/sources
	return request.getRequest(handler)
}

// CheckSourceExists Determine if the source exists with given parameters
func CheckSourceExists(handler *SourceHandler, client crhchttp.HTTPClient, sourceTypeID, name, sourceRef string) (*SourceItem, error) {
	log := log.WithName("CheckSourceExists")
	request := &sourceGetReq{
		client:   client,
		root:     handler.APIURL + handler.Spec.SourcesAPIPath,
		endpoint: SourcesEndpoint,
		errKey:   "obtaining the OpenShift source",
	}
	queries := map[string]string{}
	if name != "" {
		queries[NameFilterQueryParam] = name
	}
	if sourceRef != "" {
		queries[SourceRefFilterQueryParam] = sourceRef
	}
	if sourceTypeID != "" {
		queries[SourceTypeIDFilterQueryParam] = sourceTypeID
	}
	request.queries = queries

	// Check if Source exists already
	// https://console.redhat.com/api/sources/v1.0/sources?filter[source_type_id]=1&filter[source_ref]=eb93b259-1369-4f90-88ce-e68c6ba879a9&filter[name]=OpenShift%20on%20Azure
	bodyBytes, err := request.getRequest(handler)
	if err != nil {
		return nil, err
	}

	var data SourceResponse
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse OpenShift source response from Sources API: %v.", err)
	}

	if data.Meta.Count != 1 {
		log.Info("source does not exist")
		return nil, nil
	}

	return &data.Data[0], nil
}

// GetApplicationTypeID Request the application type ID for Cost Management
func GetApplicationTypeID(handler *SourceHandler, client crhchttp.HTTPClient) (string, error) {
	log := log.WithName("GetApplicationTypeID")
	request := &sourceGetReq{
		client:   client,
		root:     handler.APIURL + handler.Spec.SourcesAPIPath,
		endpoint: ApplicationTypesEndpoint,
		queries:  map[string]string{NameFilterQueryParam: CostManagementAppType},
		errKey:   "application type lookup",
	}

	// Get Application Type ID
	// https://console.redhat.com/api/sources/v1.0/application_types?filter[name]=/insights/platform/cost-management
	bodyBytes, err := request.getRequest(handler)
	if err != nil {
		return "", err
	}

	var data ApplicationTypeResponse
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		log.Error(err, "could not parse output of response")
		return "", fmt.Errorf("Failed to parse Cost Management application type response from Sources API: %v.", err)
	}

	if data.Meta.Count != 1 {
		err = fmt.Errorf("the cost management application type was not found, response count was %d", data.Meta.Count)
		log.Error(err, "unexpected response from application type API")
		return "", fmt.Errorf("Failed to obtain the application type ID for Cost Management: %v.", err)
	}

	return data.Data[0].ID, nil
}

// PostSource Creates a source with the provided name and cluster ID
func PostSource(handler *SourceHandler, client crhchttp.HTTPClient, sourceTypeID string) (*SourceItem, error) {
	log := log.WithName("PostSource")
	request := &sourcePostReq{
		client:   client,
		root:     handler.APIURL + handler.Spec.SourcesAPIPath,
		endpoint: SourcesEndpoint,
		values:   map[string]string{"source_type_id": sourceTypeID, "name": handler.Spec.SourceName, "source_ref": handler.Auth.ClusterID},
		errKey:   "creating the OpenShift source",
	}

	// Post Source
	// https://console.redhat.com/api/sources/v1.0/sources
	// BODY:
	// {"source_type_id": "1", "name": "source_name", "source_ref": "clusterId"}
	bodyBytes, err := request.jsonRequest(handler)
	if err != nil {
		return nil, err
	}

	var data SourceItem
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		log.Error(err, "could not parse output of response")
		return nil, fmt.Errorf("Failed to parse Source response from Sources API: %v.", err)
	}
	return &data, nil
}

// PostApplication Associate a source with an application
func PostApplication(handler *SourceHandler, client crhchttp.HTTPClient, source *SourceItem, appTypeID string) error {
	request := &sourcePostReq{
		client:   client,
		root:     handler.APIURL + handler.Spec.SourcesAPIPath,
		endpoint: ApplicationsEndpoint,
		values:   map[string]string{"source_id": source.ID, "application_type_id": appTypeID},
		errKey:   "creating the OpenShift source with the Cost Management application",
	}

	// Post Source
	// https://console.redhat.com/api/sources/v1.0/applications
	// BODY:
	// {"source_id": "source", "application_type_id": "app_type"}
	_, err := request.jsonRequest(handler)
	if err != nil {
		return err
	}

	return nil
}

// SourceCreate Creates a source with the provided name and cluster ID
func SourceCreate(handler *SourceHandler, client crhchttp.HTTPClient, sourceTypeID string) (*SourceItem, error) {
	log := log.WithName("SourceGetOrCreate")

	// Get App Type ID
	appTypeID, err := GetApplicationTypeID(handler, client)
	if err != nil {
		return nil, err
	}
	log.Info("cost Management application type is " + appTypeID)

	// Create the source
	s, err := PostSource(handler, client, sourceTypeID)
	if err == nil {
		// Associate the source with Cost Management
		err = PostApplication(handler, client, s, appTypeID)
	}

	return s, err
}

// SourceGetOrCreate Check if source exists, if not create the source if specified
func SourceGetOrCreate(handler *SourceHandler, client crhchttp.HTTPClient) (bool, metav1.Time, error) {
	log := log.WithName("SourceGetOrCreate")
	currentTime := metav1.Now()

	// Get Source Type ID
	openShiftSourceTypeID, err := GetSourceTypeID(handler, client)
	if err != nil {
		return false, currentTime, err
	}
	log.Info("OpenShift source type is " + openShiftSourceTypeID)

	// Check if Source exists already
	source, err := CheckSourceExists(handler, client, openShiftSourceTypeID, handler.Spec.SourceName, handler.Auth.ClusterID)
	if err != nil {
		return false, currentTime, err
	}
	if source != nil {
		return true, metav1.Now(), nil
	}
	log.Info("create source = " + strconv.FormatBool(*handler.Spec.CreateSource))
	msg := fmt.Sprintf("No OpenShift source registered with name %s and Cluster ID %s.", handler.Spec.SourceName, handler.Auth.ClusterID)
	if !*handler.Spec.CreateSource {
		return false, metav1.Now(), fmt.Errorf(msg)
	}
	log.Info(msg)

	// Check if cluster ID is registered
	source, err = CheckSourceExists(handler, client, openShiftSourceTypeID, "", handler.Auth.ClusterID)
	if err != nil {
		return false, currentTime, err
	}
	if source != nil {
		errMsg := fmt.Sprintf("This cluster may already be registered because an OpenShift source with Cluster ID %s is already registered with a different name (%s).", handler.Auth.ClusterID, source.Name)
		log.Info(errMsg)
		return false, metav1.Now(), fmt.Errorf(errMsg)
	}

	// Check if source name is already in use
	source, err = CheckSourceExists(handler, client, "", handler.Spec.SourceName, "")
	if err != nil {
		return false, currentTime, err
	}
	if source != nil {
		var errMsg string
		if source.SourceTypeID != openShiftSourceTypeID {
			errMsg = fmt.Sprintf("A non-OpenShift source with name %s is already registered. Source names must be unique.", handler.Spec.SourceName)
		} else {
			errMsg = fmt.Sprintf("An OpenShift source with name %s is registered with a different cluster identifier of %s."+
				" Another cluster may already be registered with this name. Source names must be unique.",
				handler.Spec.SourceName, source.SourceRef)
		}
		log.Info(errMsg)
		return false, metav1.Now(), fmt.Errorf(errMsg)
	}

	// No source is registered with this name
	// No OpenShift source is registered with this cluster ID
	log.Info(fmt.Sprintf("attempting to create OpenShift source registered with name %s and clusterID %s.", handler.Spec.SourceName, handler.Auth.ClusterID))

	// Create the source
	_, err = SourceCreate(handler, client, openShiftSourceTypeID)
	if err != nil {
		return false, metav1.Now(), err
	}

	return true, metav1.Now(), nil
}
