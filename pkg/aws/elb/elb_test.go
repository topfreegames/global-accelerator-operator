package elb

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elasticloadbalancingv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	g "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	fakeelb "github.com/topfreegames/global-accelerator-operator/pkg/aws/elb/fake"
	"testing"
)

func TestGetLoadBalancerARNFromDNS(t *testing.T) {
	RegisterFailHandler(g.Fail)
	g := NewWithT(t)

	testCases := []struct {
		description                string
		dnsName                    string
		expectedARN                string
		expectedError              bool
		describeLoadBalancersError error
		loadBalancers              []elasticloadbalancingv2types.LoadBalancer
	}{
		{
			description:                "should return err when failing to describe lbs",
			describeLoadBalancersError: errors.New("failed to describe lbs"),
			expectedError:              true,
		},
		{
			description: "should return err when failing to find the lb with dnsName",
			loadBalancers: []elasticloadbalancingv2types.LoadBalancer{
				{
					DNSName:         aws.String("b.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/zzz"),
				},
			},
			dnsName:       "a.elb.us-east-1.amazonaws.com",
			expectedError: true,
		},
		{
			description: "should successfully return the lb arn",
			dnsName:     "a.elb.us-east-1.amazonaws.com",
			loadBalancers: []elasticloadbalancingv2types.LoadBalancer{
				{
					DNSName:         aws.String("a.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
				},
			},
			expectedARN: "arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fakeELBClient := &fakeelb.MockELBClient{
				MockDescribeLoadBalancers: func(ctx context.Context, input *elasticloadbalancingv2.DescribeLoadBalancersInput, opts []func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error) {
					if tc.describeLoadBalancersError != nil {
						return nil, tc.describeLoadBalancersError
					}
					return &elasticloadbalancingv2.DescribeLoadBalancersOutput{
						LoadBalancers: tc.loadBalancers,
					}, nil
				},
			}

			loadBalancerARN, err := GetLoadBalancerARNFromDNS(context.TODO(), fakeELBClient, tc.dnsName)
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(loadBalancerARN).To(BeEquivalentTo(tc.expectedARN))
			}
		})
	}
}
