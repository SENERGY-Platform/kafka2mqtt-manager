/*
 * Copyright 2020 InfAI (CC SES)
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

package rancher2_api

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"net/http"
	"strconv"

	"github.com/parnurzeal/gorequest"
)

type Rancher2 struct {
	url         string
	accessKey   string
	secretKey   string
	namespaceId string
	projectId   string
}

func New(config config.Config) *Rancher2 {
	return &Rancher2{config.RancherUrl, config.RancherAccessKey, config.RancherSecretKey, config.RancherNamespaceId, config.RancherProjectId}
}

func (r *Rancher2) UpdateContainer(id string, name string, image string, env map[string]string, restart bool) (newId string, err error) {
	err = r.RemoveContainer(id)
	if err != nil {
		return newId, err
	}
	return r.CreateContainer(name, image, env, restart)
}

func (r *Rancher2) CreateContainer(name string, image string, env map[string]string, restart bool) (id string, err error) {
	request := gorequest.New().SetBasicAuth(r.accessKey, r.secretKey).TLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	r2Env := []Env{}
	for k, v := range env {
		r2Env = append(r2Env, Env{
			Name:  k,
			Value: v,
		})
	}
	reqBody := &Request{
		Name:        name,
		NamespaceId: r.namespaceId,
		Containers: []Container{{
			Image:           image,
			Name:            name,
			Env:             r2Env,
			ImagePullPolicy: "Always",
		}},
		Scheduling: Scheduling{Scheduler: "default-scheduler", Node: Node{RequireAll: []string{"role=worker"}}},
	}
	request.Method = "POST"
	request.Url = r.url + "projects/" + r.projectId
	if restart {
		request.Url += "/workloads"
		reqBody.Labels = map[string]string{"kafka2mqtt": name}
		reqBody.Selector = Selector{MatchLabels: map[string]string{"kafka2mqtt": name}}
	} else {
		request.Url += "/jobs"
	}
	resp, body, e := request.Send(reqBody).End()
	if resp.StatusCode != http.StatusCreated {
		err = errors.New("could not create export")
		fmt.Print(body)
		return
	}
	if len(e) > 0 {
		err = errors.New("could not create export")
		return
	}
	return name, err
}

func (r *Rancher2) RemoveContainer(id string) (err error) {
	request := gorequest.New().SetBasicAuth(r.accessKey, r.secretKey).TLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	resp, body, e := request.Delete(r.url + "projects/" + r.projectId + "/workloads/deployment:" +
		r.namespaceId + ":" + id).End()
	if resp.StatusCode == http.StatusNotFound {
		resp, body, e = request.Delete(r.url + "projects/" + r.projectId + "/workloads/job:" +
			r.namespaceId + ":" + id).End()
	}
	if resp.StatusCode != http.StatusNoContent {
		err = errors.New("could not delete export: " + body)
		return
	}
	if len(e) > 0 {
		err = errors.New("something went wrong")
		return
	}
	return
}

func (r *Rancher2) ContainerExists(id string) (exists bool, err error) {
	request := gorequest.New().SetBasicAuth(r.accessKey, r.secretKey)
	resp, _, errs := request.Get(r.url + "projects/" + r.projectId + "/workloads/deployment:" +
		r.namespaceId + ":" + id).End()
	if len(errs) > 0 {
		return false, errs[0]
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return false, errors.New("unexpected status " + strconv.Itoa(resp.StatusCode))
	}
	if resp.StatusCode == http.StatusNotFound {
		resp, _, errs = request.Get(r.url + "projects/" + r.projectId + "/workloads/job:" +
			r.namespaceId + ":" + id).End()
		if len(errs) > 0 {
			return false, errs[0]
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			return false, errors.New("unexpected status " + strconv.Itoa(resp.StatusCode))
		}
		return resp.StatusCode == http.StatusOK, nil
	}
	return true, nil
}
