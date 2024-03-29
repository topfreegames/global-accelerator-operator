package globalacceleratorawswildlifeio

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elasticloadbalancingv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/pkg/errors"
	globalacceleratorawswildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/globalaccelerator.aws.wildlife.io/v1alpha1"
	"github.com/topfreegames/global-accelerator-operator/pkg/aws/elb"
	fakeelb "github.com/topfreegames/global-accelerator-operator/pkg/aws/elb/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	globalacceleratorsdk "github.com/aws/aws-sdk-go-v2/service/globalaccelerator"
	globalacceleratortypes "github.com/aws/aws-sdk-go-v2/service/globalaccelerator/types"
	g "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/topfreegames/global-accelerator-operator/pkg/aws/globalaccelerator"
	fakeglobalaccelerator "github.com/topfreegames/global-accelerator-operator/pkg/aws/globalaccelerator/fake"

	"testing"
)

type globalAccelerator struct {
	GlobalAccelerator globalacceleratortypes.Accelerator
	Tags              []globalacceleratortypes.Tag
}

type testCase struct {
	description                string
	k8sObjects                 []client.Object
	globalAccelerators         []globalAccelerator
	listeners                  []globalacceleratortypes.Listener
	endpointGroups             []globalacceleratortypes.EndpointGroup
	loadBalancers              []elasticloadbalancingv2types.LoadBalancer
	loadBalancerListeners      []elasticloadbalancingv2types.Listener
	updatedAccelerator         globalAccelerator
	createAcceleratorError     error
	createListenerError        error
	createEndpointGroupError   error
	updateEndpointGroupError   error
	listAcceleratorError       error
	listListenersError         error
	describeAcceleratorError   error
	describeLoadBalancersError error
	describeTargetGroupsError  error
	deleteEndpointGroupError   error
	deleteAcceleratorError     error
	deleteListenerError        error
	listEndpointGroupsError    error
	updateAcceleratorError     error
	expectedError              bool
	expectedGlobalAccelerator  *string
	expectedListener           *string
	expectedPorts              []globalacceleratorawswildlifeiov1alpha1.EndpointGroupPorts
	expectedEndpointGroup      *string
}

