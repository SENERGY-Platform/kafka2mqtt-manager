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
	"context"
	"errors"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

const namespace = "kafka2mqtt-instances"

func newTestClient(t *testing.T) (*Kubernetes, *fake.Clientset) {
	t.Helper()
	client := fake.NewClientset()
	k, err := newWithClient(client, namespace)
	if err != nil {
		t.Fatal(err)
	}
	return k, client
}

func getDeployment(t *testing.T, client *fake.Clientset, name string) *appsv1.Deployment {
	t.Helper()
	d, err := client.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestNewFailsWithoutAccessToNamespace(t *testing.T) {
	client := fake.NewClientset()
	client.PrependReactor("list", "deployments", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, k8serrors.NewForbidden(schema.GroupResource{Group: "apps", Resource: "deployments"}, "", errors.New("no role"))
	})
	_, err := newWithClient(client, namespace)
	if !k8serrors.IsForbidden(err) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestCreateContainerCreatesDeployment(t *testing.T) {
	k, client := newTestClient(t)
	id, err := k.CreateContainer("k2m-1", "image:1", "user-1", map[string]string{"B": "2", "A": "1"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if id != "k2m-1" {
		t.Fatalf("expected id k2m-1, got %s", id)
	}
	d := getDeployment(t, client, "k2m-1")
	if d.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
		t.Errorf("expected Recreate strategy, got %s", d.Spec.Strategy.Type)
	}
	if d.Spec.Selector.MatchLabels["kafka2mqtt"] != "k2m-1" || d.Spec.Template.Labels["kafka2mqtt"] != "k2m-1" {
		t.Errorf("selector %v does not match template labels %v", d.Spec.Selector.MatchLabels, d.Spec.Template.Labels)
	}
	if d.Spec.Template.Labels["user"] != "user-1" {
		t.Errorf("expected user label user-1, got %v", d.Spec.Template.Labels)
	}
	if *d.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Error("instance pods must not mount a service account token")
	}
	c := d.Spec.Template.Spec.Containers[0]
	if c.Image != "image:1" {
		t.Errorf("expected image image:1, got %s", c.Image)
	}
	expectedEnv := []corev1.EnvVar{{Name: "A", Value: "1"}, {Name: "B", Value: "2"}}
	if len(c.Env) != 2 || c.Env[0] != expectedEnv[0] || c.Env[1] != expectedEnv[1] {
		t.Errorf("expected env %v, got %v", expectedEnv, c.Env)
	}
}

func TestContainersMustRestart(t *testing.T) {
	k, _ := newTestClient(t)
	if _, err := k.CreateContainer("k2m-1", "image:1", "user-1", nil, false); !errors.Is(err, errRestartRequired) {
		t.Errorf("create: expected errRestartRequired, got %v", err)
	}
	if _, err := k.UpdateContainer("k2m-1", "k2m-1", "image:1", "user-1", nil, false); !errors.Is(err, errRestartRequired) {
		t.Errorf("update: expected errRestartRequired, got %v", err)
	}
}

func TestUpdateContainerReplacesPodTemplate(t *testing.T) {
	k, client := newTestClient(t)
	_, err := k.CreateContainer("k2m-1", "image:1", "user-1", map[string]string{"KAFKA_GROUP_ID": "a"}, true)
	if err != nil {
		t.Fatal(err)
	}
	id, err := k.UpdateContainer("k2m-1", "k2m-1", "image:2", "user-1", map[string]string{"KAFKA_GROUP_ID": "b"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if id != "k2m-1" {
		t.Fatalf("expected id k2m-1, got %s", id)
	}
	c := getDeployment(t, client, "k2m-1").Spec.Template.Spec.Containers[0]
	if c.Image != "image:2" || c.Env[0].Value != "b" {
		t.Errorf("expected image:2 with KAFKA_GROUP_ID b, got %s with %v", c.Image, c.Env)
	}
}

func TestUpdateContainerWithUnchangedValuesKeepsPodTemplate(t *testing.T) {
	k, client := newTestClient(t)
	env := map[string]string{"A": "1", "B": "2", "C": "3"}
	_, err := k.CreateContainer("k2m-1", "image:1", "user-1", env, true)
	if err != nil {
		t.Fatal(err)
	}
	before := getDeployment(t, client, "k2m-1")
	_, err = k.UpdateContainer("k2m-1", "k2m-1", "image:1", "user-1", env, true)
	if err != nil {
		t.Fatal(err)
	}
	after := getDeployment(t, client, "k2m-1")
	if !equality.Semantic.DeepEqual(before.Spec.Template, after.Spec.Template) {
		t.Errorf("pod template changed:\nbefore %v\nafter  %v", before.Spec.Template, after.Spec.Template)
	}
	if !equality.Semantic.DeepEqual(before.Spec.Selector, after.Spec.Selector) {
		t.Errorf("selector changed from %v to %v", before.Spec.Selector, after.Spec.Selector)
	}
}

func TestUpdateContainerRetriesOnConflict(t *testing.T) {
	k, client := newTestClient(t)
	_, err := k.CreateContainer("k2m-1", "image:1", "user-1", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	conflicts := 1
	client.PrependReactor("update", "deployments", func(ktesting.Action) (bool, runtime.Object, error) {
		if conflicts > 0 {
			conflicts--
			return true, nil, k8serrors.NewConflict(schema.GroupResource{Group: "apps", Resource: "deployments"}, "k2m-1", errors.New("object has been modified"))
		}
		return false, nil, nil
	})
	_, err = k.UpdateContainer("k2m-1", "k2m-1", "image:2", "user-1", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if image := getDeployment(t, client, "k2m-1").Spec.Template.Spec.Containers[0].Image; image != "image:2" {
		t.Errorf("expected image:2 after retry, got %s", image)
	}
}

func TestUpdateContainerCreatesMissingDeployment(t *testing.T) {
	k, client := newTestClient(t)
	id, err := k.UpdateContainer("k2m-gone", "k2m-1", "image:1", "user-1", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if id != "k2m-1" {
		t.Fatalf("expected id k2m-1, got %s", id)
	}
	getDeployment(t, client, "k2m-1")
}

func TestRemoveContainer(t *testing.T) {
	k, _ := newTestClient(t)
	_, err := k.CreateContainer("k2m-1", "image:1", "user-1", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	exists, err := k.ContainerExists("k2m-1")
	if err != nil || !exists {
		t.Fatalf("expected k2m-1 to exist, got %v, %v", exists, err)
	}
	if err = k.RemoveContainer("k2m-1"); err != nil {
		t.Fatal(err)
	}
	exists, err = k.ContainerExists("k2m-1")
	if err != nil || exists {
		t.Fatalf("expected k2m-1 to be gone, got %v, %v", exists, err)
	}
	if err = k.RemoveContainer("k2m-1"); err != nil {
		t.Errorf("removing a missing deployment must succeed, got %v", err)
	}
}
