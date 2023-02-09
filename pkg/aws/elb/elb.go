package elb

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

type ELBClient interface {
	DescribeLoadBalancers(ctx context.Context, input *elasticloadbalancingv2.DescribeLoadBalancersInput, opts ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error)
	DescribeListeners(ctx context.Context, input *elasticloadbalancingv2.DescribeListenersInput, opts ...func(options *elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeListenersOutput, error)
}

func NewELBClient(cfg aws.Config) ELBClient {
	return elasticloadbalancingv2.NewFromConfig(cfg)
}

// TODO: Think in a way to cache this in order to avoid doing this every reconciliation
func GetLoadBalancersFromDNS(ctx context.Context, elbClient ELBClient, dnsNames []string) ([]types.LoadBalancer, error) {
	loadBalancers := []types.LoadBalancer{}
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(elbClient, &elasticloadbalancingv2.DescribeLoadBalancersInput{
		PageSize: aws.Int32(400),
	})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, loadBalancer := range output.LoadBalancers {
			for i, dnsName := range dnsNames {
				if *loadBalancer.DNSName == dnsName {
					loadBalancers = append(loadBalancers, loadBalancer)
					dnsNames = append(dnsNames[:i], dnsNames[i+1:]...)
					if len(dnsNames) == 0 {
						return loadBalancers, nil
					}
					break
				}
			}
		}
	}
	return nil, fmt.Errorf("at least one of the DNSes wasn't not found")
}

func GetLoadBalancerPorts(ctx context.Context, elbClient ELBClient, loadBalancerARN *string) ([]int32, error) {
	listeners, err := elbClient.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{
		LoadBalancerArn: loadBalancerARN,
	})
	if err != nil {
		return nil, err
	}

	ports := []int32{}
	for _, listener := range listeners.Listeners {
		ports = append(ports, *listener.Port)
	}
	return ports, nil

}
