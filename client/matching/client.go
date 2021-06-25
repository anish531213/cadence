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

package matching

import (
	"context"
	"sync"
	"time"

	"go.uber.org/yarpc"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/types"
)

var _ Client = (*clientImpl)(nil)

const (
	// DefaultTimeout is the default timeout used to make calls
	DefaultTimeout = time.Minute
	// DefaultLongPollTimeout is the long poll default timeout used to make calls
	DefaultLongPollTimeout = time.Minute * 2
)

type (
	clientIterator func() ([]interface{}, error)

	clientImpl struct {
		timeout         time.Duration
		longPollTimeout time.Duration
		clients         common.ClientCache
		loadBalancer    LoadBalancer
		clientIterator  clientIterator
	}
)

// NewClient creates a new history service TChannel client
func NewClient(
	timeout time.Duration,
	longPollTimeout time.Duration,
	clients common.ClientCache,
	lb LoadBalancer,
	clientIterator clientIterator,
) Client {
	return &clientImpl{
		timeout:         timeout,
		longPollTimeout: longPollTimeout,
		clients:         clients,
		loadBalancer:    lb,
		clientIterator:  clientIterator,
	}
}

func (c *clientImpl) AddActivityTask(
	ctx context.Context,
	request *types.AddActivityTaskRequest,
	opts ...yarpc.CallOption,
) error {
	opts = common.AggregateYarpcOptions(ctx, opts...)
	partition := c.loadBalancer.PickWritePartition(
		request.GetDomainUUID(),
		*request.GetTaskList(),
		persistence.TaskListTypeActivity,
		request.GetForwardedFrom(),
	)
	request.TaskList.Name = partition
	client, err := c.getClientForTaskList(partition)
	if err != nil {
		return err
	}
	ctx, cancel := c.createContext(ctx)
	defer cancel()
	return client.AddActivityTask(ctx, request, opts...)
}

func (c *clientImpl) AddDecisionTask(
	ctx context.Context,
	request *types.AddDecisionTaskRequest,
	opts ...yarpc.CallOption,
) error {
	opts = common.AggregateYarpcOptions(ctx, opts...)
	partition := c.loadBalancer.PickWritePartition(
		request.GetDomainUUID(),
		*request.GetTaskList(),
		persistence.TaskListTypeDecision,
		request.GetForwardedFrom(),
	)
	request.TaskList.Name = partition
	client, err := c.getClientForTaskList(request.TaskList.GetName())
	if err != nil {
		return err
	}
	ctx, cancel := c.createContext(ctx)
	defer cancel()
	return client.AddDecisionTask(ctx, request, opts...)
}

func (c *clientImpl) PollForActivityTask(
	ctx context.Context,
	request *types.MatchingPollForActivityTaskRequest,
	opts ...yarpc.CallOption,
) (*types.PollForActivityTaskResponse, error) {
	opts = common.AggregateYarpcOptions(ctx, opts...)
	partition := c.loadBalancer.PickReadPartition(
		request.GetDomainUUID(),
		*request.PollRequest.GetTaskList(),
		persistence.TaskListTypeActivity,
		request.GetForwardedFrom(),
	)
	request.PollRequest.TaskList.Name = partition
	client, err := c.getClientForTaskList(request.PollRequest.TaskList.GetName())
	if err != nil {
		return nil, err
	}
	ctx, cancel := c.createLongPollContext(ctx)
	defer cancel()
	return client.PollForActivityTask(ctx, request, opts...)
}

func (c *clientImpl) PollForDecisionTask(
	ctx context.Context,
	request *types.MatchingPollForDecisionTaskRequest,
	opts ...yarpc.CallOption,
) (*types.MatchingPollForDecisionTaskResponse, error) {
	opts = common.AggregateYarpcOptions(ctx, opts...)
	partition := c.loadBalancer.PickReadPartition(
		request.GetDomainUUID(),
		*request.PollRequest.GetTaskList(),
		persistence.TaskListTypeDecision,
		request.GetForwardedFrom(),
	)
	request.PollRequest.TaskList.Name = partition
	client, err := c.getClientForTaskList(request.PollRequest.TaskList.GetName())
	if err != nil {
		return nil, err
	}
	ctx, cancel := c.createLongPollContext(ctx)
	defer cancel()
	return client.PollForDecisionTask(ctx, request, opts...)
}

func (c *clientImpl) QueryWorkflow(
	ctx context.Context,
	request *types.MatchingQueryWorkflowRequest,
	opts ...yarpc.CallOption,
) (*types.QueryWorkflowResponse, error) {
	opts = common.AggregateYarpcOptions(ctx, opts...)
	partition := c.loadBalancer.PickReadPartition(
		request.GetDomainUUID(),
		*request.GetTaskList(),
		persistence.TaskListTypeDecision,
		request.GetForwardedFrom(),
	)
	request.TaskList.Name = partition
	client, err := c.getClientForTaskList(request.TaskList.GetName())
	if err != nil {
		return nil, err
	}
	ctx, cancel := c.createContext(ctx)
	defer cancel()
	return client.QueryWorkflow(ctx, request, opts...)
}

