package kubernetes

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	globalacceleratorawswildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/globalaccelerator.aws.wildlife.io/v1alpha1"
	infrastructurewildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/infrastructure.wildlife.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func assertService(expectedServiceName string, services []corev1.Service) bool {
	for _, service := range services {
		if service.Name == expectedServiceName {
			return true
		}
	}
	return false
}

func assertGroupedServices(expected map[string][]string, actual map[string][]corev1.Service) bool {
	if len(expected) != len(actual) {
		return false
	}

	for group, serviceNames := range expected {
		if services, found := actual[group]; found {
			for _, serviceName := range serviceNames {
				if !assertService(serviceName, services) {
					return false
				}
			}
		} else {
			return false
		}
	}
	return true

}

func getRemoteService(name, group string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("test-service-%s", name),
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				EnableAnnotation: "true",
				GroupAnnotation:  group,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol: "TCP",
					Port:     80,
					Name:     "http",
				},
			},
			Type: corev1.ServiceTypeLoadBalancer,
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						Hostname: fmt.Sprintf("%s.elb.us-east-1.amazonaws.com", name),
					},
				},
			},
		},
	}
}

func getClusterGroup(name string, clusters []string) *infrastructurewildlifeiov1alpha1.ClusterGroup {
	clusterGroup := &infrastructurewildlifeiov1alpha1.ClusterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "global-accelerator-operator-system",
		},
	}
	for _, clusterName := range clusters {
		cluster := infrastructurewildlifeiov1alpha1.Cluster{
			Name: fmt.Sprintf("remote-cluster-%s", clusterName),
			CredentialsRef: corev1.ObjectReference{
				Name:      fmt.Sprintf("remote-cluster-%s-kubeconfig", clusterName),
				Namespace: metav1.NamespaceDefault,
			},
		}
		clusterGroup.Spec.Clusters = append(clusterGroup.Spec.Clusters, cluster)
	}
	return clusterGroup
}

var _ = Describe("Kubernetes package", func() {
	BeforeEach(func() {
		managementK8sClient.Delete(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "remote-cluster-a-kubeconfig",
				Namespace: metav1.NamespaceDefault,
			},
		})
		managementK8sClient.Delete(ctx, &globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-endpoint-group",
				Namespace: metav1.NamespaceDefault,
			},
		})
		remoteK8sClientA.DeleteAllOf(ctx, &corev1.Service{}, client.InNamespace(metav1.NamespaceDefault))
		remoteK8sClientB.DeleteAllOf(ctx, &corev1.Service{}, client.InNamespace(metav1.NamespaceDefault))
	})

	DescribeTable("GetGroupedServices",
		func(funcUpdateK8sResources func(), clusterGroup *infrastructurewildlifeiov1alpha1.ClusterGroup, expectedOutput map[string][]string, expectedError bool) {
			if funcUpdateK8sResources != nil {
				funcUpdateK8sResources()
			}
			services, err := GetGroupedServices(context.TODO(), managementK8sClient, clusterGroup)
			if expectedError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
			if expectedOutput != nil {
				Expect(assertGroupedServices(expectedOutput, services)).To(BeTrue())
			}
		},

		Entry("without kubeconfig created", nil, getClusterGroup("test-cluster-group", []string{"a"}), nil, true),
		Entry("with invalid kubeconfig", func() {
			wrongRemoteCfgA := &rest.Config{
				Host: "invalid-server",
			}
			createKubeconfigSecret("remote-cluster-a", wrongRemoteCfgA)
		}, getClusterGroup("test-cluster-group", []string{"a"}), nil, true),
		Entry("should return test-service-a", func() {
			createKubeconfigSecret("remote-cluster-a", remoteCfgA)
			serviceRemoteA := getRemoteService("a", "test-endpoint-group")
			Expect(remoteK8sClientA.Create(ctx, serviceRemoteA)).Should(Succeed())
			serviceRemoteA.Status.LoadBalancer.Ingress = append(serviceRemoteA.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
				Hostname: "a.elb.us-east-1.amazonaws.com",
			})
			Expect(remoteK8sClientA.Status().Update(ctx, serviceRemoteA)).Should(Succeed())
		}, getClusterGroup("test-cluster-group", []string{"a"}),
			map[string][]string{
				"test-endpoint-group": {
					"test-service-a",
				},
			},
			false,
		),
		Entry("should return test-service-a and test-service-b in different groups", func() {
			createKubeconfigSecret("remote-cluster-a", remoteCfgA)
			createKubeconfigSecret("remote-cluster-b", remoteCfgB)

			serviceRemoteA := getRemoteService("a", "test-endpoint-group")
			Expect(remoteK8sClientA.Create(ctx, serviceRemoteA)).Should(Succeed())

			serviceRemoteB := getRemoteService("b", "test-endpoint-group-another")
			Expect(remoteK8sClientB.Create(ctx, serviceRemoteB)).Should(Succeed())
		}, getClusterGroup("test-cluster-group", []string{"a", "b"}),
			map[string][]string{
				"test-endpoint-group": {
					"test-service-a",
				},
				"test-endpoint-group-another": {
					"test-service-b",
				},
			},
			false,
		),
	)

	DescribeTable("RESTConfig",
		func(funcUpdateK8sResources func(), expectedError bool) {
			if funcUpdateK8sResources != nil {
				funcUpdateK8sResources()
			}
			_, err := RESTConfig(context.TODO(), managementK8sClient, corev1.ObjectReference{
				Name:      "remote-cluster-a-kubeconfig",
				Namespace: metav1.NamespaceDefault,
			})
			if expectedError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		Entry("without kubeconfig created", nil, true),
		Entry("with kubeconfig created, but wrong data key", func() {
			kubeconfigSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "remote-cluster-a-kubeconfig",
					Namespace: metav1.NamespaceDefault,
				},
				Data: map[string][]byte{
					"wrong": []byte(""),
				},
				Type: "cluster.x-k8s.io/secret",
			}
			Expect(managementK8sClient.Create(ctx, kubeconfigSecret)).Should(Succeed())
		}, true),
		Entry("with kubeconfig created, but invalid", func() {
			kubeconfigSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "remote-cluster-a-kubeconfig",
					Namespace: metav1.NamespaceDefault,
				},
				Data: map[string][]byte{
					"value": []byte(""),
				},
				Type: "cluster.x-k8s.io/secret",
			}
			Expect(managementK8sClient.Create(ctx, kubeconfigSecret)).Should(Succeed())
		}, true),
		Entry("with kubeconfig created, should succeed", func() {
			createKubeconfigSecret("remote-cluster-a", remoteCfgA)
		}, false),
	)

})

