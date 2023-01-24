package globalaccelerator

import (
	"context"
	"fmt"
	globalacceleratortypes "github.com/aws/aws-sdk-go-v2/service/globalaccelerator/types"
	"github.com/pkg/errors"
	"github.com/topfreegames/global-accelerator-operator/pkg/aws/elb"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/globalaccelerator"
	globalacceleratorsdk "github.com/aws/aws-sdk-go-v2/service/globalaccelerator"
)

const (
	CurrentAnnotation = "global-accelerator.alpha.wildlife.io/current"
)

var (
	ErrUnavailableGlobalAccelerator = errors.New("failed to get current GlobalAccelerator")
)

type GlobalAcceleratorClient interface {
	CreateAccelerator(ctx context.Context, input *globalaccelerator.CreateAcceleratorInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.CreateAcceleratorOutput, error)
	CreateEndpointGroup(ctx context.Context, input *globalaccelerator.CreateEndpointGroupInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.CreateEndpointGroupOutput, error)
	CreateListener(ctx context.Context, input *globalaccelerator.CreateListenerInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.CreateListenerOutput, error)
	DescribeAccelerator(ctx context.Context, input *globalaccelerator.DescribeAcceleratorInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.DescribeAcceleratorOutput, error)
	ListAccelerators(ctx context.Context, input *globalaccelerator.ListAcceleratorsInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.ListAcceleratorsOutput, error)
	ListListeners(ctx context.Context, input *globalaccelerator.ListListenersInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.ListListenersOutput, error)
	ListTagsForResource(ctx context.Context, input *globalaccelerator.ListTagsForResourceInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.ListTagsForResourceOutput, error)
	UntagResource(ctx context.Context, input *globalaccelerator.UntagResourceInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.UntagResourceOutput, error)
	UpdateEndpointGroup(ctx context.Context, input *globalaccelerator.UpdateEndpointGroupInput, opts ...func(*globalaccelerator.Options)) (*globalaccelerator.UpdateEndpointGroupOutput, error)
}

func NewGlobalAcceleratorClient(cfg aws.Config) GlobalAcceleratorClient {
	return globalaccelerator.NewFromConfig(cfg)
}

func isGlobalAcceleratorAvailable(ctx context.Context, globalAcceleratorClient GlobalAcceleratorClient, globalAcceleratorARN *string) (bool, error) {
	listenersOutput, err := globalAcceleratorClient.ListListeners(ctx, &globalacceleratorsdk.ListListenersInput{
		AcceleratorArn: globalAcceleratorARN,
	})
	if err != nil {
		return false, err
	}
	// Default quota for listeners
	if len(listenersOutput.Listeners) < 10 {
		return true, nil
	}
	return false, nil
}