func (c *clientImpl) RespondQueryTaskCompleted(
	ctx context.Context,
	request *types.MatchingRespondQueryTaskCompletedRequest,
	opts ...yarpc.CallOption,
) error {
	opts = common.AggregateYarpcOptions(ctx, opts...)
	client, err := c.getClientForTaskList(request.TaskList.GetName())
	if err != nil {
		return err
	}
	ctx, cancel := c.createContext(ctx)
	defer cancel()
	return client.RespondQueryTaskCompleted(ctx, request, opts...)
}

func (c *clientImpl) CancelOutstandingPoll(
	ctx context.Context,
	request *types.CancelOutstandingPollRequest,
	opts ...yarpc.CallOption,
) error {
	opts = common.AggregateYarpcOptions(ctx, opts...)
	client, err := c.getClientForTaskList(request.TaskList.GetName())
	if err != nil {
		return err
	}
	ctx, cancel := c.createContext(ctx)
	defer cancel()
	return client.CancelOutstandingPoll(ctx, request, opts...)
}

func (c *clientImpl) DescribeTaskList(
	ctx context.Context,
	request *types.MatchingDescribeTaskListRequest,
	opts ...yarpc.CallOption,
) (*types.DescribeTaskListResponse, error) {
	opts = common.AggregateYarpcOptions(ctx, opts...)
	client, err := c.getClientForTaskList(request.DescRequest.TaskList.GetName())
	if err != nil {
		return nil, err
	}
	ctx, cancel := c.createContext(ctx)
	defer cancel()
	return client.DescribeTaskList(ctx, request, opts...)
}

func (c *clientImpl) ListTaskListPartitions(
	ctx context.Context,
	request *types.MatchingListTaskListPartitionsRequest,
	opts ...yarpc.CallOption,
) (*types.ListTaskListPartitionsResponse, error) {
	opts = common.AggregateYarpcOptions(ctx, opts...)
	client, err := c.getClientForTaskList(request.TaskList.GetName())
	if err != nil {
		return nil, err
	}
	ctx, cancel := c.createContext(ctx)
	defer cancel()
	return client.ListTaskListPartitions(ctx, request, opts...)
}

func (c *clientImpl) GetTaskListsByDomain(
	ctx context.Context,
	request *types.MatchingGetTaskListsByDomainRequest,
	opts ...yarpc.CallOption,
) (*types.GetTaskListsByDomainResponse, error) {
	opts = common.AggregateYarpcOptions(ctx, opts...)
	clients, err := c.clientIterator()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(len(clients))
	respChan := make(chan *types.GetTaskListsByDomainResponse, len(clients))
	errChan := make(chan error, 1)
	for _, cl := range clients {
		go func(client Client, request *types.MatchingGetTaskListsByDomainRequest) {
			defer wg.Done()

			resp, err := c.getTaskListsByDomain(cl.(Client), ctx, request, opts...)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}
			respChan <- resp
		}(cl.(Client), request)
	}
	wg.Wait()
	close(respChan)
	close(errChan)
	if len(errChan) > 0 {
		err = <-errChan
		return nil, err
	}

	// De-dup task list within a domain
	taskListMap := make(map[string]struct{})
	for resp := range respChan {
		for _, tl := range resp.TaskListNames {
			if _, ok := taskListMap[tl]; !ok {
				taskListMap[tl] = struct{}{}
			}
		}
	}
	var taskListNames []string
	for tl := range taskListMap {
		taskListNames = append(taskListNames, tl)
	}

	return &types.GetTaskListsByDomainResponse{TaskListNames: taskListNames}, nil
}

func (c *clientImpl) createContext(
	parent context.Context,
) (context.Context, context.CancelFunc) {
	if parent == nil {
		return context.WithTimeout(context.Background(), c.timeout)
	}
	return context.WithTimeout(parent, c.timeout)
}

func (c *clientImpl) createLongPollContext(
	parent context.Context,
) (context.Context, context.CancelFunc) {
	if parent == nil {
		return context.WithTimeout(context.Background(), c.longPollTimeout)
	}
	return context.WithTimeout(parent, c.longPollTimeout)
}

func (c *clientImpl) getClientForTaskList(key string) (Client, error) {
	client, err := c.clients.GetClientForKey(key)
	if err != nil {
		return nil, err
	}
	return client.(Client), nil
}

func (c *clientImpl) getTaskListsByDomain(
	cl Client,
	ctx context.Context,
	request *types.MatchingGetTaskListsByDomainRequest,
	opts ...yarpc.CallOption,
) (*types.GetTaskListsByDomainResponse, error) {
	ctx, cancel := c.createContext(ctx)
	defer cancel()

	return cl.GetTaskListsByDomain(ctx, request, opts...)
}
