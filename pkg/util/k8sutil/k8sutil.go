// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8sutil

import (
	"net"
	"os"

	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // for gcp auth
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RecommendedConfigPathEnvVar is a environment variable for path configuration
const RecommendedConfigPathEnvVar = "KUBECONFIG"

// MustNewKubeClient returns new kubernetes client for cluster configuration
func MustNewKubeClient() kubernetes.Interface {
	cfg, err := GetClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	return kubernetes.NewForConfigOrDie(cfg)
}

// GetClusterConfig obtain the config from the Kube configuration used by kubeconfig, or from k8s cluster.
func GetClusterConfig() (*rest.Config, error) {
	if len(os.Getenv(RecommendedConfigPathEnvVar)) > 0 {
		// use the current context in kubeconfig
		// This is very useful for running locally.
		return clientcmd.BuildConfigFromFlags("", os.Getenv(RecommendedConfigPathEnvVar))
	}

	// Work around https://github.com/kubernetes/kubernetes/issues/40973
	// See https://github.com/coreos/etcd-operator/issues/731#issuecomment-283804819
	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) == 0 {
		addrs, err := net.LookupHost("kubernetes.default.svc")
		if err != nil {
			panic(err)
		}
		if err := os.Setenv("KUBERNETES_SERVICE_HOST", addrs[0]); err != nil {
			return nil, err
		}
	}
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) == 0 {
		if err := os.Setenv("KUBERNETES_SERVICE_PORT", "443"); err != nil {
			panic(err)
		}
	}
	return rest.InClusterConfig()
}

// IsKubernetesResourceAlreadyExistError throws error when kubernetes resources already exist.
func IsKubernetesResourceAlreadyExistError(err error) bool {
	return apierrors.IsAlreadyExists(err)
}

// IsKubernetesResourceNotFoundError throws error when there is no kubernetes resource found.
func IsKubernetesResourceNotFoundError(err error) bool {
	return apierrors.IsNotFound(err)
}

// TODO(jlewi): CascadeDeletOptions are part of garbage collection policy.
// CascadeDeleteOptions deletes the workload after the grace period
// Do we want to use this? See
// https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
func CascadeDeleteOptions(gracePeriodSeconds int64) *metav1.DeleteOptions {
	return &metav1.DeleteOptions{
		GracePeriodSeconds: func(t int64) *int64 { return &t }(gracePeriodSeconds),
		PropagationPolicy: func() *metav1.DeletionPropagation {
			foreground := metav1.DeletePropagationForeground
			return &foreground
		}(),
	}
}