func newFakeGlobalAcceleratorClient(tc testCase) *fakeglobalaccelerator.MockGlobalAcceleratorClient {
	return &fakeglobalaccelerator.MockGlobalAcceleratorClient{
		MockCreateAccelerator: func(ctx context.Context, input *globalacceleratorsdk.CreateAcceleratorInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.CreateAcceleratorOutput, error) {
			if tc.createAcceleratorError != nil {
				return nil, tc.createAcceleratorError
			}
			return &globalacceleratorsdk.CreateAcceleratorOutput{Accelerator: &globalacceleratortypes.Accelerator{AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy")}}, nil
		},
		MockCreateEndpointGroup: func(ctx context.Context, input *globalacceleratorsdk.CreateEndpointGroupInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.CreateEndpointGroupOutput, error) {
			if tc.createEndpointGroupError != nil {
				return nil, tc.createEndpointGroupError
			}
			return &globalacceleratorsdk.CreateEndpointGroupOutput{EndpointGroup: &globalacceleratortypes.EndpointGroup{EndpointGroupArn: aws.String(fmt.Sprintf("%s/endpoint-group/www", *input.ListenerArn))}}, nil
		},
		MockCreateListener: func(ctx context.Context, input *globalacceleratorsdk.CreateListenerInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.CreateListenerOutput, error) {
			if tc.createListenerError != nil {
				return nil, tc.createListenerError
			}
			return &globalacceleratorsdk.CreateListenerOutput{Listener: &globalacceleratortypes.Listener{ListenerArn: aws.String(fmt.Sprintf("%s/listener/zzz", *input.AcceleratorArn)), PortRanges: []globalacceleratortypes.PortRange{{FromPort: aws.Int32(49152), ToPort: aws.Int32(49152)}}}}, nil
		},
		MockDescribeAccelerator: func(ctx context.Context, input *globalacceleratorsdk.DescribeAcceleratorInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.DescribeAcceleratorOutput, error) {
			if tc.describeAcceleratorError != nil {
				return nil, tc.describeAcceleratorError
			}
			var globalAccelerator globalacceleratortypes.Accelerator
			for _, ga := range tc.globalAccelerators {
				if *ga.GlobalAccelerator.AcceleratorArn == *input.AcceleratorArn {
					globalAccelerator = ga.GlobalAccelerator
					break
				}
			}
			return &globalacceleratorsdk.DescribeAcceleratorOutput{Accelerator: &globalAccelerator}, nil
		},
		MockListAccelerators: func(ctx context.Context, input *globalacceleratorsdk.ListAcceleratorsInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.ListAcceleratorsOutput, error) {
			if tc.listAcceleratorError != nil {
				return nil, tc.listAcceleratorError
			}
			globalAccelerators := []globalacceleratortypes.Accelerator{}
			for _, ga := range tc.globalAccelerators {
				globalAccelerators = append(globalAccelerators, ga.GlobalAccelerator)
			}
			return &globalacceleratorsdk.ListAcceleratorsOutput{Accelerators: globalAccelerators}, nil
		},
		MockListListeners: func(ctx context.Context, input *globalacceleratorsdk.ListListenersInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.ListListenersOutput, error) {
			if tc.listListenersError != nil {
				return nil, tc.listListenersError
			}
			listeners := []globalacceleratortypes.Listener{}
			for _, listener := range tc.listeners {
				if strings.HasPrefix(*listener.ListenerArn, *input.AcceleratorArn) {
					listeners = append(listeners, listener)
				}
			}
			return &globalacceleratorsdk.ListListenersOutput{Listeners: listeners}, nil
		},
		MockListTagsForResource: func(ctx context.Context, input *globalacceleratorsdk.ListTagsForResourceInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.ListTagsForResourceOutput, error) {
			tags := []globalacceleratortypes.Tag{}
			for _, ga := range tc.globalAccelerators {
				if *input.ResourceArn == *ga.GlobalAccelerator.AcceleratorArn {
					tags = ga.Tags
					break
				}
			}
			return &globalacceleratorsdk.ListTagsForResourceOutput{Tags: tags}, nil
		},
		MockUntagResource: func(ctx context.Context, input *globalacceleratorsdk.UntagResourceInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.UntagResourceOutput, error) {
			return &globalacceleratorsdk.UntagResourceOutput{}, nil
		},
		MockUpdateEndpointGroup: func(ctx context.Context, input *globalacceleratorsdk.UpdateEndpointGroupInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.UpdateEndpointGroupOutput, error) {
			if tc.updateEndpointGroupError != nil {
				return nil, tc.updateEndpointGroupError
			}
			var endpointGroup globalacceleratortypes.EndpointGroup
			for _, eg := range tc.endpointGroups {
				if *input.EndpointGroupArn == *eg.EndpointGroupArn {
					endpointGroup = eg
					break
				}
			}
			endpointDescriptions := []globalacceleratortypes.EndpointDescription{}
			for _, endpointConfiguration := range input.EndpointConfigurations {
				endpointDescriptions = append(endpointDescriptions, globalacceleratortypes.EndpointDescription{EndpointId: endpointConfiguration.EndpointId})
			}
			endpointGroup.EndpointDescriptions = endpointDescriptions
			return &globalacceleratorsdk.UpdateEndpointGroupOutput{EndpointGroup: &endpointGroup}, nil
		},
		MockUpdateAccelerator: func(ctx context.Context, input *globalacceleratorsdk.UpdateAcceleratorInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.UpdateAcceleratorOutput, error) {
			if tc.updateAcceleratorError != nil {
				return nil, tc.updateAcceleratorError
			}
			return &globalacceleratorsdk.UpdateAcceleratorOutput{Accelerator: &tc.updatedAccelerator.GlobalAccelerator}, nil
		},
		MockDeleteEndpointGroup: func(ctx context.Context, input *globalacceleratorsdk.DeleteEndpointGroupInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.DeleteEndpointGroupOutput, error) {
			if tc.deleteEndpointGroupError != nil {
				return nil, tc.deleteEndpointGroupError
			}
			return &globalacceleratorsdk.DeleteEndpointGroupOutput{}, nil
		},
		MockDeleteListener: func(ctx context.Context, input *globalacceleratorsdk.DeleteListenerInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.DeleteListenerOutput, error) {
			if tc.deleteListenerError != nil {
				return nil, tc.deleteListenerError
			}
			return &globalacceleratorsdk.DeleteListenerOutput{}, nil
		},
		MockDeleteAccelerator: func(ctx context.Context, input *globalacceleratorsdk.DeleteAcceleratorInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.DeleteAcceleratorOutput, error) {
			if tc.deleteAcceleratorError != nil {
				return nil, tc.deleteAcceleratorError
			}
			return &globalacceleratorsdk.DeleteAcceleratorOutput{}, nil
		},
		MockListEndpointGroups: func(ctx context.Context, input *globalacceleratorsdk.ListEndpointGroupsInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.ListEndpointGroupsOutput, error) {
			if tc.listEndpointGroupsError != nil {
				return nil, tc.listEndpointGroupsError
			}
			endpointGroups := []globalacceleratortypes.EndpointGroup{}
			for _, eg := range tc.endpointGroups {
				endpointGroups = append(endpointGroups, eg)
			}
			return &globalacceleratorsdk.ListEndpointGroupsOutput{EndpointGroups: endpointGroups}, nil
		},
	}
}

func TestGetOrCreateCurrentGlobalAccelerator(t *testing.T) {
	RegisterFailHandler(g.Fail)
	g := NewWithT(t)

	testCases := []testCase{
		{
			description: "should return the GA with the current tag",
			globalAccelerators: []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/aaa"),
					},
					Tags: []globalacceleratortypes.Tag{
						{
							Key:   aws.String(globalaccelerator.ManagedAnnotation),
							Value: aws.String("global-accelerator-controller"),
						},
					},
				},
			},
			expectedGlobalAccelerator: aws.String("arn:aws:globalaccelerator::xxx:accelerator/aaa"),
		},
		{
			description:               "should create a new GA when current is unavailable",
			listAcceleratorError:      globalaccelerator.ErrUnavailableGlobalAccelerator,
			expectedError:             false,
			expectedGlobalAccelerator: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
		},
		{
			description:          "should return error when failing to get the current GA",
			listAcceleratorError: errors.New("failed to list"),
			expectedError:        true,
		},
		{
			description:            "should return error when failing to create a new GA",
			createAcceleratorError: errors.New("failed to create"),
			expectedError:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fakeGlobalAcceleratorClient := newFakeGlobalAcceleratorClient(tc)

			currentGlobalAccelerator, err := getOrCreateCurrentGlobalAccelerator(context.TODO(), fakeGlobalAcceleratorClient)
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(currentGlobalAccelerator.AcceleratorArn).To(BeEquivalentTo(tc.expectedGlobalAccelerator))
			}

		})
	}
}

