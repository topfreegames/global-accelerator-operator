package fake

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/globalaccelerator"
)

type MockGlobalAcceleratorClient struct {
	MockCreateAccelerator   func(ctx context.Context, input *globalaccelerator.CreateAcceleratorInput, opts []func(*globalaccelerator.Options)) (*globalaccelerator.CreateAcceleratorOutput, error)
	MockCreateEndpointGroup func(ctx context.Context, input *globalaccelerator.CreateEndpointGroupInput, opts []func(*globalaccelerator.Options)) (*globalaccelerator.CreateEndpointGroupOutput, error)
	MockCreateListener      func(ctx context.Context, input *globalaccelerator.CreateListenerInput, opts []func(*globalaccelerator.Options)) (*globalaccelerator.CreateListenerOutput, error)
	MockDescribeAccelerator func(ctx context.Context, input *globalaccelerator.DescribeAcceleratorInput, opts []func(*globalaccelerator.Options)) (*globalaccelerator.DescribeAcceleratorOutput, error)
	MockListAccelerators    func(ctx context.Context, input *globalaccelerator.ListAcceleratorsInput, opts []func(*globalaccelerator.Options)) (*globalaccelerator.ListAcceleratorsOutput, error)
	MockListListeners       func(ctx context.Context, input *globalaccelerator.ListListenersInput, opts []func(*globalaccelerator.Options)) (*globalaccelerator.ListListenersOutput, error)
	MockListTagsForResource func(ctx context.Context, input *globalaccelerator.ListTagsForResourceInput, opts []func(*globalaccelerator.Options)) (*globalaccelerator.ListTagsForResourceOutput, error)
	MockUntagResource       func(ctx context.Context, input *globalaccelerator.UntagResourceInput, opts []func(*globalaccelerator.Options)) (*globalaccelerator.UntagResourceOutput, error)
	MockUpdateEndpointGroup func(ctx context.Context, input *globalaccelerator.UpdateEndpointGroupInput, opts []func(*globalaccelerator.Options)) (*globalaccelerator.UpdateEndpointGroupOutput, error)
}

func (m *MockGlobalAcceleratorClient) CreateAccelerator(ctx context.Context, input *globalaccelerator.CreateAcceleratorInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.CreateAcceleratorOutput, error) {
	return m.MockCreateAccelerator(ctx, input, opts)
}

func (m *MockGlobalAcceleratorClient) CreateEndpointGroup(ctx context.Context, input *globalaccelerator.CreateEndpointGroupInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.CreateEndpointGroupOutput, error) {
	return m.MockCreateEndpointGroup(ctx, input, opts)
}
func (m *MockGlobalAcceleratorClient) CreateListener(ctx context.Context, input *globalaccelerator.CreateListenerInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.CreateListenerOutput, error) {
	return m.MockCreateListener(ctx, input, opts)
}
func (m *MockGlobalAcceleratorClient) DescribeAccelerator(ctx context.Context, input *globalaccelerator.DescribeAcceleratorInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.DescribeAcceleratorOutput, error) {
	return m.MockDescribeAccelerator(ctx, input, opts)
}
func (m *MockGlobalAcceleratorClient) ListAccelerators(ctx context.Context, input *globalaccelerator.ListAcceleratorsInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.ListAcceleratorsOutput, error) {
	return m.MockListAccelerators(ctx, input, opts)
}
func (m *MockGlobalAcceleratorClient) ListListeners(ctx context.Context, input *globalaccelerator.ListListenersInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.ListListenersOutput, error) {
	return m.MockListListeners(ctx, input, opts)
}
func (m *MockGlobalAcceleratorClient) ListTagsForResource(ctx context.Context, input *globalaccelerator.ListTagsForResourceInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.ListTagsForResourceOutput, error) {
	return m.MockListTagsForResource(ctx, input, opts)
}
func (m *MockGlobalAcceleratorClient) UntagResource(ctx context.Context, input *globalaccelerator.UntagResourceInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.UntagResourceOutput, error) {
	return m.MockUntagResource(ctx, input, opts)
}
func (m *MockGlobalAcceleratorClient) UpdateEndpointGroup(ctx context.Context, input *globalaccelerator.UpdateEndpointGroupInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.UpdateEndpointGroupOutput, error) {
	return m.MockUpdateEndpointGroup(ctx, input, opts)
}
