//
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

package data

import (
	"context"

	appInterfaces "github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"
	"github.com/edgexfoundry/app-record-replay/internal/interfaces"
	"github.com/edgexfoundry/app-record-replay/pkg/dtos"
	coreDtos "github.com/edgexfoundry/go-mod-core-contracts/v3/dtos"
)

// NewManager is the factory function which instantiates a Data Manager
func NewManager(ctx context.Context, service appInterfaces.ApplicationService) interfaces.DataManager {
	return &dataManager{
		dataChan: make(chan []coreDtos.Event, 1),
		ctx:      ctx,
		appSvc:   service,
	}
}

// dataManager implements interface that records and replays captured data
type dataManager struct {
	dataChan chan []coreDtos.Event
	ctx      context.Context
	appSvc   appInterfaces.ApplicationService
}

// StartRecording starts a recording session based on the values in the request.
// An error is returned if the request data is incomplete or a record or replay session is currently running.
func (m *dataManager) StartRecording(request dtos.RecordRequest) error {
	//TODO implement me using TDD
	panic("implement me")
}

// CancelRecording cancels the current recording session
func (m *dataManager) CancelRecording() {
	//TODO implement me using TDD
	panic("implement me")
}

// RecordingStatus returns the status of the current recording session
func (m *dataManager) RecordingStatus() dtos.RecordStatus {
	//TODO implement me using TDD
	panic("implement me")
}

// StartReplay starts a replay session based on the values in the request
// An error is returned if the request data is incomplete or a record or replay session is currently running.
func (m *dataManager) StartReplay(request dtos.ReplayRequest) error {
	//TODO implement me using TDD
	panic("implement me")
}

// CancelReplay cancels the current replay session
func (m *dataManager) CancelReplay() {
	//TODO implement me using TDD
	panic("implement me")
}

// ReplayStatus returns the status of the current replay session
func (m *dataManager) ReplayStatus() dtos.ReplayStatus {
	//TODO implement me using TDD
	panic("implement me")
}

// ExportRecordedData returns the data for the last record session
// An error is returned if the no record session was run or a record session is currently running
func (m *dataManager) ExportRecordedData() (dtos.RecordedData, error) {
	//TODO implement me using TDD
	panic("implement me")
}

// ImportRecordedData imports data from a previously exported record session.
// An error is returned if a record or replay session is currently running or the data is incomplete
func (m *dataManager) ImportRecordedData(data dtos.RecordedData) error {
	//TODO implement me using TDD
	panic("implement me")
}

// Pipeline functions

// countEvents counts the number of Events the function receives.
func (m *dataManager) countEvents(_ appInterfaces.AppFunctionContext, data any) (bool, interface{}) {
	//TODO implement me using TDD
	return false, nil
}

// processBatchedData processes the batched data for the current recording session
func (m *dataManager) processBatchedData(_ appInterfaces.AppFunctionContext, data any) (bool, interface{}) {
	//TODO implement me using TDD
	return false, nil
}