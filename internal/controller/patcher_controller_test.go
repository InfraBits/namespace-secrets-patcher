/*
Copyright 2025 Infra Bits.

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

package controller

import (
	"context"

	v1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	namespacesecretspatcherv1 "github.com/infrabits/namespace-secrets-patcher/api/v1"
)

var _ = Describe("Patcher Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      "test-resource",
			Namespace: "source-namespace",
		}
		patcher := &namespacesecretspatcherv1.Patcher{}
		sourceNamespace := &v1.Namespace{}
		sourceSecret := &v1.Secret{}
		targetNamespace := &v1.Namespace{}

		BeforeEach(func() {
			By("creating the source namespace")
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name: "source-namespace",
			}, sourceNamespace); err != nil && errors.IsNotFound(err) {
				sourceNamespace = &v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "source-namespace",
					},
				}
				Expect(k8sClient.Create(ctx, sourceNamespace)).To(Succeed())
			}

			By("creating the source secret")
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "source-secret",
				Namespace: "source-namespace",
			}, sourceSecret); err != nil && errors.IsNotFound(err) {
				sourceSecret = &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "source-secret",
						Namespace: "source-namespace",
					},
					Type: v1.SecretTypeOpaque,
					Data: map[string][]byte{
						"test": []byte("abc213"),
					},
				}
				Expect(k8sClient.Create(ctx, sourceSecret)).To(Succeed())
			}

			By("creating the target namespace")
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name: "target-namespace",
			}, targetNamespace); err != nil && errors.IsNotFound(err) {
				targetNamespace = &v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "target-namespace",
					},
				}
				Expect(k8sClient.Create(ctx, targetNamespace)).To(Succeed())
			}

			By("creating the custom resource for the Kind Patcher")
			if err := k8sClient.Get(ctx, typeNamespacedName, patcher); err != nil && errors.IsNotFound(err) {
				resource := &namespacesecretspatcherv1.Patcher{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-resource",
						Namespace: "source-namespace",
					},
					Spec: namespacesecretspatcherv1.PatcherSpec{
						Secret: "source-secret",
						Targets: []namespacesecretspatcherv1.TargetSpec{
							{
								Name: "target-namespace",
								Type: "match",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &namespacesecretspatcherv1.Patcher{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the source namespace")
			Expect(k8sClient.Delete(ctx, sourceNamespace)).To(Succeed())

			By("Cleanup the source secret")
			Expect(k8sClient.Delete(ctx, sourceSecret)).To(Succeed())

			By("Cleanup the target namespace")
			Expect(k8sClient.Delete(ctx, targetNamespace)).To(Succeed())

			By("Cleanup the specific resource instance Patcher")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &PatcherReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			targetSecret := &v1.Secret{}
			fetchErr := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "source-secret",
				Namespace: "target-namespace",
			}, targetSecret)
			Expect(fetchErr).NotTo(HaveOccurred())

			Expect(targetSecret.Data["test"]).To(Equal([]byte("abc213")))
		})
	})
})
