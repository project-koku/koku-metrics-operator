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

package sources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/project-koku/korekuta-operator-go/crhchttp"
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
	SourceTypeID string
	SourceRef    string
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

// type SourceHandler struct {
// 	Log     *logr.Logger
// 	CostCfg *crhchttp.CostManagementConfig
// 	source  source
// }

// type source struct {
// 	sourceType      SourceTypeDataItem
// 	sourceItem      SourceItem
// 	applicationType ApplicationTypeDataItem
// }

// GetSourceTypeID Request the source type ID for OpenShift
func GetSourceTypeID(costConfig *crhchttp.CostManagementConfig) (string, error) {
	log := costConfig.Log.WithValues("costmanagement", "GetSourceTypeID")
	client := crhchttp.GetClient(costConfig)
	sourceAPIRoot := costConfig.APIURL + costConfig.SourcesAPIPath
	var emptyBytes []byte
	emptyBody := bytes.NewBuffer(emptyBytes)

	// Get Source Type ID
	// https://cloud.redhat.com/api/sources/v1.0/source_types?filter[name]=openshift
	sourceTypeURI := sourceAPIRoot + SourceTypesEndpoint + "?" + NameFilterQueryParam + "=" + OpenShiftSourceType
	req, err := crhchttp.SetupRequest(costConfig, "", "GET", sourceTypeURI, emptyBody)
	if err != nil {
		return "", fmt.Errorf("Failed to construct query for OpenShift source type from Sources API: %v", err)
	}
	log.Info("GET Request - " + sourceTypeURI)
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "Could not send request")
		return "", fmt.Errorf("Failed to query OpenShift source type from Sources API: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("HTTP Response Status:", resp.StatusCode, http.StatusText(resp.StatusCode))

	bodyBytes, err := crhchttp.ProcessResponse(log, resp)
	if err != nil {
		log.Error(err, "The following error occurred")
		return "", fmt.Errorf("Failed to process the response for obtaining the OpenShift source type ID: %v", err)
	}
	bodyString := string(bodyBytes)
	log.Info("Response body: ")
	log.Info(bodyString)

	var data SourceTypeResponse
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		log.Error(err, "Could not parse output of response")
		return "", fmt.Errorf("Failed to parse OpenShift source type response from Sources API: %v", err)
	}

	if data.Meta.Count != 1 {
		err = fmt.Errorf("the openshift source type was not found, response count was %d", data.Meta.Count)
		log.Error(err, "Unexpected response from source type API.")
		return "", fmt.Errorf("Failed to obtain the source type ID for OpenShift: %v", err)
	}

	return data.Data[0].ID, nil
}

// CheckSourceExists Determine if the source exists with given parameters
func CheckSourceExists(costConfig *crhchttp.CostManagementConfig, sourceTypeID string, name string, sourceRef string) (*SourceItem, error) {
	log := costConfig.Log.WithValues("costmanagement", "CheckSourceExists")
	client := crhchttp.GetClient(costConfig)
	sourceAPIRoot := costConfig.APIURL + costConfig.SourcesAPIPath
	var emptyBytes []byte
	emptyBody := bytes.NewBuffer(emptyBytes)

	// Check if Source exists already
	// https://cloud.redhat.com/api/sources/v1.0/sources?filter[source_type_id]=1&filter[source_ref]=eb93b259-1369-4f90-88ce-e68c6ba879a9&filter[name]=OpenShift%20on%20Azure
	sourceURI := sourceAPIRoot + SourcesEndpoint
	req, err := crhchttp.SetupRequest(costConfig, "", "GET", sourceURI, emptyBody)
	if err != nil {
		return nil, fmt.Errorf("Failed to construct query for OpenShift sources from Sources API: %v", err)
	}
	q := req.URL.Query()

	if sourceTypeID != "" {
		q.Add(SourceTypeIDFilterQueryParam, sourceTypeID)
	}
	if name != "" {
		q.Add(NameFilterQueryParam, name)
	}
	if sourceRef != "" {
		q.Add(SourceRefFilterQueryParam, sourceRef)
	}
	req.URL.RawQuery = q.Encode()

	log.Info("GET Request - " + q.Encode())
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "Could not send request")
		return nil, fmt.Errorf("Failed to query OpenShift sources from Sources API: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("HTTP Response Status:", resp.StatusCode, http.StatusText(resp.StatusCode))

	bodyBytes, err := crhchttp.ProcessResponse(log, resp)
	if err != nil {
		log.Error(err, "The following error occurred")
		return nil, fmt.Errorf("Failed to process the response for obtaining the OpenShift source: %v", err)
	}
	bodyString := string(bodyBytes)
	log.Info("Response body: ")
	log.Info(bodyString)

	var data SourceResponse
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		log.Error(err, "Could not parse output of response")
		return nil, fmt.Errorf("Failed to parse OpenShift source type response from Sources API: %v", err)
	}

	if data.Meta.Count != 1 {
		log.Info("Source does not exist.")
		return nil, nil
	}

	return &data.Data[0], nil
}

