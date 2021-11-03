/*
Copyright 2021.

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
	"errors"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	KUBECONFIG_SECRET = "kubeconfig"
	GITHUB_API_SECRET = "github-api-token"
	PYXIS_API_SECRET  = "pyxis-api-secret"
)

// reconcileKubeConfigSecret will ensure that the kubeconfig Secret is present and up to date.
func (r *OperatorPipelineReconciler) reconcileKubeConfigSecret(meta metav1.ObjectMeta) error {
	key := types.NamespacedName{
		Namespace: meta.Namespace,
		Name:      KUBECONFIG_SECRET,
	}

	secret := newSecret(key)
	if IsObjectFound(r.Client, key, secret) {
		log.Info("existing kubeconfig secret found")
		return nil // Existing Secret found, do nothing...
	}

	log.Info("creating new kubeconfig secret")
	return r.Client.Create(context.TODO(), secret)
}

// reconcileGitHubAPISecret will ensure that the GitHub API Secret is present and up to date.
func (r *OperatorPipelineReconciler) reconcileGitHubAPISecret(meta metav1.ObjectMeta) error {
	key := types.NamespacedName{
		Namespace: meta.Namespace,
		Name:      GITHUB_API_SECRET,
	}

	secret := newSecret(key)
	if IsObjectFound(r.Client, key, secret) {
		log.Info("existing github api secret found")
		return nil // Existing Secret found, do nothing...
	}

	return errors.New("github api secret not found")
}

// reconcilePyxisAPISecret will ensure that the Pyxis API Secret is present and up to date.
func (r *OperatorPipelineReconciler) reconcilePyxisAPISecret(meta metav1.ObjectMeta) error {
	key := types.NamespacedName{
		Namespace: meta.Namespace,
		Name:      PYXIS_API_SECRET,
	}

	secret := newSecret(key)
	if IsObjectFound(r.Client, key, secret) {
		log.Info("existing pyxis api secret found")
		return nil // Existing Secret found, do nothing...
	}

	return errors.New("pyxis api secret not found")
}

// newSecret will create and return a new Secret instance using the given Name/Namespace.
func newSecret(key types.NamespacedName) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}
}
