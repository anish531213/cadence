// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package frontend

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/uber/cadence/client/frontend"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/client"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/config"
	"github.com/uber/cadence/common/dynamicconfig"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/resource"
	"github.com/uber/cadence/common/types"
)

type (
	dcRedirectionHandlerSuite struct {
		suite.Suite
		*require.Assertions

		controller               *gomock.Controller
		mockResource             *resource.Test
		mockFrontendHandler      *MockHandler
		mockRemoteFrontendClient *frontend.MockClient
		mockClusterMetadata      *cluster.MockMetadata

		mockDCRedirectionPolicy *MockDCRedirectionPolicy

		domainName             string
		domainID               string
		currentClusterName     string
		alternativeClusterName string
		config                 *Config

		handler *DCRedirectionHandlerImpl
	}
)

func TestDCRedirectionHandlerSuite(t *testing.T) {
	s := new(dcRedirectionHandlerSuite)
	suite.Run(t, s)
}

func (s *dcRedirectionHandlerSuite) SetupSuite() {
}

func (s *dcRedirectionHandlerSuite) TearDownSuite() {
}

func (s *dcRedirectionHandlerSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.domainName = "some random domain name"
	s.domainID = "some random domain ID"
	s.currentClusterName = cluster.TestCurrentClusterName
	s.alternativeClusterName = cluster.TestAlternativeClusterName

	s.mockDCRedirectionPolicy = &MockDCRedirectionPolicy{}

	s.controller = gomock.NewController(s.T())
	s.mockResource = resource.NewTest(s.controller, metrics.Frontend)
	s.mockClusterMetadata = s.mockResource.ClusterMetadata
	s.mockRemoteFrontendClient = s.mockResource.RemoteFrontendClient

	s.mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.currentClusterName).AnyTimes()
	s.mockClusterMetadata.EXPECT().IsGlobalDomainEnabled().Return(true).AnyTimes()

	s.config = NewConfig(
		dynamicconfig.NewCollection(
			dynamicconfig.NewNopClient(),
			s.mockResource.GetLogger(),
		),
		0,
		false,
		false,
	)
	frontendHandler := NewWorkflowHandler(s.mockResource, s.config, nil, client.NewVersionChecker())

	s.mockFrontendHandler = NewMockHandler(s.controller)
	s.handler = NewDCRedirectionHandler(frontendHandler, s.mockResource, s.config, config.DCRedirectionPolicy{})
	s.handler.frontendHandler = s.mockFrontendHandler
	s.handler.redirectionPolicy = s.mockDCRedirectionPolicy
}

func (s *dcRedirectionHandlerSuite) TearDownTest() {
	s.controller.Finish()
	s.mockResource.Finish(s.T())
	s.mockDCRedirectionPolicy.AssertExpectations(s.T())
}

