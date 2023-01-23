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
	globalacceleratorawswildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/globalaccelerator.aws.wildlife.io/v1alpha1"
	infrastructurewildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/infrastructure.wildlife.io/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("EndpointGroup controller", func() {
	const (
		timeout  = time.Second * 60
		interval = time.Millisecond * 250
	)

	var clusterGroup *infrastructurewildlifeiov1alpha1.ClusterGroup

	BeforeEach(func() {
		clusterGroup = &infrastructurewildlifeiov1alpha1.ClusterGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster-group",
				Namespace: "global-accelerator-operator-system",
			},
		}
		managementK8sClient.Delete(ctx, clusterGroup)
		endpointGroup := &globalacceleratorawswildlifeiov1alpha1.EndpointGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-endpoint-group",
				Namespace: "global-accelerator-operator-system",
			},
		}
		managementK8sClient.Delete(ctx, endpointGroup)
	})

	endpointGroupKey := types.NamespacedName{Name: "test-endpoint-group", Namespace: "global-accelerator-operator-system"}
	endpointGroup := &globalacceleratorawswildlifeiov1alpha1.EndpointGroup{}

	DescribeTable("Reconcile controller",
		func(remoteClusters []infrastructurewildlifeiov1alpha1.Cluster, eventualFunc func() bool, expectedDNSNamesLen int, funcUpdateK8sResources func()) {
			clusterGroup.Spec.Clusters = remoteClusters
			if funcUpdateK8sResources != nil {
				funcUpdateK8sResources()
			}
			Expect(managementK8sClient.Create(ctx, clusterGroup)).Should(Succeed())
			Eventually(eventualFunc, timeout, interval).Should(BeTrue())
			Expect(len(endpointGroup.Spec.DNSNames)).Should(Equal(expectedDNSNamesLen))
		},

		Entry("create ClusterGroup with remote-cluster-a",
			[]infrastructurewildlifeiov1alpha1.Cluster{
				{
					Name: "remote-cluster-a",
					CredentialsRef: corev1.ObjectReference{
						Name:      "remote-cluster-a-kubeconfig",
						Namespace: metav1.NamespaceDefault,
					},
				},
			},
			func() bool {
				err := managementK8sClient.Get(ctx, endpointGroupKey, endpointGroup)
				return err == nil
			},
			1,
			nil,
		),
		Entry("add remote-cluster-b to ClusterGroup",
			[]infrastructurewildlifeiov1alpha1.Cluster{
				{
					Name: "remote-cluster-a",
					CredentialsRef: corev1.ObjectReference{
						Name:      "remote-cluster-a-kubeconfig",
						Namespace: metav1.NamespaceDefault,
					},
				},
				{
					Name: "remote-cluster-b",
					CredentialsRef: corev1.ObjectReference{
						Name:      "remote-cluster-b-kubeconfig",
						Namespace: metav1.NamespaceDefault,
					},
				},
			},
			func() bool {
				err := managementK8sClient.Get(ctx, endpointGroupKey, endpointGroup)
				if err != nil {
					return false
				}
				if len(endpointGroup.Spec.DNSNames) != 2 {
					return false
				}
				return true
			},
			2,
			nil,
		),
		Entry("remove annotation from test-service-b",
			[]infrastructurewildlifeiov1alpha1.Cluster{
				{
					Name: "remote-cluster-a",
					CredentialsRef: corev1.ObjectReference{
						Name:      "remote-cluster-a-kubeconfig",
						Namespace: metav1.NamespaceDefault,
					},
				},
				{
					Name: "remote-cluster-b",
					CredentialsRef: corev1.ObjectReference{
						Name:      "remote-cluster-b-kubeconfig",
						Namespace: metav1.NamespaceDefault,
					},
				},
			},
			func() bool {
				err := managementK8sClient.Get(ctx, endpointGroupKey, endpointGroup)
				if err != nil {
					return false
				}
				if len(endpointGroup.Spec.DNSNames) != 1 {
					return false
				}
				if endpointGroup.Spec.DNSNames[0] != "a.elb.us-east-1.amazonaws.com" {
					return false
				}

				return true
			},
			1,
			func() {
				currentServiceRemoteB := &corev1.Service{}
				Expect(remoteK8sClientB.Get(ctx, client.ObjectKey{
					Name:      "test-service-b",
					Namespace: metav1.NamespaceDefault,
				}, currentServiceRemoteB)).Should(Succeed())
				delete(currentServiceRemoteB.ObjectMeta.Annotations, EnableAnnotation)
				Expect(remoteK8sClientB.Update(ctx, currentServiceRemoteB)).Should(Succeed())
			},
		),
	)

})

func TestGetDNSNameFromGroupedServices(t *testing.T) {
	testCases := []struct {
		description    string
		services       []corev1.Service
		expectedOutput []string
		expectedError  bool
	}{
		{
			description: "should return Hostname of test-service-a",
			services: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-service-a",
					},
					Status: corev1.ServiceStatus{
						LoadBalancer: corev1.LoadBalancerStatus{
							Ingress: []corev1.LoadBalancerIngress{
								{
									Hostname: "a.elb.us-east-1.amazonaws.com",
								},
							},
						},
					},
				},
			},
			expectedOutput: []string{
				"a.elb.us-east-1.amazonaws.com",
			},
		},
		{
			description: "should return Hostname of test-service-a and test-service-b",
			services: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-service-a",
					},
					Status: corev1.ServiceStatus{
						LoadBalancer: corev1.LoadBalancerStatus{
							Ingress: []corev1.LoadBalancerIngress{
								{
									Hostname: "a.elb.us-east-1.amazonaws.com",
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-service-b",
					},
					Status: corev1.ServiceStatus{
						LoadBalancer: corev1.LoadBalancerStatus{
							Ingress: []corev1.LoadBalancerIngress{
								{
									Hostname: "b.elb.us-east-1.amazonaws.com",
								},
							},
						},
					},
				},
			},
			expectedOutput: []string{
				"a.elb.us-east-1.amazonaws.com",
				"b.elb.us-east-1.amazonaws.com",
			},
		},
		{
			description: "should return error with test-service-a without Hostname",
			services: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-service-a",
					},
				},
			},
			expectedError: true,
		},
	}

	RegisterFailHandler(Fail)
	g := NewWithT(t)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			dnsNames, err := getDNSNameFromGroupedServices(tc.services)
			if !tc.expectedError {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(dnsNames).To(BeEquivalentTo(tc.expectedOutput))
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}
