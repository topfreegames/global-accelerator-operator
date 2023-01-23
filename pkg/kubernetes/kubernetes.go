package kubernetes

import (
	"context"

	"github.com/pkg/errors"
	globalacceleratorawswildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/globalaccelerator.aws.wildlife.io/v1alpha1"
	infrastructurewildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/infrastructure.wildlife.io/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	EnableAnnotation = "global-accelerator.alpha.wildlife.io/enable"
	GroupAnnotation  = "global-accelerator.alpha.wildlife.io/group"
)

func NewEndpointGroup(name, namespace string) *globalacceleratorawswildlifeiov1alpha1.EndpointGroup {
	return &globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func isServiceEnabled(service *corev1.Service) bool {
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

func GetGroupedServices(ctx context.Context, managementClient client.Client, clusterGroup *infrastructurewildlifeiov1alpha1.ClusterGroup) (map[string][]corev1.Service, error) {
	groupedServices := make(map[string][]corev1.Service)

	for _, cluster := range clusterGroup.Spec.Clusters {
		config, err := RESTConfig(ctx, managementClient, cluster.CredentialsRef)
		if err != nil {
			return nil, err
		}

		remoteClient, err := client.New(config, client.Options{})
		if err != nil {
			return nil, errors.Errorf("failed to instanciate remote client for Cluster %s", cluster.Name)
		}

		serviceList := &corev1.ServiceList{}
		err = remoteClient.List(ctx, serviceList)
		if err != nil {
			return nil, errors.Errorf("failed to list services in remote cluster %s", cluster.Name)
		}
		for _, service := range serviceList.Items {
			if isServiceEnabled(&service) {
				groupedServices[service.Annotations[GroupAnnotation]] = append(groupedServices[service.Annotations[GroupAnnotation]], service)
			}
		}
	}
	return groupedServices, nil
}

func UpdateEndpointGroupMembers(ctx context.Context, managementClient client.Client, endpointGroup *globalacceleratorawswildlifeiov1alpha1.EndpointGroup, groupMembers []string) error {
	_, err := controllerutil.CreateOrUpdate(ctx, managementClient, endpointGroup, func() error {
		endpointGroup.Spec.DNSNames = groupMembers
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