// GetApplicationTypeID Request the application type ID for Cost Management
func GetApplicationTypeID(costConfig *crhchttp.CostManagementConfig) (string, error) {
	log := costConfig.Log.WithValues("costmanagement", "GetApplicationTypeID")
	client := crhchttp.GetClient(costConfig)
	sourceAPIRoot := costConfig.APIURL + costConfig.SourcesAPIPath
	var emptyBytes []byte
	emptyBody := bytes.NewBuffer(emptyBytes)

	// Get Application Type ID
	// https://cloud.redhat.com/api/sources/v1.0/application_types?filter[name]=/insights/platform/cost-management
	appTypeURI := sourceAPIRoot + ApplicationTypesEndpoint + "?" + NameFilterQueryParam + "=" + CostManagementAppType
	req, err := crhchttp.SetupRequest(costConfig, "", "GET", appTypeURI, emptyBody)
	if err != nil {
		return "", fmt.Errorf("Failed to construct query for Cost Management application type from Sources API: %v", err)
	}
	log.Info("GET Request - " + appTypeURI)
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "Could not send request")
		return "", fmt.Errorf("Failed to query Cost Management application type from Sources API: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("HTTP Response Status:", resp.StatusCode, http.StatusText(resp.StatusCode))

	bodyBytes, err := crhchttp.ProcessResponse(log, resp)
	if err != nil {
		log.Error(err, "The following error occurred")
		return "", fmt.Errorf("Failed to process the response for obtaining the Cost Management application type ID: %v", err)
	}
	bodyString := string(bodyBytes)
	log.Info("Response body: ")
	log.Info(bodyString)

	var data ApplicationTypeResponse
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		log.Error(err, "Could not parse output of response")
		return "", fmt.Errorf("Failed to parse Cost Management application type response from Sources API: %v", err)
	}

	if data.Meta.Count != 1 {
		err = fmt.Errorf("the cost management application type was not found, response count was %d", data.Meta.Count)
		log.Error(err, "Unexpected response from application type API.")
		return "", fmt.Errorf("Failed to obtain the application type ID for Cost Management: %v", err)
	}

	return data.Data[0].ID, nil
}

// PostSource Creates a source with the provided name and cluster ID
func PostSource(costConfig *crhchttp.CostManagementConfig, sourceTypeID string) (*SourceItem, error) {
	log := costConfig.Log.WithValues("costmanagement", "PostSource")
	client := crhchttp.GetClient(costConfig)
	sourceAPIRoot := costConfig.APIURL + costConfig.SourcesAPIPath

	// Post Source
	// https://cloud.redhat.com/api/sources/v1.0/sources
	// BODY:
	// {"source_type_id": "1", "name": "source_name", "source_ref": "clusterId"}
	sourceURI := sourceAPIRoot + SourcesEndpoint
	values := map[string]string{"source_type_id": sourceTypeID, "name": costConfig.SourceName, "source_ref": costConfig.ClusterID}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("Failed to construct body for call to create a Source with the Sources API: %v", err)
	}
	req, err := crhchttp.SetupRequest(costConfig, "application/json", "POST", sourceURI, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("Failed to construct call to create a Source with the Sources API: %v", err)
	}
	log.Info("POST Request - " + sourceURI)
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "Could not send request")
		return nil, fmt.Errorf("Failed to send request to create a Source with the Sources API: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("HTTP Response Status:", resp.StatusCode, http.StatusText(resp.StatusCode))

	bodyBytes, err := crhchttp.ProcessResponse(log, resp)
	if err != nil {
		log.Error(err, "The following error occurred")
		return nil, fmt.Errorf("Failed to process the response for creating the OpenShift source: %v", err)
	}
	bodyString := string(bodyBytes)
	log.Info("Response body: ")
	log.Info(bodyString)

	var data SourceItem
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		log.Error(err, "Could not parse output of response")
		return nil, fmt.Errorf("Failed to parse Source response from Sources API: %v", err)
	}
	return &data, nil
}

