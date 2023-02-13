package globalaccelerator

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	globalacceleratorsdk "github.com/aws/aws-sdk-go-v2/service/globalaccelerator"
	globalacceleratortypes "github.com/aws/aws-sdk-go-v2/service/globalaccelerator/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	globalacceleratorawswildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/globalaccelerator.aws.wildlife.io/v1alpha1"
	fakeglobalaccelerator "github.com/topfreegames/global-accelerator-operator/pkg/aws/globalaccelerator/fake"
	"k8s.io/apimachinery/pkg/util/rand"
	"strings"
	"testing"
)

type globalAccelerator struct {
	GlobalAccelerator globalacceleratortypes.Accelerator
	Tags              []globalacceleratortypes.Tag
}

func TestIsGlobalAcceleratorAvailable(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description        string
		expectedError      bool
		expectedResult     bool
		globalAccelerators []globalAccelerator
		listeners          []globalacceleratortypes.Listener
		listListenersError error
	}{
		{
			description:        "should return error when failing to list listeners",
			expectedError:      true,
			listListenersError: errors.New("failed to list listeners"),
		},
		{
			description: "should return false when exceeding the number of listeners",
			listeners: func() []globalacceleratortypes.Listener {
				listeners := []globalacceleratortypes.Listener{}
				for i := 1; i < 15; i++ {
					listeners = append(listeners, globalacceleratortypes.Listener{
						ListenerArn: aws.String(fmt.Sprintf("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/%s", rand.String(3))),
					})
				}
				return listeners
			}(),
		},
		{
			description: "should return true when there is room for more listeners in the GA",
			listeners: []globalacceleratortypes.Listener{
				{
					ListenerArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/www"),
				},
			},
			expectedResult: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			tc.globalAccelerators = []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
					},
				},
			}

			fakeGlobalAcceleratorClient := &fakeglobalaccelerator.MockGlobalAcceleratorClient{
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

					return &globalacceleratorsdk.ListListenersOutput{
						Listeners: listeners,
					}, nil
				},
			}
			ok, err := isGlobalAcceleratorAvailable(context.TODO(), fakeGlobalAcceleratorClient, aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"))
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(ok).To(BeEquivalentTo(tc.expectedResult))
			}
		})
	}
}

