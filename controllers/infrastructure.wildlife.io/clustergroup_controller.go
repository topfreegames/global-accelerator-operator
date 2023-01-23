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

package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	infrastructurewildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/infrastructure.wildlife.io/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/topfreegames/global-accelerator-operator/pkg/kubernetes"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	EnableAnnotation = "global-accelerator.alpha.wildlife.io/enable"
	GroupAnnotation  = "global-accelerator.alpha.wildlife.io/group"
)

var (
	requeue1min = ctrl.Result{RequeueAfter: 1 * time.Minute}
)

type ClusterGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func getDNSNameFromGroupedServices(services []corev1.Service) ([]string, error) {
	var dnsNames []string
	for _, service := range services {
		if len(service.Status.LoadBalancer.Ingress) == 0 {
			return nil, fmt.Errorf("service %s without Hostname defined", service.Name)
		}
		dnsNames = append(dnsNames, service.Status.LoadBalancer.Ingress[0].Hostname)
	}
	return dnsNames, nil
}

//+kubebuilder:rbac:groups=infrastructure.wildlife.io,resources=clustergroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.wildlife.io,resources=clustergroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.wildlife.io,resources=clustergroups/finalizers,verbs=update

func (r *ClusterGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	clusterGroup := &infrastructurewildlifeiov1alpha1.ClusterGroup{}
	if err := r.Get(ctx, req.NamespacedName, clusterGroup); err != nil {
		return requeue1min, err
	}

	groupedServices, err := kubernetes.GetGroupedServices(ctx, r.Client, clusterGroup)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to list services in %s", clusterGroup.Name))
		return requeue1min, err
	}
	for group, services := range groupedServices {
		endpointGroup := kubernetes.NewEndpointGroup(group, "global-accelerator-operator-system")
		dnsNames, err := getDNSNameFromGroupedServices(services)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to retrieve Hostname in %s", group))
			continue
		}

		err = kubernetes.UpdateEndpointGroupMembers(ctx, r.Client, endpointGroup, dnsNames)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to update EndpointGroup members in %s", group))
			continue
		}
		logger.Info(fmt.Sprintf("updated EndpointGroup members for %s", group))
	}

	return requeue1min, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurewildlifeiov1alpha1.ClusterGroup{}).
		Complete(r)
}