func TestCreateListener(t *testing.T) {
	RegisterFailHandler(g.Fail)
	g := NewWithT(t)
	testCases := []testCase{
		{
			description:        "should return error when failing to retrieve listeners",
			listListenersError: errors.New("failed to list listeners"),
			expectedError:      true,
		},
		{
			description: "should return error when no port is available in listener",
			listeners: []globalacceleratortypes.Listener{
				{
					ListenerArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz"),
					PortRanges: []globalacceleratortypes.PortRange{
						{
							FromPort: aws.Int32(49152),
							ToPort:   aws.Int32(65535),
						},
					},
				},
			},
			expectedError: true,
		},
		{
			description:         "should return error when failing to create listener",
			createListenerError: errors.New("failed to create listeners"),
			expectedError:       true,
		},
		{
			description:      "should successfully create listener",
			expectedListener: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fakeGlobalAcceleratorClient := newFakeGlobalAcceleratorClient(tc)
			listener, err := createListener(context.TODO(), fakeGlobalAcceleratorClient, "arn:aws:globalaccelerator::xxx:accelerator/yyy", 1)
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(listener.ListenerArn).To(BeEquivalentTo(tc.expectedListener))
			}
		})
	}
}

func TestEndpointGroupReconciler(t *testing.T) {
	RegisterFailHandler(g.Fail)
	g := NewWithT(t)

	testCases := []testCase{
		{
			description: "should return error when failing to get current GA",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint-group",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
				},
			},
			listAcceleratorError: errors.New("failed to list accelerators"),
			expectedError:        true,
		},
		{
			description: "should return error when failing to get current GA using assigned ARN",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint-group",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
					Status: globalacceleratorawswildlifeiov1alpha1.EndpointGroupStatus{
						GlobalAcceleratorARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
					},
				},
			},
			describeAcceleratorError: errors.New("failed to describe accelerators"),
			expectedError:            true,
		},
		{
			description: "should return err when failing to describe lbs",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint-group",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
				},
			},
			describeLoadBalancersError: errors.New("failed to describe lbs"),
			expectedError:              true,
			expectedGlobalAccelerator:  aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
		},
		{
			description: "should return error when failing to get the endpoint group ports",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint-group",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
				},
			},
			loadBalancers: []elasticloadbalancingv2types.LoadBalancer{
				{
					DNSName:         aws.String("a.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
				},
			},
			describeTargetGroupsError: errors.New("failed to describe targetGroups"),
			expectedError:             true,
			expectedGlobalAccelerator: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
		},
		{
			description: "should return error when failing to create listener",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint-group",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
				},
			},
			loadBalancers: []elasticloadbalancingv2types.LoadBalancer{
				{
					DNSName:         aws.String("a.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
				},
			},
			loadBalancerListeners: []elasticloadbalancingv2types.Listener{
				{
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
					Port:            aws.Int32(80),
				},
			},
			createListenerError:       errors.New("failed to create listener"),
			expectedError:             true,
			expectedGlobalAccelerator: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
		},
		{
			description: "should return err when failing to create endpointGroup",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint-group",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
				},
			},
			loadBalancers: []elasticloadbalancingv2types.LoadBalancer{
				{
					DNSName:         aws.String("a.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
				},
			},
			loadBalancerListeners: []elasticloadbalancingv2types.Listener{
				{
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
					Port:            aws.Int32(80),
				},
			},
			createEndpointGroupError: errors.New("failed to create endpointGroup"),
			expectedError:            true,
			expectedPorts: []globalacceleratorawswildlifeiov1alpha1.EndpointGroupPorts{
				{
					ListenerPort: 49152,
					EndpointPort: 80,
				},
			},
			expectedGlobalAccelerator: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
			expectedListener:          aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz"),
		},
		{
			description: "should return err when failing to update endpointGroup",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint-group",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
					Status: globalacceleratorawswildlifeiov1alpha1.EndpointGroupStatus{
						EndpointGroupARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz/endpoint-group/www",
					},
				},
			},
			loadBalancers: []elasticloadbalancingv2types.LoadBalancer{
				{
					DNSName:         aws.String("a.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
				},
			},
			loadBalancerListeners: []elasticloadbalancingv2types.Listener{
				{
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
					Port:            aws.Int32(80),
				},
			},
			updateEndpointGroupError:  errors.New("failed to update endpointGroup"),
			expectedGlobalAccelerator: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
			expectedListener:          aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz"),
			expectedPorts: []globalacceleratorawswildlifeiov1alpha1.EndpointGroupPorts{
				{
					ListenerPort: 49152,
					EndpointPort: 80,
				},
			},
			expectedError: true,
		},
		{
			description: "should successfully create the endpointGroup",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint-group",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
				},
			},
			loadBalancers: []elasticloadbalancingv2types.LoadBalancer{
				{
					DNSName:         aws.String("a.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
				},
			},
			loadBalancerListeners: []elasticloadbalancingv2types.Listener{
				{
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
					Port:            aws.Int32(80),
				},
			},
			expectedPorts: []globalacceleratorawswildlifeiov1alpha1.EndpointGroupPorts{
				{
					ListenerPort: 49152,
					EndpointPort: 80,
				},
			},
			expectedGlobalAccelerator: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
			expectedListener:          aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz"),
			expectedEndpointGroup:     aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz/endpoint-group/www"),
		},
		{
			description: "should successfully update the endpointGroup",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint-group",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
					Status: globalacceleratorawswildlifeiov1alpha1.EndpointGroupStatus{
						EndpointGroupARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz/endpoint-group/www",
					},
				},
			},
			loadBalancers: []elasticloadbalancingv2types.LoadBalancer{
				{
					DNSName:         aws.String("a.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
				},
			},
			loadBalancerListeners: []elasticloadbalancingv2types.Listener{
				{
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/yyy"),
					Port:            aws.Int32(80),
				},
			},
			expectedPorts: []globalacceleratorawswildlifeiov1alpha1.EndpointGroupPorts{
				{
					ListenerPort: 49152,
					EndpointPort: 80,
				},
			},
			expectedGlobalAccelerator: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
			expectedListener:          aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz"),
			expectedEndpointGroup:     aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz/endpoint-group/www"),
		},
		{
			description: "should only delete endpointGroup if deletionTimestamp is not zero and there is another endpointGroup",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-endpoint-group",
						Namespace:         metav1.NamespaceDefault,
						DeletionTimestamp: &metav1.Time{Time: time.Now().UTC()},
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
					Status: globalacceleratorawswildlifeiov1alpha1.EndpointGroupStatus{
						EndpointGroupARN:     "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz/endpoint-group/www",
						GlobalAcceleratorARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
						ListenerARN:          "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz",
					},
				},
			},
			globalAccelerators: []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
						Enabled:        aws.Bool(true),
					},
					Tags: []globalacceleratortypes.Tag{
						{
							Key:   aws.String(globalaccelerator.ManagedAnnotation),
							Value: aws.String("global-accelerator-controller"),
						},
					},
				},
			},
			endpointGroups: []globalacceleratortypes.EndpointGroup{
				{
					EndpointGroupArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz/endpoint-group/xxx"),
				},
			},
			listeners: []globalacceleratortypes.Listener{
				{
					ListenerArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz"),
				},
			},
			expectedError: false,
		},
		{
			description: "should not delete accelerator if endpoitGroup deletionTimestamp is not zero and there is another listener",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-endpoint-group",
						Namespace:         metav1.NamespaceDefault,
						DeletionTimestamp: &metav1.Time{Time: time.Now().UTC()},
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
					Status: globalacceleratorawswildlifeiov1alpha1.EndpointGroupStatus{
						EndpointGroupARN:     "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz/endpoint-group/www",
						GlobalAcceleratorARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
						ListenerARN:          "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz",
					},
				},
			},
			globalAccelerators: []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
						Enabled:        aws.Bool(true),
					},
					Tags: []globalacceleratortypes.Tag{
						{
							Key:   aws.String(globalaccelerator.ManagedAnnotation),
							Value: aws.String("global-accelerator-controller"),
						},
					},
				},
			},
			endpointGroups: []globalacceleratortypes.EndpointGroup{},
			listeners: []globalacceleratortypes.Listener{
				{
					ListenerArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/aaa"),
				},
			},
			expectedError: false,
		},
		{
			description: "should delete accelerator if endpointGroup deletionTimestamp is not zero and there is no other listener or endpointGroup",
			k8sObjects: []client.Object{
				&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-endpoint-group",
						Namespace:         metav1.NamespaceDefault,
						DeletionTimestamp: &metav1.Time{Time: time.Now().UTC()},
					},
					Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
						DNSNames: []string{
							"a.elb.us-east-1.amazonaws.com",
						},
					},
					Status: globalacceleratorawswildlifeiov1alpha1.EndpointGroupStatus{
						EndpointGroupARN:     "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz/endpoint-group/www",
						GlobalAcceleratorARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
						ListenerARN:          "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz",
					},
				},
			},
			globalAccelerators: []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
						Enabled:        aws.Bool(true),
					},
					Tags: []globalacceleratortypes.Tag{
						{
							Key:   aws.String(globalaccelerator.ManagedAnnotation),
							Value: aws.String("global-accelerator-controller"),
						},
					},
				},
			},
			endpointGroups: []globalacceleratortypes.EndpointGroup{},
			listeners:      []globalacceleratortypes.Listener{},
			expectedError:  false,
		},
	}

	err := globalacceleratorawswildlifeiov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(tc.k8sObjects...).Build()

			fakeGlobalAcceleratorClient := newFakeGlobalAcceleratorClient(tc)

			fakeELBClient := &fakeelb.MockELBClient{
				MockDescribeLoadBalancers: func(ctx context.Context, input *elasticloadbalancingv2.DescribeLoadBalancersInput, opts []func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error) {
					if tc.describeLoadBalancersError != nil {
						return nil, tc.describeLoadBalancersError
					}
					return &elasticloadbalancingv2.DescribeLoadBalancersOutput{
						LoadBalancers: tc.loadBalancers,
					}, nil
				},
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

			reconciler := &EndpointGroupReconciler{
				Client: fakeClient,
				NewGlobalAcceleratorClientFactory: func(cfg aws.Config) globalaccelerator.GlobalAcceleratorClient {
					return fakeGlobalAcceleratorClient
				},
				NewELBClientFactory: func(cfg aws.Config) elb.ELBClient {
					return fakeELBClient
				},
			}
			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{
				NamespacedName: client.ObjectKey{
					Namespace: metav1.NamespaceDefault,
					Name:      "test-endpoint-group",
				},
			})
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
			}

			var endpointGroup globalacceleratorawswildlifeiov1alpha1.EndpointGroup
			err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: "test-endpoint-group", Namespace: metav1.NamespaceDefault}, &endpointGroup)
			if tc.expectedGlobalAccelerator != nil {
				g.Expect(endpointGroup.Status.GlobalAcceleratorARN).To(BeEquivalentTo(*tc.expectedGlobalAccelerator))
			}
			if tc.expectedListener != nil {
				g.Expect(endpointGroup.Status.ListenerARN).To(BeEquivalentTo(*tc.expectedListener))
			}
			if tc.expectedEndpointGroup != nil {
				g.Expect(endpointGroup.Status.EndpointGroupARN).To(BeEquivalentTo(*tc.expectedEndpointGroup))
			}
			if tc.expectedPorts != nil {
				g.Expect(endpointGroup.Status.Ports).To(BeEquivalentTo(tc.expectedPorts))
			}

		})
	}
}

func TestReconcileDelete(t *testing.T) {
	RegisterFailHandler(g.Fail)
	g := NewWithT(t)
	testCases := []testCase{
		{
			description:        "should return error when failing to retrieve listeners",
			listListenersError: errors.New("failed to list listeners"),
			expectedError:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fakeGlobalAcceleratorClient := newFakeGlobalAcceleratorClient(tc)
			listener, err := createListener(context.TODO(), fakeGlobalAcceleratorClient, "arn:aws:globalaccelerator::xxx:accelerator/yyy", 1)
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(listener.ListenerArn).To(BeEquivalentTo(tc.expectedListener))
			}
		})
	}
}
