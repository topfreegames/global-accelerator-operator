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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	globalacceleratorawswildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/globalaccelerator.aws.wildlife.io/v1alpha1"
	infrastructurewildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/infrastructure.wildlife.io/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	EnableAnnotation = "global-accelerator.alpha.wildlife.io/enable"
	GroupAnnotation  = "global-accelerator.alpha.wildlife.io/group"
)

// EndpointGroupReconciler reconciles a EndpointGroup object
type EndpointGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func RESTConfig(ctx context.Context, kubeClient client.Client, credentialsRef corev1.ObjectReference) (*rest.Config, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: credentialsRef.Namespace,
		Name:      credentialsRef.Name,
	}
	if err := kubeClient.Get(ctx, secretKey, secret); err != nil {
		return nil, err
	}
	kubeConfig, ok := secret.Data["value"]
	if !ok {
		return nil, errors.Errorf("missing key value in secret data")
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, errors.Errorf("failed to create REST configuration")
	}

	return config, nil
}

func isEnabled(service *corev1.Service) bool {
	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return false
	}

	if value, ok := service.Annotations[EnableAnnotation]; !ok || value != "true" {
		return false
	}

	if _, ok := service.Annotations[GroupAnnotation]; !ok {
		return false
	}

	return true
}

//+kubebuilder:rbac:groups=infrastructure.wildlife.io,resources=clustergroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.wildlife.io,resources=clustergroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.wildlife.io,resources=clustergroups/finalizers,verbs=update

func (r *EndpointGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	clusterGroup := &infrastructurewildlifeiov1alpha1.ClusterGroup{}
	if err := r.Get(ctx, req.NamespacedName, clusterGroup); err != nil {
		return ctrl.Result{}, err
	}

	groupedServices := make(map[string][]corev1.Service)

	for _, cluster := range clusterGroup.Spec.Clusters {
		config, err := RESTConfig(ctx, r.Client, cluster.CredentialsRef)
		if err != nil {
			return ctrl.Result{}, err
		}

		remoteClient, err := client.New(config, client.Options{})
		if err != nil {
			return ctrl.Result{}, errors.Errorf("failed to instanciate remote client for Cluster %s", cluster.Name)
		}

		serviceList := &corev1.ServiceList{}
		err = remoteClient.List(ctx, serviceList)
		if err != nil {
			return ctrl.Result{}, errors.Errorf("failed to list services in remote cluster %s", cluster.Name)
		}
		for _, service := range serviceList.Items {
			if isEnabled(&service) {
				groupedServices[service.Annotations[GroupAnnotation]] = append(groupedServices[service.Annotations[GroupAnnotation]], service)
			}
		}
	}
	// TODO: Create the EndpointGroup CR based on the desired state calculated

	for group, services := range groupedServices {
		endpointGroup := &globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      group,
				Namespace: "global-accelerator-operator-system",
			},
			Spec: globalacceleratorawswildlifeiov1alpha1.EndpointGroupSpec{
				Hostnames: []string{},
			},
		}

		for _, service := range services {
			endpointGroup.Spec.Hostnames = append(endpointGroup.Spec.Hostnames, service.Status.LoadBalancer.Ingress[0].Hostname)
		}

		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, endpointGroup, func() error {
			return nil
		})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EndpointGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurewildlifeiov1alpha1.ClusterGroup{}).
		Complete(r)
}
