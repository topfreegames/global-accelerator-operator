package fake

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type MockELBClient struct {
	MockDescribeLoadBalancers func(ctx context.Context, input *elasticloadbalancingv2.DescribeLoadBalancersInput, opts []func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error)
	MockDescribeListeners     func(ctx context.Context, input *elasticloadbalancingv2.DescribeListenersInput, opts []func(options *elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeListenersOutput, error)
}

func (m *MockELBClient) DescribeLoadBalancers(ctx context.Context, input *elasticloadbalancingv2.DescribeLoadBalancersInput, opts ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error) {
	return m.MockDescribeLoadBalancers(ctx, input, opts)
}

func (m *MockELBClient) DescribeListeners(ctx context.Context, input *elasticloadbalancingv2.DescribeListenersInput, opts ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeListenersOutput, error) {
	return m.MockDescribeListeners(ctx, input, opts)
}
