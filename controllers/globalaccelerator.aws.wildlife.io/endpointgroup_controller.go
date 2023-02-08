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
	"github.com/pkg/errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	globalacceleratorawswildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/globalaccelerator.aws.wildlife.io/v1alpha1"

	awsclient "github.com/topfreegames/global-accelerator-operator/pkg/aws"

	globalacceleratortypes "github.com/aws/aws-sdk-go-v2/service/globalaccelerator/types"

	"github.com/topfreegames/global-accelerator-operator/pkg/aws/elb"
	"github.com/topfreegames/global-accelerator-operator/pkg/aws/globalaccelerator"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	requeue5min = ctrl.Result{RequeueAfter: 5 * time.Minute}
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

func createListener(ctx context.Context, globalAcceleratorClient globalaccelerator.GlobalAcceleratorClient, globalAcceleratorARN string) (*globalacceleratortypes.Listener, error) {
	listeners, err := globalaccelerator.GetListenersFromGlobalAccelerator(ctx, globalAcceleratorClient, globalAcceleratorARN)
	if err != nil {
		return nil, err
	}

	listenerPort := globalaccelerator.GetAvailableListenerPort(listeners)
	listener, err := globalaccelerator.CreateListener(ctx, globalAcceleratorClient, globalAcceleratorARN, listenerPort)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

//+kubebuilder:rbac:groups=globalaccelerator.aws.wildlife.io.infrastructure.wildlife.io,resources=endpointgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=globalaccelerator.aws.wildlife.io.infrastructure.wildlife.io,resources=endpointgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=globalaccelerator.aws.wildlife.io.infrastructure.wildlife.io,resources=endpointgroups/finalizers,verbs=update

func (r *EndpointGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	endpointGroup := &globalacceleratorawswildlifeiov1alpha1.EndpointGroup{}
	if err := r.Get(ctx, req.NamespacedName, endpointGroup); err != nil {
		return requeue5min, err
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
	elbClient := r.NewELBClientFactory(*cfg)

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

	var listenerARN string
	if endpointGroup.Status.ListenerARN == "" {
		listener, err := createListener(ctx, globalAcceleratorClient, *globalAccelerator.AcceleratorArn)
		if err != nil {
			return requeue5min, err
		}
		endpointGroup.Status.ListenerARN = *listener.ListenerArn
		err = r.Status().Update(ctx, endpointGroup)
		if err != nil {
			return requeue5min, err
		}
		listenerARN = *listener.ListenerArn
	} else {
		listenerARN = endpointGroup.Status.ListenerARN
	}

	endpointConfigurations, err := globalaccelerator.GetEndpointGroupConfigurations(ctx, elbClient, endpointGroup.Spec.DNSNames)
	if err != nil {
		return requeue5min, err
	}

	if endpointGroup.Status.EndpointGroupARN == "" {
		createdEndpointGroup, err := globalaccelerator.CreateEndpointGroup(ctx, globalAcceleratorClient, listenerARN, endpointConfigurations)
		if err != nil {
			return requeue5min, err
		}
		endpointGroup.Status.EndpointGroupARN = *createdEndpointGroup.EndpointGroupArn
		err = r.Status().Update(ctx, endpointGroup)
		if err != nil {
			return requeue5min, err
		}
	} else {
		_, err = globalaccelerator.UpdateEndpointGroup(ctx, globalAcceleratorClient, endpointGroup.Status.EndpointGroupARN, endpointConfigurations)
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
