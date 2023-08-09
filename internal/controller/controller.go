// Copyright (c) 2023 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package controller

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	appInterfaces "github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"
	"github.com/edgexfoundry/app-record-replay/internal/interfaces"
	"github.com/edgexfoundry/app-record-replay/pkg/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/common"
)

const (
	recordRoute = common.ApiBase + "/record"
	replayRoute = common.ApiBase + "/replay"
	dataRoute   = common.ApiBase + "/data"

	failedRouteMessage = "failed to added %s route for %s method: %v"

	failedRequestJSON              = "Unable to process request JSON"
	failedRecordRequestValidate    = "Record request failed validation: Duration and/or EventLimit must be set"
	failedRecordDurationValidate   = "Record request failed validation: Duration must be > 0 when set"
	failedRecordEventLimitValidate = "Record request failed validation: Event Limit must be > 0 when set"
	failedRecording                = "Recording failed"
	failedReplayRateValidate       = "Replay request failed validation: Replay Rate must be greater than 0"
	failedRepeatCountValidate      = "Replay request failed validation: Repeat Count must be equal or greater than 0"
	failedReplay                   = "Replay failed"
	failedDataCompression          = "failed to compress recorded data of type"
	failedToUncompressData         = "failed to uncompress data"
	failedImportingData            = "Import data failed"
	noDataFound                    = "no recorded data found"

	noCompression       = ""
	zlibCompression     = "ZLIB"
	gzipCompression     = "GZIP"
	contenEncodingGzip  = "gzip"
	contentEncodingZlib = "deflate" // standard value used for zlib is deflate
)

type httpController struct {
	lc          logger.LoggingClient
	dataManager interfaces.DataManager
	appSdk      appInterfaces.ApplicationService
}

// New is the factory function which instantiates a new HTTP Controller
func New(dataManager interfaces.DataManager, appSdk appInterfaces.ApplicationService) interfaces.HttpController {
	return &httpController{
		lc:          appSdk.LoggingClient(),
		dataManager: dataManager,
		appSdk:      appSdk,
	}
}

func (c *httpController) AddRoutes() error {
	if err := c.appSdk.AddRoute(recordRoute, c.startRecording, http.MethodPost); err != nil {
		return fmt.Errorf(failedRouteMessage, recordRoute, http.MethodPost, err)
	}
	if err := c.appSdk.AddRoute(recordRoute, c.recordingStatus, http.MethodGet); err != nil {
		return fmt.Errorf(failedRouteMessage, recordRoute, http.MethodGet, err)
	}
	if err := c.appSdk.AddRoute(recordRoute, c.cancelRecording, http.MethodDelete); err != nil {
		return fmt.Errorf(failedRouteMessage, recordRoute, http.MethodDelete, err)
	}

	if err := c.appSdk.AddRoute(replayRoute, c.startReplay, http.MethodPost); err != nil {
		return fmt.Errorf(failedRouteMessage, replayRoute, http.MethodPost, err)
	}
	if err := c.appSdk.AddRoute(replayRoute, c.replayStatus, http.MethodGet); err != nil {
		return fmt.Errorf(failedRouteMessage, replayRoute, http.MethodGet, err)
	}
	if err := c.appSdk.AddRoute(replayRoute, c.cancelReplay, http.MethodDelete); err != nil {
		return fmt.Errorf(failedRouteMessage, replayRoute, http.MethodDelete, err)
	}

	if err := c.appSdk.AddRoute(dataRoute, c.exportRecordedData, http.MethodGet); err != nil {
		return fmt.Errorf(failedRouteMessage, dataRoute, http.MethodGet, err)
	}
	if err := c.appSdk.AddRoute(dataRoute, c.importRecordedData, http.MethodPost); err != nil {
		return fmt.Errorf(failedRouteMessage, dataRoute, http.MethodPost, err)
	}

	c.lc.Info("Add Record & Replay routes")

	return nil
}

// StartRecording starts a recording session based on the values in the request.
// An error is returned if the request data is incomplete.
func (c *httpController) startRecording(writer http.ResponseWriter, request *http.Request) {
	startRequest := &dtos.RecordRequest{}

	if err := json.NewDecoder(request.Body).Decode(startRequest); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("%s: %v", failedRequestJSON, err)))
		return
	}

	if startRequest.Duration == 0 && startRequest.EventLimit == 0 {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(failedRecordRequestValidate))
		return
	}

	if startRequest.Duration < 0 {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(failedRecordDurationValidate))
		return
	}

	if startRequest.EventLimit < 0 {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(failedRecordEventLimitValidate))
		return
	}

	if err := c.dataManager.StartRecording(*startRequest); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(fmt.Sprintf("%s: %v", failedRecording, err)))
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}

// CancelRecording cancels the current recording session
func (c *httpController) cancelRecording(writer http.ResponseWriter, request *http.Request) {
	if err := c.dataManager.CancelRecording(); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(fmt.Sprintf("failed to cancel recording: %v", err)))
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}

// recordingStatus returns the status of the current recording session
func (c *httpController) recordingStatus(writer http.ResponseWriter, request *http.Request) {
	recordingStatus := c.dataManager.RecordingStatus()

	jsonResponse, err := json.Marshal(recordingStatus)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(fmt.Sprintf("failed to marshal recording status: %s", err)))
		return
	}

	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(jsonResponse)
}