func TestUpdateEndpointGroupMembers(t *testing.T) {
	testCases := []struct {
		description   string
		endpointGroup *globalacceleratorawswildlifeiov1alpha1.EndpointGroup
		groupMembers  []string
	}{
		{
			description: "should create EndpointGroup with dnsName a",
			groupMembers: []string{
				"a.elb.us-east-1.amazonaws.com",
			},
		},
		{
			description:   "should update EndpointGroup with dnsNames a and b",
			endpointGroup: NewEndpointGroup("test-endpoint-group", metav1.NamespaceDefault),
			groupMembers: []string{
				"a.elb.us-east-1.amazonaws.com",
				"b.elb.us-east-1.amazonaws.com",
			},
		},
	}

	RegisterFailHandler(Fail)
	g := NewWithT(t)

	err := globalacceleratorawswildlifeiov1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			var fakeManagementClient client.WithWatch
			if tc.endpointGroup != nil {
				fakeManagementClient = fake.NewClientBuilder().WithObjects(tc.endpointGroup).WithScheme(scheme.Scheme).Build()
			} else {
				fakeManagementClient = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			}
			endpointGroup := NewEndpointGroup("test-endpoint-group", metav1.NamespaceDefault)

			err := UpdateEndpointGroupMembers(context.TODO(), fakeManagementClient, endpointGroup, tc.groupMembers)
			g.Expect(err).ToNot(HaveOccurred())
			err = fakeManagementClient.Get(context.TODO(), types.NamespacedName{
				Name:      "test-endpoint-group",
				Namespace: metav1.NamespaceDefault,
			}, endpointGroup)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(endpointGroup.Spec.DNSNames).To(BeEquivalentTo(tc.groupMembers))
		})
	}
}

func TestIsServiceEnabled(t *testing.T) {
	testCases := []struct {
		description    string
		input          *corev1.Service
		expectedOutput bool
	}{
		{
			description: "should return true with Service with correct annotations",
			input: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: metav1.NamespaceDefault,
					Annotations: map[string]string{
						EnableAnnotation: "true",
						GroupAnnotation:  "test-endpoint-group",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
			},
			expectedOutput: true,
		},
		{
			description: "should return true when missing Group annotation",
			input: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: metav1.NamespaceDefault,
					Annotations: map[string]string{
						EnableAnnotation: "true",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
			},
			expectedOutput: false,
		},
		{
			description: "should return true when missing enable annotation",
			input: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: metav1.NamespaceDefault,
					Annotations: map[string]string{
						GroupAnnotation: "test-endpoint-group",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
			},
			expectedOutput: false,
		},
		{
			description: "should return true when enable annotation value is different from true",
			input: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: metav1.NamespaceDefault,
					Annotations: map[string]string{
						EnableAnnotation: "false",
						GroupAnnotation:  "test-endpoint-group",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
			},
			expectedOutput: false,
		},
		{
			description: "should return true when Service type is different from Load Balancer",
			input: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: metav1.NamespaceDefault,
					Annotations: map[string]string{
						EnableAnnotation: "true",
						GroupAnnotation:  "test-endpoint-group",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
				},
			},
			expectedOutput: false,
		},
	}

	RegisterFailHandler(Fail)
	g := NewWithT(t)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			isEnabled := isServiceEnabled(tc.input)
			g.Expect(isEnabled).To(BeEquivalentTo(tc.expectedOutput))

		})
	}
}
