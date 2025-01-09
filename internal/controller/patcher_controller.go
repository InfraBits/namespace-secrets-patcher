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
	namespacesecretspatcherv1 "github.com/infrabits/namespace-secrets-patcher/api/v1"
	patcherv1 "github.com/infrabits/namespace-secrets-patcher/api/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PatcherReconciler reconciles a Patcher object
type PatcherReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=namespace-secrets-patcher.infrabits.nl,resources=patchers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=namespace-secrets-patcher.infrabits.nl,resources=patchers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=namespace-secrets-patcher.infrabits.nl,resources=patchers/finalizers,verbs=update
// +kubebuilder:rbac:groups=namespace-secrets-patcher.infrabits.nl,resources=namespace,verbs=get;list;watch
// +kubebuilder:rbac:groups=namespace-secrets-patcher.infrabits.nl,resources=secret,verbs=get;create;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the patcher
	var patcher patcherv1.Patcher
	if err := r.Get(
		ctx,
		client.ObjectKey{Name: req.Name, Namespace: req.Namespace},
		&patcher,
	); err != nil {
		if apierrors.IsNotFound(err) {
			// Happens on deletion - we do not cascade delete's so nothing todo
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the secret
	var secret v1.Secret
	if err := r.Get(ctx, client.ObjectKey{Name: patcher.Spec.Secret, Namespace: patcher.Namespace}, &secret); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch all namespaces to match against
	var namespaceList v1.NamespaceList
	if err := r.List(
		ctx,
		&namespaceList,
	); err != nil {
		return ctrl.Result{}, err
	}

	// Patch as required
	for _, namespace := range namespaceList.Items {
		if patcher.NameSpaceIsTarget(namespace.Name) {
			var targetSecret v1.Secret
			if err := r.Get(ctx, client.ObjectKey{Name: patcher.Spec.Secret, Namespace: namespace.Name}, &targetSecret); err != nil {
				if !apierrors.IsNotFound(err) {
					continue
				}

				logger.Info("Creating secret in target namespace",
					"source", patcher.Namespace,
					"target", namespace.Name,
					"secret", patcher.Spec.Secret,
				)
				secret.DeepCopyInto(&targetSecret)
				targetSecret.Namespace = namespace.Name
				targetSecret.ResourceVersion = ""
				if err := r.Client.Create(ctx, &targetSecret); err != nil {
					// This happens when a namespace is being deleted
					if !apierrors.IsNotFound(err) {
						return ctrl.Result{}, err
					}
				}
				continue
			}

			if !equality.Semantic.DeepEqual(secret.Data, targetSecret.Data) {
				logger.Info("Updating secret in target namespace",
					"source", patcher.Namespace,
					"target", namespace.Name,
					"secret", patcher.Spec.Secret,
				)
				targetSecret.Data = secret.Data
				if err := r.Client.Update(ctx, &targetSecret); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&namespacesecretspatcherv1.Patcher{}).
		Watches(
			&v1.Secret{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, secret client.Object) []reconcile.Request {
				// Find any patcher instances referencing this secret
				var patcherList patcherv1.PatcherList
				if err := r.List(
					ctx,
					&patcherList,
				); err != nil {
					return []reconcile.Request{}
				}

				requests := []reconcile.Request{}
				for _, patcher := range patcherList.Items {
					if patcher.Spec.Secret == secret.GetName() {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      patcher.Name,
								Namespace: patcher.Namespace,
							},
						})
					}
				}
				return requests
			}),
		).
		Watches(
			&v1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, namespace client.Object) []reconcile.Request {
				// Find any patcher instances referencing (directly or indirectly) this namespace
				var patcherList patcherv1.PatcherList
				if err := r.List(
					ctx,
					&patcherList,
				); err != nil {
					return []reconcile.Request{}
				}

				requests := []reconcile.Request{}
				for _, patcher := range patcherList.Items {
					if patcher.NameSpaceIsTarget(namespace.GetName()) {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      patcher.Name,
								Namespace: patcher.Namespace,
							},
						})
					}
				}
				return requests
			}),
		).
		Complete(r)
}
