/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package globalacceleratorawswildlifeio

import (
	"context"
	"time"

	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	globalacceleratorawswildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/globalaccelerator.aws.wildlife.io/v1alpha1"

	awsclient "github.com/topfreegames/global-accelerator-operator/pkg/aws"

	globalacceleratortypes "github.com/aws/aws-sdk-go-v2/service/globalaccelerator/types"

	"github.com/topfreegames/global-accelerator-operator/pkg/aws/elb"
	"github.com/topfreegames/global-accelerator-operator/pkg/aws/globalaccelerator"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	endpointGroupFinalizer = "endpointgroup.globalaccelerator.aws.wildlife.io/finalizer"
	requeue5min            = ctrl.Result{RequeueAfter: 5 * time.Minute}
	requeue1min            = ctrl.Result{RequeueAfter: 1 * time.Minute}
)

type EndpointGroupReconciler struct {
	client.Client
	Scheme                            *runtime.Scheme
	NewGlobalAcceleratorClientFactory func(cfg aws.Config) globalaccelerator.GlobalAcceleratorClient
	NewELBClientFactory               func(cfg aws.Config) elb.ELBClient
}

func getOrCreateCurrentGlobalAccelerator(ctx context.Context, globalAcceleratorClient globalaccelerator.GlobalAcceleratorClient) (*globalacceleratortypes.Accelerator, error) {
	currentGlobalAccelerator, err := globalaccelerator.GetCurrentGlobalAccelerator(ctx, globalAcceleratorClient)
	if err != nil {
		if errors.Is(err, globalaccelerator.ErrUnavailableGlobalAccelerator) {
			createdGlobalAccelerator, err := globalaccelerator.CreateGlobalAccelerator(ctx, globalAcceleratorClient)
			if err != nil {
				return nil, err
			}
			currentGlobalAccelerator = createdGlobalAccelerator
		} else {
			return nil, err
		}
	}
	return currentGlobalAccelerator, nil
}