// PostApplication Associate a source with an application
func PostApplication(costConfig *crhchttp.CostManagementConfig, source *SourceItem, appTypeID string) (*SourceItem, error) {
	log := costConfig.Log.WithValues("costmanagement", "PostApplication")
	client := crhchttp.GetClient(costConfig)
	sourceAPIRoot := costConfig.APIURL + costConfig.SourcesAPIPath

	// Post Source
	// https://cloud.redhat.com/api/sources/v1.0/applications
	// BODY:
	// {"source_id": "source", "application_type_id": "app_type"}
	applicationURI := sourceAPIRoot + ApplicationsEndpoint
	values := map[string]string{"source_id": source.ID, "application_type_id": appTypeID}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("Failed to construct body for call to associate an Application with a Source using the Sources API: %v", err)
	}
	req, err := crhchttp.SetupRequest(costConfig, "application/json", "POST", applicationURI, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("Failed to construct call to create an Application for a Source using the Sources API: %v", err)
	}
	log.Info("POST Request - " + applicationURI)
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "Could not send request")
		return nil, fmt.Errorf("Failed to send request to create an Application with the Sources API: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("HTTP Response Status:", resp.StatusCode, http.StatusText(resp.StatusCode))

	bodyBytes, err := crhchttp.ProcessResponse(log, resp)
	if err != nil {
		log.Error(err, "The following error occurred")
		return nil, fmt.Errorf("Failed to process the response for associating the OpenShift source with the Cost Management application: %v", err)
	}
	bodyString := string(bodyBytes)
	log.Info("Response body: ")
	log.Info(bodyString)

	return source, nil
}

// SourceCreate Creates a source with the provided name and cluster ID
func SourceCreate(costConfig *crhchttp.CostManagementConfig, sourceTypeID string) (*SourceItem, error) {
	log := costConfig.Log.WithValues("costmanagement", "SourceGetOrCreate")

	// Get App Type ID
	appTypeID, err := GetApplicationTypeID(costConfig)
	if err != nil {
		return nil, err
	}
	log.Info("Cost Management application type is " + appTypeID)

	// Create the source
	source, err := PostSource(costConfig, sourceTypeID)
	if err == nil {
		// Associate the source with Cost Management
		source, err = PostApplication(costConfig, source, appTypeID)
	}

	return source, err
}

// SourceGetOrCreate Check if source exists, if not create the source if specified
func SourceGetOrCreate(costConfig *crhchttp.CostManagementConfig) (bool, metav1.Time, error) {
	log := costConfig.Log.WithValues("costmanagement", "SourceGetOrCreate")
	currentTime := metav1.Now()

	// Get Source Type ID
	openShiftSourceTypeID, err := GetSourceTypeID(costConfig)
	if err != nil {
		return false, currentTime, err
	}
	log.Info("OpenShift source type is " + openShiftSourceTypeID)

	// Check if Source exists already
	source, err := CheckSourceExists(costConfig, openShiftSourceTypeID, costConfig.SourceName, costConfig.ClusterID)
	if err != nil {
		return false, currentTime, err
	}
	if source != nil {
		return true, metav1.Now(), nil
	}
	log.Info("Create source = " + strconv.FormatBool(costConfig.CreateSource))
	if !costConfig.CreateSource {
		return false, metav1.Now(), fmt.Errorf("no OpenShift source registered with name %s and Cluster ID %s", costConfig.SourceName, costConfig.ClusterID)
	}
	log.Info("No OpenShift source registered with name " + costConfig.SourceName + " and Cluster ID " + costConfig.ClusterID + ".")

	// Check if cluster ID is registered
	source, err = CheckSourceExists(costConfig, openShiftSourceTypeID, "", costConfig.ClusterID)
	if err != nil {
		return false, currentTime, err
	}
	if source != nil {
		var errMsg string
		errMsg = "OpenShift source with Cluster ID " + costConfig.ClusterID + " is registered with different name."
		errMsg += " The cluster may already be registered with a different name."
		log.Info(errMsg)
		return false, metav1.Now(), fmt.Errorf(errMsg)
	}

	// Check if source name is already in use
	source, err = CheckSourceExists(costConfig, "", costConfig.SourceName, "")
	if err != nil {
		return false, currentTime, err
	}
	if source != nil {
		var errMsg string
		if source.SourceTypeID != openShiftSourceTypeID {
			errMsg = "A non-OpenShift source with name " + costConfig.SourceName + " is already registered. Source names must be unique."
		} else {
			errMsg = "An OpenShift source with name " + costConfig.SourceName + " is registered with different cluster identifier of " + source.SourceRef + "."
			errMsg += " Another cluster may already be registered with a this name. Source names must be unique."
		}
		log.Info(errMsg)
		return false, metav1.Now(), fmt.Errorf(errMsg)
	}

	// No source is registered with this name
	// No OpenShift source is registered with this cluster ID
	log.Info("Attempting to create OpenShift source registered with name " + costConfig.SourceName + " and Cluster ID " + costConfig.ClusterID + ".")

	// Create the source
	_, err = SourceCreate(costConfig, openShiftSourceTypeID)
	if err != nil {
		return false, metav1.Now(), err
	}

	return true, metav1.Now(), nil
}
