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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	infrastructurewildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/v1alpha1"
)

var _ = Describe("EndpointGroup controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)
	clusterGroup := &infrastructurewildlifeiov1alpha1.ClusterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-group",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: infrastructurewildlifeiov1alpha1.ClusterGroupSpec{
			Clusters: []infrastructurewildlifeiov1alpha1.Cluster{
				{
					Name: "remote-cluster-a",
					CredentialsRef: corev1.ObjectReference{
						Name:      "remote-cluster-a-kubeconfig",
						Namespace: metav1.NamespaceDefault,
					},
				},
			},
		},
	}

	Context("Creating new ClusterGroup in Management Cluster", func() {
		It("Should create the EndpointGroup CR", func() {
			Expect(managementK8sClient.Create(ctx, clusterGroup)).Should(Succeed())
			clusterGroupKey := types.NamespacedName{Name: "test-cluster-group", Namespace: metav1.NamespaceDefault}
			retrievedClusterGroup := &infrastructurewildlifeiov1alpha1.ClusterGroup{}
			Eventually(func() bool {
				Expect(managementK8sClient.Get(ctx, clusterGroupKey, retrievedClusterGroup)).Should(Succeed())
				return len(retrievedClusterGroup.Spec.Clusters) == 1
			}, timeout, interval).Should(BeTrue())

			clusterGroup.Spec.Clusters = append(clusterGroup.Spec.Clusters, infrastructurewildlifeiov1alpha1.Cluster{
				Name: "remote-cluster-b",
				CredentialsRef: corev1.ObjectReference{
					Name:      "remote-cluster-b-kubeconfig",
					Namespace: metav1.NamespaceDefault,
				},
			})

			Expect(managementK8sClient.Update(ctx, clusterGroup)).Should(Succeed())
			Eventually(func() bool {
				Expect(managementK8sClient.Get(ctx, clusterGroupKey, retrievedClusterGroup)).Should(Succeed())
				return len(retrievedClusterGroup.Spec.Clusters) == 2
			}, timeout, interval).Should(BeTrue())

			// TODO: Should evaluate the creation of EndpointGroup CR
		})
	})
})