func TestGetCurrentGlobalAccelerator(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description          string
		expectedError        bool
		expectedAccelerator  string
		globalAccelerators   []globalAccelerator
		listeners            []globalacceleratortypes.Listener
		listAcceleratorError error
		listListenersError   error
		untagResourceError   error
	}{
		{
			description:          "should return error when failed to list accelerators",
			expectedError:        true,
			listAcceleratorError: errors.New("error listing accelerators"),
		},
		{
			description: "should return error when failed to check current GA availability",
			globalAccelerators: []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
					},
					Tags: []globalacceleratortypes.Tag{
						{
							Key:   aws.String(CurrentAnnotation),
							Value: aws.String("true"),
						},
					},
				},
			},
			expectedError:      true,
			listListenersError: errors.New("error listing listeners"),
		},
		{
			description: "should return error when no GA is being managed",
			globalAccelerators: []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
					},
				},
			},
			expectedError: true,
		},
		{
			description: "should return error when the managed GA is unavailable",
			globalAccelerators: []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
					},
					Tags: []globalacceleratortypes.Tag{
						{
							Key:   aws.String(CurrentAnnotation),
							Value: aws.String("true"),
						},
					},
				},
			},
			listeners: func() []globalacceleratortypes.Listener {
				listeners := []globalacceleratortypes.Listener{}
				for i := 1; i < 15; i++ {
					listeners = append(listeners, globalacceleratortypes.Listener{
						ListenerArn: aws.String(fmt.Sprintf("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/%s", rand.String(3))),
					})
				}
				return listeners
			}(),
			expectedError: true,
		},
		{
			description: "should return error when failed to untag a unavailable GA",
			globalAccelerators: []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
					},
					Tags: []globalacceleratortypes.Tag{
						{
							Key:   aws.String(CurrentAnnotation),
							Value: aws.String("true"),
						},
					},
				},
			},
			listeners: func() []globalacceleratortypes.Listener {
				listeners := []globalacceleratortypes.Listener{}
				for i := 1; i < 15; i++ {
					listeners = append(listeners, globalacceleratortypes.Listener{
						ListenerArn: aws.String(fmt.Sprintf("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/%s", rand.String(3))),
					})
				}
				return listeners
			}(),
			untagResourceError: errors.New("failed to untag GA"),
			expectedError:      true,
		},
		{
			description: "should return GA when it's available",
			globalAccelerators: []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
					},
					Tags: []globalacceleratortypes.Tag{
						{
							Key:   aws.String(CurrentAnnotation),
							Value: aws.String("true"),
						},
					},
				},
			},
			expectedAccelerator: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			fakeGlobalAcceleratorClient := &fakeglobalaccelerator.MockGlobalAcceleratorClient{
				MockListAccelerators: func(ctx context.Context, input *globalacceleratorsdk.ListAcceleratorsInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.ListAcceleratorsOutput, error) {
					if tc.listAcceleratorError != nil {
						return nil, tc.listAcceleratorError
					}
					globalAccelerators := []globalacceleratortypes.Accelerator{}
					for _, ga := range tc.globalAccelerators {
						globalAccelerators = append(globalAccelerators, ga.GlobalAccelerator)
					}

					return &globalacceleratorsdk.ListAcceleratorsOutput{
						Accelerators: globalAccelerators,
					}, nil
				},
				MockListTagsForResource: func(ctx context.Context, input *globalacceleratorsdk.ListTagsForResourceInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.ListTagsForResourceOutput, error) {
					tags := []globalacceleratortypes.Tag{}
					for _, ga := range tc.globalAccelerators {
						if *input.ResourceArn == *ga.GlobalAccelerator.AcceleratorArn {
							tags = ga.Tags
							break
						}
					}

					return &globalacceleratorsdk.ListTagsForResourceOutput{
						Tags: tags,
					}, nil
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

					return &globalacceleratorsdk.ListListenersOutput{
						Listeners: listeners,
					}, nil
				},
				MockUntagResource: func(ctx context.Context, input *globalacceleratorsdk.UntagResourceInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.UntagResourceOutput, error) {
					if tc.untagResourceError != nil {
						return nil, tc.untagResourceError
					}
					return &globalacceleratorsdk.UntagResourceOutput{}, nil
				},
			}
			accelerator, err := GetCurrentGlobalAccelerator(context.TODO(), fakeGlobalAcceleratorClient)
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(*accelerator.AcceleratorArn).To(BeEquivalentTo(tc.expectedAccelerator))
			}
		})
	}
}

func TestCreateGlobalAccelerator(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description            string
		expectedError          bool
		expectedAccelerator    string
		createAcceleratorError error
	}{
		{
			description:            "should return error when failed to create accelerator",
			createAcceleratorError: errors.New("failed to create accelerator"),
			expectedError:          true,
		},
		{
			description:         "should successfully create the GA",
			expectedAccelerator: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fakeGlobalAcceleratorClient := &fakeglobalaccelerator.MockGlobalAcceleratorClient{
				MockCreateAccelerator: func(ctx context.Context, input *globalacceleratorsdk.CreateAcceleratorInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.CreateAcceleratorOutput, error) {
					if tc.createAcceleratorError != nil {
						return nil, tc.createAcceleratorError
					}
					return &globalacceleratorsdk.CreateAcceleratorOutput{
						Accelerator: &globalacceleratortypes.Accelerator{
							AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
						},
					}, nil
				},
			}
			accelerator, err := CreateGlobalAccelerator(context.TODO(), fakeGlobalAcceleratorClient)
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(*accelerator.AcceleratorArn).To(BeEquivalentTo(tc.expectedAccelerator))
			}
		})
	}
}