func GetCurrentGlobalAccelerator(ctx context.Context, globalAcceleratorClient GlobalAcceleratorClient) (*globalacceleratortypes.Accelerator, error) {
	// TODO: Paginate this properly
	globalAcceleratorsOutput, err := globalAcceleratorClient.ListAccelerators(ctx, &globalacceleratorsdk.ListAcceleratorsInput{
		MaxResults: aws.Int32(100),
	})
	if err != nil {
		return nil, err
	}

	for _, globalAccelerator := range globalAcceleratorsOutput.Accelerators {
		tagsOutput, err := globalAcceleratorClient.ListTagsForResource(ctx, &globalacceleratorsdk.ListTagsForResourceInput{
			ResourceArn: globalAccelerator.AcceleratorArn,
		})
		if err != nil {
			// TODO: Log that it failed to retrieve tags from a GA
			continue
		}
		for _, tag := range tagsOutput.Tags {
			if *tag.Key == CurrentAnnotation {
				ok, err := isGlobalAcceleratorAvailable(ctx, globalAcceleratorClient, globalAccelerator.AcceleratorArn)
				if err != nil {
					return nil, err
				}
				if ok {
					return &globalAccelerator, nil
				} else {
					_, err := globalAcceleratorClient.UntagResource(ctx, &globalacceleratorsdk.UntagResourceInput{
						ResourceArn: globalAccelerator.AcceleratorArn,
						TagKeys: []string{
							CurrentAnnotation,
						},
					})
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return nil, ErrUnavailableGlobalAccelerator
}

func CreateGlobalAccelerator(ctx context.Context, globalAcceleratorClient GlobalAcceleratorClient) (*globalacceleratortypes.Accelerator, error) {
	createGlobalAcceleratorOutput, err := globalAcceleratorClient.CreateAccelerator(ctx, &globalacceleratorsdk.CreateAcceleratorInput{
		Name: aws.String(fmt.Sprintf("global-accelerator-%s", rand.String(8))),
		Tags: []globalacceleratortypes.Tag{
			{
				Key:   aws.String(CurrentAnnotation),
				Value: aws.String("true"),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return createGlobalAcceleratorOutput.Accelerator, nil
}

func GetGlobalAcceleratorWithARN(ctx context.Context, globalAcceleratorClient GlobalAcceleratorClient, globalAcceleratorARN string) (*globalacceleratortypes.Accelerator, error) {
	describeGlobalAcceleratorOutput, err := globalAcceleratorClient.DescribeAccelerator(ctx, &globalacceleratorsdk.DescribeAcceleratorInput{
		AcceleratorArn: aws.String(globalAcceleratorARN),
	})
	if err != nil {
		return nil, err
	}
	return describeGlobalAcceleratorOutput.Accelerator, nil
}

func GetListenersFromGlobalAccelerator(ctx context.Context, globalAcceleratorClient GlobalAcceleratorClient, globalAcceleratorARN string) ([]globalacceleratortypes.Listener, error) {
	listenersOutput, err := globalAcceleratorClient.ListListeners(ctx, &globalacceleratorsdk.ListListenersInput{
		AcceleratorArn: aws.String(globalAcceleratorARN),
	})
	if err != nil {
		return nil, err
	}
	return listenersOutput.Listeners, nil
}

func CreateListener(ctx context.Context, globalAcceleratorClient GlobalAcceleratorClient, globalAcceleratorARN string, listenerPort int) (*globalacceleratortypes.Listener, error) {
	createdListener, err := globalAcceleratorClient.CreateListener(ctx, &globalacceleratorsdk.CreateListenerInput{
		AcceleratorArn: aws.String(globalAcceleratorARN),
		Protocol:       globalacceleratortypes.ProtocolTcp,
		PortRanges: []globalacceleratortypes.PortRange{
			{
				FromPort: aws.Int32(int32(listenerPort)),
				ToPort:   aws.Int32(int32(listenerPort)),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return createdListener.Listener, nil
}

func checkIfListenerPortIsAvailable(listeners []globalacceleratortypes.Listener, listenerPort int) bool {
	for _, listener := range listeners {
		for _, portRange := range listener.PortRanges {
			if listenerPort >= int(*portRange.FromPort) && listenerPort <= int(*portRange.ToPort) {
				{
					return false
				}
			}
		}
	}
	return true
}

// This shouldn't become a infinite loop since there is a limit for the number
// of listeners per GA
func GetAvailableListenerPort(listeners []globalacceleratortypes.Listener) int {
	var listenerPort int

	var availablePort bool
	for ok := true; ok; ok = !availablePort {
		listenerPort = rand.IntnRange(49152, 65535)
		availablePort = checkIfListenerPortIsAvailable(listeners, listenerPort)
	}

	return listenerPort
}

// TODO: An EndpointGroup can only belong to one region, we should infer the region or enable a way to define
func CreateEndpointGroup(ctx context.Context, globalAcceleratorClient GlobalAcceleratorClient, listenerARN string, endpointConfigurations []globalacceleratortypes.EndpointConfiguration) (*globalacceleratortypes.EndpointGroup, error) {
	createEndpointGroupOutput, err := globalAcceleratorClient.CreateEndpointGroup(ctx, &globalacceleratorsdk.CreateEndpointGroupInput{
		ListenerArn:            &listenerARN,
		EndpointGroupRegion:    aws.String("us-east-1"),
		EndpointConfigurations: endpointConfigurations,
	})
	if err != nil {
		return nil, err
	}
	return createEndpointGroupOutput.EndpointGroup, nil
}

func UpdateEndpointGroup(ctx context.Context, globalAcceleratorClient GlobalAcceleratorClient, endpointGroupARN string, endpointConfigurations []globalacceleratortypes.EndpointConfiguration) (*globalacceleratortypes.EndpointGroup, error) {
	updateEndpointGroupOutput, err := globalAcceleratorClient.UpdateEndpointGroup(ctx, &globalacceleratorsdk.UpdateEndpointGroupInput{
		EndpointGroupArn:       aws.String(endpointGroupARN),
		EndpointConfigurations: endpointConfigurations,
	})
	if err != nil {
		return nil, err
	}
	return updateEndpointGroupOutput.EndpointGroup, nil

}

func GetEndpointGroupConfigurations(ctx context.Context, elbClient elb.ELBClient, dnsNames []string) ([]globalacceleratortypes.EndpointConfiguration, error) {
	endpoints := []globalacceleratortypes.EndpointConfiguration{}
	for _, dnsName := range dnsNames {
		loadBalancerARN, err := elb.GetLoadBalancerARNFromDNS(ctx, elbClient, dnsName)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, globalacceleratortypes.EndpointConfiguration{
			EndpointId: aws.String(loadBalancerARN),
		})
	}
	return endpoints, nil
}
