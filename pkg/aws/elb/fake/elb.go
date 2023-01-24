package fake

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type MockELBClient struct {
	MockDescribeLoadBalancers func(ctx context.Context, input *elasticloadbalancingv2.DescribeLoadBalancersInput, opts []func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error)
}

func (m *MockELBClient) DescribeLoadBalancers(ctx context.Context, input *elasticloadbalancingv2.DescribeLoadBalancersInput, opts ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error) {
	return m.MockDescribeLoadBalancers(ctx, input, opts)
}