func TestGetGlobalAcceleratorWithARN(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description              string
		globalAcceleratorARN     string
		globalAccelerators       []globalAccelerator
		expectedAccelerator      string
		expectedError            bool
		describeAcceleratorError error
	}{
		{
			description:              "should return error when failing to describe accelerator",
			describeAcceleratorError: errors.New("failed to describe accelerator"),
			expectedError:            true,
		},
		{
			description:          "should successfully return the global accelerator",
			globalAcceleratorARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
			globalAccelerators: []globalAccelerator{
				{
					GlobalAccelerator: globalacceleratortypes.Accelerator{
						AcceleratorArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy"),
					},
				},
			},
			expectedAccelerator: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fakeGlobalAcceleratorClient := &fakeglobalaccelerator.MockGlobalAcceleratorClient{
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
					return &globalacceleratorsdk.DescribeAcceleratorOutput{
						Accelerator: &globalAccelerator,
					}, nil
				},
			}
			accelerator, err := GetGlobalAcceleratorWithARN(context.TODO(), fakeGlobalAcceleratorClient, tc.globalAcceleratorARN)
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(*accelerator.AcceleratorArn).To(BeEquivalentTo(tc.expectedAccelerator))
			}
		})
	}
}

func TestGetListenersFromGlobalAccelerator(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description          string
		globalAcceleratorARN string
		expectedError        bool
		listeners            []globalacceleratortypes.Listener
		listListenersError   error
	}{
		{
			description:        "should return err when failing to list listeners",
			listListenersError: errors.New("failed to list listeners"),
			expectedError:      true,
		},
		{
			description:          "should successfully return the listener from a GA",
			globalAcceleratorARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
			listeners: []globalacceleratortypes.Listener{
				{
					ListenerArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/www"),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fakeGlobalAcceleratorClient := &fakeglobalaccelerator.MockGlobalAcceleratorClient{
				MockListListeners: func(ctx context.Context, input *globalacceleratorsdk.ListListenersInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.ListListenersOutput, error) {
					if tc.listListenersError != nil {
						return nil, tc.listListenersError
					}
					return &globalacceleratorsdk.ListListenersOutput{}, nil
				},
			}
			_, err := GetListenersFromGlobalAccelerator(context.TODO(), fakeGlobalAcceleratorClient, tc.globalAcceleratorARN)
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
			}
		})
	}
}

func TestCreateListener(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	type expectedListener struct {
		listenerARN string
		ports       []globalacceleratortypes.PortRange
	}

	testCases := []struct {
		description          string
		globalAcceleratorARN string
		listenerPort         []int32
		expectedListener     expectedListener
		expectedError        error
		createListenerError  error
	}{
		{
			description:          "should return error when failed to create listener",
			createListenerError:  errors.New("failed to create listener"),
			globalAcceleratorARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
			expectedError:        errors.New("failed to create listener"),
		},
		{
			description: "should successfully create listener",
			listenerPort: []int32{
				80,
				443,
			},
			globalAcceleratorARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy",
			expectedListener: expectedListener{
				listenerARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/zzz",
				ports: []globalacceleratortypes.PortRange{
					{
						FromPort: aws.Int32(80),
						ToPort:   aws.Int32(80),
					},
					{
						FromPort: aws.Int32(443),
						ToPort:   aws.Int32(443),
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fakeGlobalAcceleratorClient := &fakeglobalaccelerator.MockGlobalAcceleratorClient{
				MockCreateListener: func(ctx context.Context, input *globalacceleratorsdk.CreateListenerInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.CreateListenerOutput, error) {
					if tc.createListenerError != nil {
						return nil, tc.createListenerError
					}
					return &globalacceleratorsdk.CreateListenerOutput{
						Listener: &globalacceleratortypes.Listener{
							ListenerArn: aws.String(fmt.Sprintf("%s/listener/zzz", *input.AcceleratorArn)),
							PortRanges:  input.PortRanges,
						},
					}, nil
				},
			}
			listener, err := CreateListener(context.TODO(), fakeGlobalAcceleratorClient, tc.globalAcceleratorARN, tc.listenerPort)
			if tc.expectedError != nil {
				g.Expect(err.Error()).To(BeEquivalentTo(tc.expectedError.Error()))
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(*listener.ListenerArn).To(BeEquivalentTo(tc.expectedListener.listenerARN))
				g.Expect(listener.PortRanges).To(BeEquivalentTo(tc.expectedListener.ports))
			}
		})
	}
}

func TestCheckIfListenerPortIsAvailable(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description    string
		listenerPort   int
		expectedOutput bool
		listeners      []globalacceleratortypes.Listener
	}{
		{
			description:  "should return false when port is already assigned",
			listenerPort: 49152,
			listeners: []globalacceleratortypes.Listener{
				{
					PortRanges: []globalacceleratortypes.PortRange{
						{
							FromPort: aws.Int32(49152),
							ToPort:   aws.Int32(49152),
						},
					},
				},
			},
		},
		{
			description:    "should return true when port is available",
			listenerPort:   49153,
			expectedOutput: true,
			listeners: []globalacceleratortypes.Listener{
				{
					PortRanges: []globalacceleratortypes.PortRange{
						{
							FromPort: aws.Int32(49152),
							ToPort:   aws.Int32(49152),
						},
					},
				},
				{
					PortRanges: []globalacceleratortypes.PortRange{
						{
							FromPort: aws.Int32(49154),
							ToPort:   aws.Int32(65535),
						},
					},
				},
			},
		},
		{
			description:    "should return true when no listener is created",
			listenerPort:   49152,
			expectedOutput: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			isPortAvailable := checkIfListenerPortIsAvailable(tc.listeners, tc.listenerPort)
			g.Expect(isPortAvailable).To(BeEquivalentTo(tc.expectedOutput))
		})
	}
}

