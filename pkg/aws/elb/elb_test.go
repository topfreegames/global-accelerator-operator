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

func TestGetLoadBalancersFromDNS(t *testing.T) {
	RegisterFailHandler(g.Fail)
	g := NewWithT(t)

	testCases := []struct {
		description                string
		dnsNames                   []string
		expectedARN                []string
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
			dnsNames: []string{
				"a.elb.us-east-1.amazonaws.com",
			},
			expectedError: true,
		},
		{
			description: "should successfully return the lb arn",
			dnsNames: []string{
				"a.elb.us-east-1.amazonaws.com",
				"b.elb.us-east-1.amazonaws.com",
			},
			loadBalancers: []elasticloadbalancingv2types.LoadBalancer{
				{
					DNSName:         aws.String("a.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
				},
				{
					DNSName:         aws.String("b.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/zzz"),
				},
			},
			expectedARN: []string{
				"arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy",
				"arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/zzz",
			},
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

			loadBalancers, err := GetLoadBalancersFromDNS(context.TODO(), fakeELBClient, tc.dnsNames)
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				var loadBalancersARNs []string
				for _, loadBalancer := range loadBalancers {
					loadBalancersARNs = append(loadBalancersARNs, *loadBalancer.LoadBalancerArn)
				}
				g.Expect(loadBalancersARNs).To(BeEquivalentTo(tc.expectedARN))
			}
		})
	}
}

func TestGetLoadBalancerPorts(t *testing.T) {
	RegisterFailHandler(g.Fail)
	g := NewWithT(t)

	testCases := []struct {
		description               string
		loadBalancerARN           string
		expectedError             error
		expectedPorts             []int32
		describeTargetGroupsError error
		loadBalancerListeners     []elasticloadbalancingv2types.Listener
	}{
		{
			description:               "should return error failing to describe target port",
			loadBalancerARN:           "arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy",
			describeTargetGroupsError: errors.New("failed to describe target groups"),
			expectedError:             errors.New("failed to describe target groups"),
		},
		{
			description:     "should retrieve the correct ports",
			loadBalancerARN: "arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy",
			expectedPorts:   []int32{},
		},
		{
			description: "should retrieve the correct ports",
			loadBalancerListeners: []elasticloadbalancingv2types.Listener{
				{
					Port:            aws.Int32(int32(80)),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
				},
				{
					Port:            aws.Int32(int32(443)),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
				},
				{
					Port:            aws.Int32(int32(8443)),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/zzz"),
				},
				{
					Port:            aws.Int32(int32(3000)),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/zzz"),
				},
			},
			loadBalancerARN: "arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy",
			expectedPorts: []int32{
				80,
				443,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fakeELBClient := &fakeelb.MockELBClient{
				MockDescribeListeners: func(ctx context.Context, input *elasticloadbalancingv2.DescribeListenersInput, opts []func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeListenersOutput, error) {
					if tc.describeTargetGroupsError != nil {
						return nil, tc.describeTargetGroupsError
					}
					var loadBalancerListeners []elasticloadbalancingv2types.Listener
					for _, loadBalancerListener := range tc.loadBalancerListeners {
						if *loadBalancerListener.LoadBalancerArn == *input.LoadBalancerArn {
							loadBalancerListeners = append(loadBalancerListeners, loadBalancerListener)
						}
					}
					return &elasticloadbalancingv2.DescribeListenersOutput{
						Listeners: loadBalancerListeners,
					}, nil
				},
			}

			loadBalancerPorts, err := GetLoadBalancerPorts(context.TODO(), fakeELBClient, &tc.loadBalancerARN)
			if tc.expectedError != nil {
				g.Expect(err).Should(BeEquivalentTo(err))
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(loadBalancerPorts).To(BeEquivalentTo(tc.expectedPorts))
			}
		})
	}
}