// startReplay starts a replay session based on the values in the request
// An error is returned if the request data is incomplete or a record or replay session is currently running.
func (c *httpController) startReplay(writer http.ResponseWriter, request *http.Request) {
	startRequest := &dtos.ReplayRequest{}

	if err := json.NewDecoder(request.Body).Decode(startRequest); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("%s: %v", failedRequestJSON, err)))
		return
	}

	if startRequest.ReplayRate <= 0 {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(failedReplayRateValidate))
		return
	}

	if startRequest.RepeatCount < 0 {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(failedRepeatCountValidate))
		return
	}

	if err := c.dataManager.StartReplay(*startRequest); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(fmt.Sprintf("%s: %v", failedReplay, err)))
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}

// cancelReplay cancels the current replay session
func (c *httpController) cancelReplay(writer http.ResponseWriter, request *http.Request) {
	if err := c.dataManager.CancelReplay(); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(fmt.Sprintf("failed to cancel replay: %v", err)))
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}

// replayStatus returns the status of the current replay session
func (c *httpController) replayStatus(writer http.ResponseWriter, request *http.Request) {
	replayStatus := c.dataManager.ReplayStatus()

	jsonResponse, err := json.Marshal(replayStatus)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(fmt.Sprintf("failed to marshal replay status: %s", err)))
		return
	}

	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(jsonResponse)
}

// exportRecordedData returns the data for the last record session
// An error is returned if the no record session was run or a record session is currently running
func (c *httpController) exportRecordedData(writer http.ResponseWriter, request *http.Request) {
	recordedData, err := c.dataManager.ExportRecordedData()
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(fmt.Sprintf("failed to export recorded data: %v", err)))
		return
	}

	compression := request.URL.Query().Get("compression")
	switch compression {
	case noCompression:
		jsonResponse, err := json.Marshal(recordedData)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("failed to marshal recorded data"))
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(jsonResponse)

	case zlibCompression:
		writer.Header().Set("Content-Encoding", "ZLIB")
		writer.Header().Set("Content-Type", "application/json")
		zlibWriter := zlib.NewWriter(writer)
		defer zlibWriter.Close()
		err = json.NewEncoder(zlibWriter).Encode(&recordedData)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte(fmt.Sprintf("%s %s: %s", failedDataCompression, zlibCompression, err)))
			return
		}
		writer.WriteHeader(http.StatusOK)

	case gzipCompression:
		writer.Header().Set("Content-Encoding", "GZIP")
		writer.Header().Set("Content-Type", "application/json")
		gZipWriter := gzip.NewWriter(writer)
		defer gZipWriter.Close()
		err = json.NewEncoder(gZipWriter).Encode(&recordedData)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte(fmt.Sprintf("%s %s: %s", failedDataCompression, gzipCompression, err)))
			return
		}
		writer.WriteHeader(http.StatusOK)

	default:
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(fmt.Sprintf("compression format not available: %s", compression)))
		return
	}

}

// importRecordedData imports data from a previously exported record session.
// An error is returned if a record or replay session is currently running or the data is incomplete
func (c *httpController) importRecordedData(writer http.ResponseWriter, request *http.Request) {
	importedRecordedData := &dtos.RecordedData{}
	var reader io.ReadCloser
	var err error
	var overWriteProfilesDevices bool

	contentType := request.Header.Get(common.ContentType)
	if contentType != common.ContentTypeJSON {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("Invalid content type '%s'. Must be application/json", contentType)))
		return
	}

	queryParam := request.URL.Query().Get("overwrite")
	if len(queryParam) == 0 {
		overWriteProfilesDevices = true
	} else {
		overWriteProfilesDevices, err = strconv.ParseBool(queryParam)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(fmt.Sprintf("failed to parse overwrite parameter: %v", err)))
			return
		}
	}

	compression := request.Header.Get("Content-Encoding")
	switch compression {

	case noCompression:
		reader = request.Body

	case contenEncodingGzip:
		reader, err = gzip.NewReader(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(fmt.Sprintf("%s: %s", failedToUncompressData, err)))
			return
		}

	case contentEncodingZlib:
		reader, err = zlib.NewReader(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(fmt.Sprintf("%s: %s", failedToUncompressData, err)))
			return
		}
	default:
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("compression format %s not supported", compression)))
		return

	}
	defer reader.Close()
	err = json.NewDecoder(reader).Decode(&importedRecordedData)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("%s: %v", failedRequestJSON, err)))
		return
	}

	if len(importedRecordedData.RecordedEvents) < 1 {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("%s: no recorded events", noDataFound)))
		return
	}

	if len(importedRecordedData.Devices) < 1 {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("%s: no devices", noDataFound)))
		return
	}

	if len(importedRecordedData.Profiles) < 1 {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("%s: no profiles", noDataFound)))
		return
	}

	if err := c.dataManager.ImportRecordedData(importedRecordedData, overWriteProfilesDevices); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(fmt.Sprintf("%s: %v", failedImportingData, err)))
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}
