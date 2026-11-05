/*
 * Copyright 2026 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kubernetes_api

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

var errRestartRequired = errors.New("deploy_mode kubernetes runs instances only as restarting deployments")

// Kubernetes runs each instance as a Deployment in one namespace, authenticated
// as the service account of the pod the manager runs in.
type Kubernetes struct {
	deployments typedappsv1.DeploymentInterface
}

func New(conf config.Config) (*Kubernetes, error) {
	if conf.KubernetesNamespace == "" {
		return nil, errors.New("kubernetes_namespace is required for deploy_mode kubernetes")
	}
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return newWithClient(clientset, conf.KubernetesNamespace)
}

func newWithClient(client kubernetes.Interface, namespace string) (*Kubernetes, error) {
	k := &Kubernetes{deployments: client.AppsV1().Deployments(namespace)}
	// Fails at startup rather than at the first request when the service account
	// has no role in the instance namespace.
	ctx, cancel := util.GetTimeoutContext()
	defer cancel()
	_, err := k.deployments.List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return nil, fmt.Errorf("unable to list deployments in namespace %s: %w", namespace, err)
	}
	return k, nil
}

func (k *Kubernetes) CreateContainer(name string, image string, userid string, env map[string]string, restart bool) (id string, err error) {
	if !restart {
		return "", errRestartRequired
	}
	ctx, cancel := util.GetTimeoutContext()
	defer cancel()
	_, err = k.deployments.Create(ctx, deployment(name, image, userid, env), metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return name, nil
}

// UpdateContainer replaces the pod template of the existing deployment instead of
// deleting and recreating it. The Recreate strategy stops the old pod before the
// new one starts, so two pods never forward the same topic at once.
func (k *Kubernetes) UpdateContainer(id string, name string, image string, userid string, env map[string]string, restart bool) (newId string, err error) {
	if !restart {
		return "", errRestartRequired
	}
	ctx, cancel := util.GetTimeoutContext()
	defer cancel()
	// The deployment controller writes the status between Get and Update, which
	// changes the resource version and lets the Update fail with a conflict.
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing, err := k.deployments.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			return err
		}
		// Built from the existing name, because the selector is immutable and has
		// to keep matching the template labels.
		desired := deployment(existing.Name, image, userid, env)
		existing.Labels = desired.Labels
		existing.Spec.Strategy = desired.Spec.Strategy
		existing.Spec.Template = desired.Spec.Template
		_, err = k.deployments.Update(ctx, existing, metav1.UpdateOptions{})
		newId = existing.Name
		return err
	})
	if k8serrors.IsNotFound(err) {
		return k.CreateContainer(name, image, userid, env, restart)
	}
	if err != nil {
		return "", err
	}
	return newId, nil
}

// RemoveContainer treats a missing deployment as removed. The controller deletes an
// instance only after this succeeds, so an error here would keep it forever.
func (k *Kubernetes) RemoveContainer(id string) (err error) {
	ctx, cancel := util.GetTimeoutContext()
	defer cancel()
	err = k.deployments.Delete(ctx, id, metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (k *Kubernetes) ContainerExists(id string) (exists bool, err error) {
	ctx, cancel := util.GetTimeoutContext()
	defer cancel()
	_, err = k.deployments.Get(ctx, id, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

func deployment(name string, image string, userid string, env map[string]string) *appsv1.Deployment {
	labels := map[string]string{
		"user":       userid,
		"kafka2mqtt": name,
	}
	// Sorted so that an update with unchanged values leaves the pod template as it
	// is and does not restart the pod.
	envVars := []corev1.EnvVar{}
	for _, key := range slices.Sorted(maps.Keys(env)) {
		envVars = append(envVars, corev1.EnvVar{Name: key, Value: env[key]})
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: new(int32(1)),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"kafka2mqtt": name}},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: new(false),
					NodeSelector:                 map[string]string{"role": "worker"},
					Containers: []corev1.Container{{
						Name:            name,
						Image:           image,
						ImagePullPolicy: corev1.PullAlways,
						Env:             envVars,
					}},
				},
			},
		},
	}
}