func TestGetAvailableListenerPort(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description    string
		listenerPorts  []int32
		expectedOutput int
		expectedError  error
		listeners      []globalacceleratortypes.Listener
	}{
		{
			description:    "should return the first high port",
			expectedOutput: 49152,
		},
		{
			description: "should return the first available high port",
			listenerPorts: []int32{
				49152,
				49153,
			},
			expectedOutput: 49154,
		},
		{
			description: "should return error when no port is available",
			listenerPorts: []int32{
				49152,
				49153,
			},
			expectedError: errors.New("failed to get available listener port"),
			listeners: []globalacceleratortypes.Listener{
				{
					PortRanges: []globalacceleratortypes.PortRange{
						{
							FromPort: aws.Int32(49154),
							ToPort:   aws.Int32(65534),
						},
					},
				},
			},
		},
		{
			description:   "should return the next available port after the listeners",
			listenerPorts: []int32{},
			listeners: []globalacceleratortypes.Listener{
				{
					PortRanges: []globalacceleratortypes.PortRange{
						{
							FromPort: aws.Int32(49152),
							ToPort:   aws.Int32(49152),
						},
						{
							FromPort: aws.Int32(49153),
							ToPort:   aws.Int32(49153),
						},
					},
				},
			},
			expectedOutput: 49154,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			port, err := GetAvailableListenerPort(tc.listeners, tc.listenerPorts)
			if tc.expectedError != nil {
				g.Expect(err.Error()).To(BeEquivalentTo(tc.expectedError.Error()))
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(port).To(BeEquivalentTo(&tc.expectedOutput))
			}
		})
	}
}

func TestGetPortOverrides(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description    string
		endpointGroup  *globalacceleratorawswildlifeiov1alpha1.EndpointGroup
		expectedOutput []globalacceleratortypes.PortOverride
	}{
		{
			description: "should return the equivalent PortOverride",
			endpointGroup: &globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
				Status: globalacceleratorawswildlifeiov1alpha1.EndpointGroupStatus{
					Ports: []globalacceleratorawswildlifeiov1alpha1.EndpointGroupPorts{
						{
							ListenerPort: 49152,
							EndpointPort: 80,
						},
					},
				},
			},
			expectedOutput: []globalacceleratortypes.PortOverride{
				{
					ListenerPort: aws.Int32(49152),
					EndpointPort: aws.Int32(80),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			portOverride := GetPortOverrides(tc.endpointGroup)
			g.Expect(portOverride).To(BeEquivalentTo(tc.expectedOutput))
		})
	}
}

func TestCreateEndpointGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description              string
		expectedError            bool
		listenerARN              string
		expectedEndpointGroup    string
		createEndpointGroupError error
	}{
		{
			description:              "should return error when failed to created ep",
			expectedError:            true,
			createEndpointGroupError: errors.New("failed to create ep"),
		},
		{
			description:           "should successfully create ep",
			listenerARN:           "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener",
			expectedEndpointGroup: "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/endpoint-group/www",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			fakeGlobalAcceleratorClient := &fakeglobalaccelerator.MockGlobalAcceleratorClient{
				MockCreateEndpointGroup: func(ctx context.Context, input *globalacceleratorsdk.CreateEndpointGroupInput, opts []func(*globalacceleratorsdk.Options)) (*globalacceleratorsdk.CreateEndpointGroupOutput, error) {
					if tc.createEndpointGroupError != nil {
						return nil, tc.createEndpointGroupError
					}
					return &globalacceleratorsdk.CreateEndpointGroupOutput{
						EndpointGroup: &globalacceleratortypes.EndpointGroup{
							EndpointGroupArn: aws.String(fmt.Sprintf("%s/endpoint-group/www", *input.ListenerArn)),
						},
					}, nil
				},
			}
			endpointGroup, err := CreateEndpointGroup(context.TODO(), fakeGlobalAcceleratorClient, tc.listenerARN, []globalacceleratortypes.PortOverride{}, []globalacceleratortypes.EndpointConfiguration{})
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(*endpointGroup.EndpointGroupArn).To(BeEquivalentTo(tc.expectedEndpointGroup))
			}
		})
	}
}

func TestUpdateEndpointGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description                  string
		expectedError                bool
		endpointGroupARN             string
		endpointConfigurations       []globalacceleratortypes.EndpointConfiguration
		endpointGroups               []globalacceleratortypes.EndpointGroup
		expectedEndpointDescriptions []globalacceleratortypes.EndpointDescription
		updateEndpointGroupError     error
	}{
		{
			description:              "should return error when failed to update ep",
			updateEndpointGroupError: errors.New("failed to update ep"),
			expectedError:            true,
		},
		{
			description:      "should successfully update ep",
			endpointGroupARN: "arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/endpoint-group/www",
			endpointConfigurations: []globalacceleratortypes.EndpointConfiguration{
				{
					EndpointId: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/ccc"),
				},
			},
			endpointGroups: []globalacceleratortypes.EndpointGroup{
				{
					EndpointGroupArn: aws.String("arn:aws:globalaccelerator::xxx:accelerator/yyy/listener/endpoint-group/www"),
					EndpointDescriptions: []globalacceleratortypes.EndpointDescription{
						{
							EndpointId: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/aaa"),
						},
						{
							EndpointId: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/bbb"),
						},
					},
				},
			},
			expectedEndpointDescriptions: []globalacceleratortypes.EndpointDescription{
				{
					EndpointId: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/ccc"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			fakeGlobalAcceleratorClient := &fakeglobalaccelerator.MockGlobalAcceleratorClient{
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
						endpointDescriptions = append(endpointDescriptions, globalacceleratortypes.EndpointDescription{
							EndpointId: endpointConfiguration.EndpointId,
						})
					}

					endpointGroup.EndpointDescriptions = endpointDescriptions
					return &globalacceleratorsdk.UpdateEndpointGroupOutput{
						EndpointGroup: &endpointGroup,
					}, nil
				},
			}
			endpointGroup, err := UpdateEndpointGroup(context.TODO(), fakeGlobalAcceleratorClient, tc.endpointGroupARN, []globalacceleratortypes.PortOverride{}, tc.endpointConfigurations)
			if tc.expectedError {
				g.Expect(err).Should(HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(endpointGroup.EndpointDescriptions).To(BeEquivalentTo(tc.expectedEndpointDescriptions))
			}
		})
	}
}

func TestGetEndpointGroupConfigurations(t *testing.T) {
	RegisterFailHandler(Fail)
	g := NewWithT(t)

	testCases := []struct {
		description                        string
		loadBalancers                      []types.LoadBalancer
		expectedEndpointGroupConfiguration []globalacceleratortypes.EndpointConfiguration
	}{
		{
			description: "should successfully return the endpointConfiguration list",
			loadBalancers: []types.LoadBalancer{
				{
					DNSName:         aws.String("a.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/aaa"),
				},
				{
					DNSName:         aws.String("b.elb.us-east-1.amazonaws.com"),
					LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/bbb"),
				},
			},
			expectedEndpointGroupConfiguration: []globalacceleratortypes.EndpointConfiguration{
				{
					EndpointId: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/aaa"),
				},
				{
					EndpointId: aws.String("arn:aws:elasticloadbalancing:us-east-1:xxx:loadbalancer/bbb"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			endpointConfigurations := GetEndpointGroupConfigurations(tc.loadBalancers)
			g.Expect(endpointConfigurations).To(BeEquivalentTo(tc.expectedEndpointGroupConfiguration))
		})
	}
}
