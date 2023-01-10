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
	"os"
	"path/filepath"
	"testing"
	"time"

	globalacceleratorawswildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/globalaccelerator.aws.wildlife.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrastructurewildlifeiov1alpha1 "github.com/topfreegames/global-accelerator-operator/apis/infrastructure.wildlife.io/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx                  context.Context
	cancel               context.CancelFunc
	managementCfg        *rest.Config
	managementK8sClient  client.Client
	managementTestEnv    *envtest.Environment
	remoteCfgA           *rest.Config
	remoteK8sClientA     client.Client
	remoteTestEnvA       *envtest.Environment
	remoteCfgB           *rest.Config
	remoteK8sClientB     client.Client
	remoteTestEnvB       *envtest.Environment
	reconciliationPeriod = 5 * time.Second
)

func TestClusterGroupReconciler(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

func createKubeconfigSecret(clusterName string, cfg *rest.Config) {
	apiConfig := &api.Config{
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   cfg.Host,
				CertificateAuthorityData: cfg.CAData,
			},
		},
		Contexts: map[string]*api.Context{
			clusterName: {
				Cluster:  clusterName,
				AuthInfo: clusterName,
			},
		},
		CurrentContext: clusterName,
		AuthInfos: map[string]*api.AuthInfo{
			clusterName: {
				ClientCertificateData: cfg.CertData,
				ClientKeyData:         cfg.KeyData,
			},
		},
	}

	out, err := clientcmd.Write(*apiConfig)
	Expect(err).NotTo(HaveOccurred())

	kubeconfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kubeconfig", clusterName),
			Namespace: metav1.NamespaceDefault,
		},
		Data: map[string][]byte{
			"value": out,
		},
		Type: "cluster.x-k8s.io/secret",
	}
	Expect(managementK8sClient.Create(ctx, kubeconfigSecret)).Should(Succeed())
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	Expect(os.Setenv("KUBEBUILDER_ASSETS", "../../bin/k8s/1.23.5-darwin-amd64")).To(Succeed())

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	var err error
	managementTestEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	managementCfg, err = managementTestEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(managementCfg).NotTo(BeNil())

	managementK8sClient, err = client.New(managementCfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(managementK8sClient).NotTo(BeNil())

	remoteTestEnvA = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	remoteCfgA, err = remoteTestEnvA.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(remoteCfgA).NotTo(BeNil())

	remoteK8sClientA, err = client.New(remoteCfgA, client.Options{})
	Expect(err).NotTo(HaveOccurred())
	Expect(remoteK8sClientA).NotTo(BeNil())

	remoteTestEnvB = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	remoteCfgB, err = remoteTestEnvB.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(remoteCfgB).NotTo(BeNil())

	remoteK8sClientB, err = client.New(remoteCfgB, client.Options{})
	Expect(err).NotTo(HaveOccurred())
	Expect(remoteK8sClientB).NotTo(BeNil())

	err = infrastructurewildlifeiov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = globalacceleratorawswildlifeiov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	Expect(managementK8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "global-accelerator-operator-system",
		},
	})).Should(Succeed())

	k8sManager, err := ctrl.NewManager(managementCfg, ctrl.Options{
		Scheme:     scheme.Scheme,
		SyncPeriod: &reconciliationPeriod,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&ClusterGroupReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	createKubeconfigSecret("remote-cluster-a", remoteCfgA)
	createKubeconfigSecret("remote-cluster-b", remoteCfgB)

	serviceRemoteA := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-a",
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				EnableAnnotation: "true",
				GroupAnnotation:  "test-endpoint-group",
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
	}

	serviceRemoteB := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-b",
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				EnableAnnotation: "true",
				GroupAnnotation:  "test-endpoint-group",
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
						Hostname: "b.elb.us-east-1.amazonaws.com",
					},
				},
			},
		},
	}

	Expect(remoteK8sClientA.Create(ctx, serviceRemoteA)).Should(Succeed())
	serviceRemoteA.Status.LoadBalancer.Ingress = append(serviceRemoteA.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
		Hostname: "a.elb.us-east-1.amazonaws.com",
	})
	Expect(remoteK8sClientA.Status().Update(ctx, serviceRemoteA)).Should(Succeed())

	Expect(remoteK8sClientB.Create(ctx, serviceRemoteB)).Should(Succeed())
	serviceRemoteB.Status.LoadBalancer.Ingress = append(serviceRemoteB.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
		Hostname: "b.elb.us-east-1.amazonaws.com",
	})
	Expect(remoteK8sClientB.Status().Update(ctx, serviceRemoteB)).Should(Succeed())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := managementTestEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	err = remoteTestEnvA.Stop()
	Expect(err).NotTo(HaveOccurred())
	err = remoteTestEnvB.Stop()
	Expect(err).NotTo(HaveOccurred())
})
