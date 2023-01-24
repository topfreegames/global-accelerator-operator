package elb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type ELBClient interface {
	DescribeLoadBalancers(ctx context.Context, input *elasticloadbalancingv2.DescribeLoadBalancersInput, opts ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error)
}

func NewELBClient(cfg aws.Config) ELBClient {
	return elasticloadbalancingv2.NewFromConfig(cfg)
}

// TODO: Think in a way to cache this in order to avoid doing this every reconciliation
func GetLoadBalancerARNFromDNS(ctx context.Context, elbClient ELBClient, dnsName string) (string, error) {
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(elbClient, &elasticloadbalancingv2.DescribeLoadBalancersInput{
		PageSize: aws.Int32(400),
	})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return "", err
		}
		for _, loadBalancer := range output.LoadBalancers {
			if *loadBalancer.DNSName == dnsName {
				return *loadBalancer.LoadBalancerArn, nil
			}
		}
	}
	return "", fmt.Errorf("loadBalancer not found")
}