func createListener(ctx context.Context, globalAcceleratorClient globalaccelerator.GlobalAcceleratorClient, globalAcceleratorARN string, numberOfPorts int) (*globalacceleratortypes.Listener, error) {
	listeners, err := globalaccelerator.GetListenersFromGlobalAccelerator(ctx, globalAcceleratorClient, globalAcceleratorARN)
	if err != nil {
		return nil, err
	}

	listenerPorts := []int32{}
	for i := 0; i < numberOfPorts; i++ {
		listenerPort, err := globalaccelerator.GetAvailableListenerPort(listeners, listenerPorts)
		if err != nil {
			return nil, err
		}
		listenerPorts = append(listenerPorts, int32(*listenerPort))
	}
	listener, err := globalaccelerator.CreateListener(ctx, globalAcceleratorClient, globalAcceleratorARN, listenerPorts)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (r *EndpointGroupReconciler) reconcileDelete(ctx context.Context, endpointGroup *globalacceleratorawswildlifeiov1alpha1.EndpointGroup) (ctrl.Result, error) {

	cfg, err := awsclient.GetConfig(ctx, "us-west-2") // Global Accelerator API is available in this region
	if err != nil {
		return requeue5min, err
	}

	globalAcceleratorClient := r.NewGlobalAcceleratorClientFactory(*cfg)

	endpointGroupARN := endpointGroup.Status.EndpointGroupARN
	listenerARN := endpointGroup.Status.ListenerARN
	acceleratorARN := endpointGroup.Status.GlobalAcceleratorARN

	if endpointGroupARN != "" {
		err = globalaccelerator.DeleteEndpointGroup(ctx, globalAcceleratorClient, endpointGroupARN)
		var endpointGroupNotFound *globalacceleratortypes.EndpointGroupNotFoundException
		if err != nil && !errors.As(err, &endpointGroupNotFound) {
			return requeue5min, err
		}
	}

	shouldDeleteListener, err := shouldDeleteListener(ctx, globalAcceleratorClient, listenerARN)
	if err != nil {
		return requeue5min, err
	}
	if shouldDeleteListener {
		err = globalaccelerator.DeleteListener(ctx, globalAcceleratorClient, listenerARN)
		if err != nil {
			return requeue5min, err
		}
	}

	shouldDeleteAccelerator, err := shouldDeleteAccelerator(ctx, globalAcceleratorClient, acceleratorARN)
	if err != nil {
		return requeue5min, err
	}
	if shouldDeleteAccelerator {
		err = globalaccelerator.DeleteAccelerator(ctx, globalAcceleratorClient, acceleratorARN)

		// requeue if accelerator is not ready
		if errors.Is(err, globalaccelerator.ErrGlobalAcceleratorNotReady) {
			return requeue1min, nil
		}
		if err != nil {
			return requeue5min, err
		}
	}

	controllerutil.RemoveFinalizer(endpointGroup, endpointGroupFinalizer)
	if err := r.Update(ctx, endpointGroup); err != nil {
		return requeue5min, err
	}

	return requeue5min, nil
}

func shouldDeleteListener(ctx context.Context, globalAcceleratorClient globalaccelerator.GlobalAcceleratorClient, listenerARN string) (bool, error) {
	endpointGroups, err := globalaccelerator.GetEndpointGroupsFromListener(ctx, globalAcceleratorClient, listenerARN)
	var listenerNotFound *globalacceleratortypes.ListenerNotFoundException
	if errors.As(err, &listenerNotFound) || len(endpointGroups) > 0 {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func shouldDeleteAccelerator(ctx context.Context, globalAcceleratorClient globalaccelerator.GlobalAcceleratorClient, acceleratorARN string) (bool, error) {
	listeners, err := globalaccelerator.GetListenersFromGlobalAccelerator(ctx, globalAcceleratorClient, acceleratorARN)
	var acceleratorNotFound *globalacceleratortypes.AcceleratorNotFoundException
	if errors.As(err, &acceleratorNotFound) || len(listeners) > 0 {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

//+kubebuilder:rbac:groups=globalaccelerator.aws.wildlife.io.infrastructure.wildlife.io,resources=endpointgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=globalaccelerator.aws.wildlife.io.infrastructure.wildlife.io,resources=endpointgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=globalaccelerator.aws.wildlife.io.infrastructure.wildlife.io,resources=endpointgroups/finalizers,verbs=update

func (r *EndpointGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	endpointGroup := &globalacceleratorawswildlifeiov1alpha1.EndpointGroup{}
	if err := r.Get(ctx, req.NamespacedName, endpointGroup); err != nil {
		return requeue5min, client.IgnoreNotFound(err)
	}

	if !endpointGroup.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, endpointGroup)
	}

	if !controllerutil.ContainsFinalizer(endpointGroup, endpointGroupFinalizer) {
		controllerutil.AddFinalizer(endpointGroup, endpointGroupFinalizer)
		if err := r.Update(ctx, endpointGroup); err != nil {
			return requeue5min, err
		}
	}

	cfg, err := awsclient.GetConfig(ctx, "us-west-2") // Global Accelerator API is available in this region
	if err != nil {
		return requeue5min, err
	}

	globalAcceleratorClient := r.NewGlobalAcceleratorClientFactory(*cfg)

	// TODO: An EndpointGroup can only belong to one region, we should infer the region or enable a way to define
	cfg, err = awsclient.GetConfig(ctx, "us-east-1")
	if err != nil {
		return requeue5min, err
	}

	var globalAccelerator *globalacceleratortypes.Accelerator

	if endpointGroup.Status.GlobalAcceleratorARN == "" {
		globalAccelerator, err = getOrCreateCurrentGlobalAccelerator(ctx, globalAcceleratorClient)
		if err != nil {
			return requeue5min, err
		}
		endpointGroup.Status.GlobalAcceleratorARN = *globalAccelerator.AcceleratorArn
		err = r.Status().Update(ctx, endpointGroup)
		if err != nil {
			return requeue5min, err
		}

	} else {
		globalAccelerator, err = globalaccelerator.GetGlobalAcceleratorWithARN(ctx, globalAcceleratorClient, endpointGroup.Status.GlobalAcceleratorARN)
		if err != nil {
			return requeue5min, err
		}
	}

	elbClient := r.NewELBClientFactory(*cfg)

	loadBalancers, err := elb.GetLoadBalancersFromDNS(ctx, elbClient, endpointGroup.Spec.DNSNames)
	if err != nil {
		return requeue5min, err
	}

	//TODO: Validate that all loadBalances uses the same ports
	loadBalancerPorts, err := elb.GetLoadBalancerPorts(ctx, elbClient, loadBalancers[0].LoadBalancerArn)
	if err != nil {
		return requeue5min, err
	}

	var listenerARN string
	if endpointGroup.Status.ListenerARN == "" {
		listener, err := createListener(ctx, globalAcceleratorClient, *globalAccelerator.AcceleratorArn, len(loadBalancerPorts))
		if err != nil {
			return requeue5min, err
		}
		endpointGroup.Status.ListenerARN = *listener.ListenerArn

		for i, _ := range listener.PortRanges {
			endpointGroup.Status.Ports = append(endpointGroup.Status.Ports,
				globalacceleratorawswildlifeiov1alpha1.EndpointGroupPorts{
					ListenerPort: *listener.PortRanges[i].ToPort,
					EndpointPort: loadBalancerPorts[i],
				})
		}
		err = r.Status().Update(ctx, endpointGroup)
		if err != nil {
			return requeue5min, err
		}
		listenerARN = *listener.ListenerArn
	} else {
		// TODO: We should update the resource to ensure its state
		listenerARN = endpointGroup.Status.ListenerARN
	}

	endpointGroupConfigurations := globalaccelerator.GetEndpointGroupConfigurations(loadBalancers)

	portOverrides := globalaccelerator.GetPortOverrides(endpointGroup)

	if endpointGroup.Status.EndpointGroupARN == "" {
		createdEndpointGroup, err := globalaccelerator.CreateEndpointGroup(ctx, globalAcceleratorClient, listenerARN, portOverrides, endpointGroupConfigurations)
		if err != nil {
			return requeue5min, err
		}
		endpointGroup.Status.EndpointGroupARN = *createdEndpointGroup.EndpointGroupArn
		err = r.Status().Update(ctx, endpointGroup)
		if err != nil {
			return requeue5min, err
		}
	} else {
		_, err = globalaccelerator.UpdateEndpointGroup(ctx, globalAcceleratorClient, endpointGroup.Status.EndpointGroupARN, portOverrides, endpointGroupConfigurations)
		if err != nil {
			return requeue5min, err
		}
	}

	return requeue5min, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EndpointGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&globalacceleratorawswildlifeiov1alpha1.EndpointGroup{}).
		Complete(r)
}