func (s *dcRedirectionHandlerSuite) TestDescribeTaskList() {
	apiName := "DescribeTaskList"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.DescribeTaskListRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.DescribeTaskList(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().DescribeTaskList(gomock.Any(), req).Return(&types.DescribeTaskListResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().DescribeTaskList(gomock.Any(), req).Return(&types.DescribeTaskListResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestDescribeWorkflowExecution() {
	apiName := "DescribeWorkflowExecution"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.DescribeWorkflowExecutionRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.DescribeWorkflowExecution(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().DescribeWorkflowExecution(gomock.Any(), req).Return(&types.DescribeWorkflowExecutionResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().DescribeWorkflowExecution(gomock.Any(), req).Return(&types.DescribeWorkflowExecutionResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestGetWorkflowExecutionHistory() {
	apiName := "GetWorkflowExecutionHistory"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.GetWorkflowExecutionHistoryRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.GetWorkflowExecutionHistory(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().GetWorkflowExecutionHistory(gomock.Any(), req).Return(&types.GetWorkflowExecutionHistoryResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().GetWorkflowExecutionHistory(gomock.Any(), req).Return(&types.GetWorkflowExecutionHistoryResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestListArchivedWorkflowExecutions() {
	apiName := "ListArchivedWorkflowExecutions"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.ListArchivedWorkflowExecutionsRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.ListArchivedWorkflowExecutions(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().ListArchivedWorkflowExecutions(gomock.Any(), req).Return(&types.ListArchivedWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().ListArchivedWorkflowExecutions(gomock.Any(), req).Return(&types.ListArchivedWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestListClosedWorkflowExecutions() {
	apiName := "ListClosedWorkflowExecutions"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.ListClosedWorkflowExecutionsRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.ListClosedWorkflowExecutions(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().ListClosedWorkflowExecutions(gomock.Any(), req).Return(&types.ListClosedWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().ListClosedWorkflowExecutions(gomock.Any(), req).Return(&types.ListClosedWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestListOpenWorkflowExecutions() {
	apiName := "ListOpenWorkflowExecutions"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.ListOpenWorkflowExecutionsRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.ListOpenWorkflowExecutions(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().ListOpenWorkflowExecutions(gomock.Any(), req).Return(&types.ListOpenWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().ListOpenWorkflowExecutions(gomock.Any(), req).Return(&types.ListOpenWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestListWorkflowExecutions() {
	apiName := "ListWorkflowExecutions"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.ListWorkflowExecutionsRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.ListWorkflowExecutions(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().ListWorkflowExecutions(gomock.Any(), req).Return(&types.ListWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().ListWorkflowExecutions(gomock.Any(), req).Return(&types.ListWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestScanWorkflowExecutions() {
	apiName := "ScanWorkflowExecutions"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.ListWorkflowExecutionsRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.ScanWorkflowExecutions(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().ScanWorkflowExecutions(gomock.Any(), req).Return(&types.ListWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().ScanWorkflowExecutions(gomock.Any(), req).Return(&types.ListWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestCountWorkflowExecutions() {
	apiName := "CountWorkflowExecutions"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.CountWorkflowExecutionsRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.CountWorkflowExecutions(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().CountWorkflowExecutions(gomock.Any(), req).Return(&types.CountWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().CountWorkflowExecutions(gomock.Any(), req).Return(&types.CountWorkflowExecutionsResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestPollForActivityTask() {
	apiName := "PollForActivityTask"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.PollForActivityTaskRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.PollForActivityTask(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().PollForActivityTask(gomock.Any(), req).Return(&types.PollForActivityTaskResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().PollForActivityTask(gomock.Any(), req).Return(&types.PollForActivityTaskResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestPollForDecisionTask() {
	apiName := "PollForDecisionTask"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.PollForDecisionTaskRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.PollForDecisionTask(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().PollForDecisionTask(gomock.Any(), req).Return(&types.PollForDecisionTaskResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().PollForDecisionTask(gomock.Any(), req).Return(&types.PollForDecisionTaskResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestQueryWorkflow() {
	apiName := "QueryWorkflow"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.QueryWorkflowRequest{
		Domain:                s.domainName,
		QueryConsistencyLevel: types.QueryConsistencyLevelStrong.Ptr(),
	}
	resp, err := s.handler.QueryWorkflow(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().QueryWorkflow(gomock.Any(), req).Return(&types.QueryWorkflowResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().QueryWorkflow(gomock.Any(), req).Return(&types.QueryWorkflowResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRecordActivityTaskHeartbeat() {
	apiName := "RecordActivityTaskHeartbeat"

	s.mockDCRedirectionPolicy.On("WithDomainIDRedirect",
		s.domainID, apiName, mock.Anything).Return(nil).Times(1)

	token, err := s.handler.tokenSerializer.Serialize(&common.TaskToken{
		DomainID: s.domainID,
	})
	s.Nil(err)
	req := &types.RecordActivityTaskHeartbeatRequest{
		TaskToken: token,
	}
	resp, err := s.handler.RecordActivityTaskHeartbeat(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RecordActivityTaskHeartbeat(gomock.Any(), req).Return(&types.RecordActivityTaskHeartbeatResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RecordActivityTaskHeartbeat(gomock.Any(), req).Return(&types.RecordActivityTaskHeartbeatResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRecordActivityTaskHeartbeatByID() {
	apiName := "RecordActivityTaskHeartbeatByID"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.RecordActivityTaskHeartbeatByIDRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.RecordActivityTaskHeartbeatByID(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RecordActivityTaskHeartbeatByID(gomock.Any(), req).Return(&types.RecordActivityTaskHeartbeatResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RecordActivityTaskHeartbeatByID(gomock.Any(), req).Return(&types.RecordActivityTaskHeartbeatResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRequestCancelWorkflowExecution() {
	apiName := "RequestCancelWorkflowExecution"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.RequestCancelWorkflowExecutionRequest{
		Domain: s.domainName,
	}
	err := s.handler.RequestCancelWorkflowExecution(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RequestCancelWorkflowExecution(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RequestCancelWorkflowExecution(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestResetStickyTaskList() {
	apiName := "ResetStickyTaskList"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.ResetStickyTaskListRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.ResetStickyTaskList(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().ResetStickyTaskList(gomock.Any(), req).Return(&types.ResetStickyTaskListResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().ResetStickyTaskList(gomock.Any(), req).Return(&types.ResetStickyTaskListResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestResetWorkflowExecution() {
	apiName := "ResetWorkflowExecution"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.ResetWorkflowExecutionRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.ResetWorkflowExecution(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().ResetWorkflowExecution(gomock.Any(), req).Return(&types.ResetWorkflowExecutionResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().ResetWorkflowExecution(gomock.Any(), req).Return(&types.ResetWorkflowExecutionResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRespondActivityTaskCanceled() {
	apiName := "RespondActivityTaskCanceled"

	s.mockDCRedirectionPolicy.On("WithDomainIDRedirect",
		s.domainID, apiName, mock.Anything).Return(nil).Times(1)

	token, err := s.handler.tokenSerializer.Serialize(&common.TaskToken{
		DomainID: s.domainID,
	})
	s.Nil(err)
	req := &types.RespondActivityTaskCanceledRequest{
		TaskToken: token,
	}
	err = s.handler.RespondActivityTaskCanceled(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RespondActivityTaskCanceled(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RespondActivityTaskCanceled(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRespondActivityTaskCanceledByID() {
	apiName := "RespondActivityTaskCanceledByID"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.RespondActivityTaskCanceledByIDRequest{
		Domain: s.domainName,
	}
	err := s.handler.RespondActivityTaskCanceledByID(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RespondActivityTaskCanceledByID(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RespondActivityTaskCanceledByID(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRespondActivityTaskCompleted() {
	apiName := "RespondActivityTaskCompleted"

	s.mockDCRedirectionPolicy.On("WithDomainIDRedirect",
		s.domainID, apiName, mock.Anything).Return(nil).Times(1)

	token, err := s.handler.tokenSerializer.Serialize(&common.TaskToken{
		DomainID: s.domainID,
	})
	s.Nil(err)
	req := &types.RespondActivityTaskCompletedRequest{
		TaskToken: token,
	}
	err = s.handler.RespondActivityTaskCompleted(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RespondActivityTaskCompleted(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RespondActivityTaskCompleted(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRespondActivityTaskCompletedByID() {
	apiName := "RespondActivityTaskCompletedByID"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.RespondActivityTaskCompletedByIDRequest{
		Domain: s.domainName,
	}
	err := s.handler.RespondActivityTaskCompletedByID(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RespondActivityTaskCompletedByID(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RespondActivityTaskCompletedByID(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRespondActivityTaskFailed() {
	apiName := "RespondActivityTaskFailed"

	s.mockDCRedirectionPolicy.On("WithDomainIDRedirect",
		s.domainID, apiName, mock.Anything).Return(nil).Times(1)

	token, err := s.handler.tokenSerializer.Serialize(&common.TaskToken{
		DomainID: s.domainID,
	})
	s.Nil(err)
	req := &types.RespondActivityTaskFailedRequest{
		TaskToken: token,
	}
	err = s.handler.RespondActivityTaskFailed(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RespondActivityTaskFailed(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RespondActivityTaskFailed(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRespondActivityTaskFailedByID() {
	apiName := "RespondActivityTaskFailedByID"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.RespondActivityTaskFailedByIDRequest{
		Domain: s.domainName,
	}
	err := s.handler.RespondActivityTaskFailedByID(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RespondActivityTaskFailedByID(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RespondActivityTaskFailedByID(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRespondDecisionTaskCompleted() {
	apiName := "RespondDecisionTaskCompleted"

	s.mockDCRedirectionPolicy.On("WithDomainIDRedirect",
		s.domainID, apiName, mock.Anything).Return(nil).Times(1)

	token, err := s.handler.tokenSerializer.Serialize(&common.TaskToken{
		DomainID: s.domainID,
	})
	s.Nil(err)
	req := &types.RespondDecisionTaskCompletedRequest{
		TaskToken: token,
	}
	resp, err := s.handler.RespondDecisionTaskCompleted(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RespondDecisionTaskCompleted(gomock.Any(), req).Return(&types.RespondDecisionTaskCompletedResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RespondDecisionTaskCompleted(gomock.Any(), req).Return(&types.RespondDecisionTaskCompletedResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRespondDecisionTaskFailed() {
	apiName := "RespondDecisionTaskFailed"

	s.mockDCRedirectionPolicy.On("WithDomainIDRedirect",
		s.domainID, apiName, mock.Anything).Return(nil).Times(1)

	token, err := s.handler.tokenSerializer.Serialize(&common.TaskToken{
		DomainID: s.domainID,
	})
	s.Nil(err)
	req := &types.RespondDecisionTaskFailedRequest{
		TaskToken: token,
	}
	err = s.handler.RespondDecisionTaskFailed(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RespondDecisionTaskFailed(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RespondDecisionTaskFailed(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestRespondQueryTaskCompleted() {
	apiName := "RespondQueryTaskCompleted"

	s.mockDCRedirectionPolicy.On("WithDomainIDRedirect",
		s.domainID, apiName, mock.Anything).Return(nil).Times(1)

	token, err := s.handler.tokenSerializer.SerializeQueryTaskToken(&common.QueryTaskToken{
		DomainID: s.domainID,
	})
	req := &types.RespondQueryTaskCompletedRequest{
		TaskToken: token,
	}
	err = s.handler.RespondQueryTaskCompleted(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().RespondQueryTaskCompleted(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().RespondQueryTaskCompleted(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestSignalWithStartWorkflowExecution() {
	apiName := "SignalWithStartWorkflowExecution"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.SignalWithStartWorkflowExecutionRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.SignalWithStartWorkflowExecution(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().SignalWithStartWorkflowExecution(gomock.Any(), req).Return(&types.StartWorkflowExecutionResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().SignalWithStartWorkflowExecution(gomock.Any(), req).Return(&types.StartWorkflowExecutionResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestSignalWorkflowExecution() {
	apiName := "SignalWorkflowExecution"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.SignalWorkflowExecutionRequest{
		Domain: s.domainName,
	}
	err := s.handler.SignalWorkflowExecution(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().SignalWorkflowExecution(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().SignalWorkflowExecution(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestStartWorkflowExecution() {
	apiName := "StartWorkflowExecution"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.StartWorkflowExecutionRequest{
		Domain: s.domainName,
	}
	resp, err := s.handler.StartWorkflowExecution(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().StartWorkflowExecution(gomock.Any(), req).Return(&types.StartWorkflowExecutionResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().StartWorkflowExecution(gomock.Any(), req).Return(&types.StartWorkflowExecutionResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestTerminateWorkflowExecution() {
	apiName := "TerminateWorkflowExecution"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.TerminateWorkflowExecutionRequest{
		Domain: s.domainName,
	}
	err := s.handler.TerminateWorkflowExecution(context.Background(), req)
	s.Nil(err)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().TerminateWorkflowExecution(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().TerminateWorkflowExecution(gomock.Any(), req).Return(nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}

func (s *dcRedirectionHandlerSuite) TestListTaskListPartitions() {
	apiName := "ListTaskListPartitions"

	s.mockDCRedirectionPolicy.On("WithDomainNameRedirect",
		s.domainName, apiName, mock.Anything).Return(nil).Times(1)

	req := &types.ListTaskListPartitionsRequest{
		Domain: s.domainName,
		TaskList: &types.TaskList{
			Name: "test_tesk_list",
			Kind: types.TaskListKind(0).Ptr(),
		},
	}
	resp, err := s.handler.ListTaskListPartitions(context.Background(), req)
	s.Nil(err)
	// the resp is initialized to nil, since inner function is not called
	s.Nil(resp)

	callFn := s.mockDCRedirectionPolicy.Calls[0].Arguments[2].(func(string) error)
	s.mockFrontendHandler.EXPECT().ListTaskListPartitions(gomock.Any(), req).Return(&types.ListTaskListPartitionsResponse{}, nil).Times(1)
	err = callFn(s.currentClusterName)
	s.Nil(err)
	s.mockRemoteFrontendClient.EXPECT().ListTaskListPartitions(gomock.Any(), req).Return(&types.ListTaskListPartitionsResponse{}, nil).Times(1)
	err = callFn(s.alternativeClusterName)
	s.Nil(err)
}
