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

package rancher_api

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"github.com/hashicorp/go-uuid"
	"github.com/parnurzeal/gorequest"
	"log"
	"net/http"
	"strconv"
)

type Rancher struct {
	url       string
	accessKey string
	secretKey string
	stackId   string
}

func New(config config.Config) (*Rancher, error) {
	r := &Rancher{config.RancherUrl, config.RancherAccessKey, config.RancherSecretKey, config.RancherStackId}
	err := r.selfCheck()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r Rancher) CreateContainer(name string, image string, env map[string]string, restart bool) (id string, err error) {
	id, err, _ = r.createContainer(name, image, env, restart)
	return id, err
}

func (r Rancher) createContainer(name string, image string, env map[string]string, restart bool) (id string, err error, code int) {
	labels := map[string]string{
		"io.rancher.container.pull_image":          "always",
		"io.rancher.scheduler.affinity:host_label": "role=worker",
	}
	if !restart {
		labels["io.rancher.container.start_once"] = "true"
	}

	request := gorequest.New().SetBasicAuth(r.accessKey, r.secretKey)

	reqBody := &Request{
		Type:          "service",
		Name:          name,
		StackId:       r.stackId,
		Scale:         1,
		StartOnCreate: true,
		LaunchConfig: LaunchConfig{
			ImageUuid:   "docker:" + image,
			Environment: env,
			Labels:      labels,
		},
	}

	resp, body, e := request.Post(r.url + "services").Send(reqBody).End()
	code = resp.StatusCode
	if resp.StatusCode != http.StatusCreated {
		err = errors.New("could not create instance")
		log.Println("ERROR: Rancher response code", resp.StatusCode, "when creating container, Body:", body)
		return
	}
	if len(e) > 0 {
		err = errors.New("could not create instance")
		for i := range e {
			log.Println("ERROR: Rancher create error", e[i].Error())
		}
		return
	}

	data := map[string]interface{}{}
	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		return
	}
	id, ok := data["id"].(string)
	if !ok {
		return id, errors.New("could not get service id"), code
	}
	return
}

func (r Rancher) RemoveContainer(id string) (err error) {
	request := gorequest.New().SetBasicAuth(r.accessKey, r.secretKey)
	resp, body, e := request.Delete(r.url + "services/" + id).End()
	if len(e) > 0 {
		err = errors.New("could not delete instance: " + body)
		return
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected status code while removing container " + strconv.Itoa(resp.StatusCode))
	}
	return
}

func (r Rancher) UpdateContainer(id string, name string, image string, env map[string]string, restart bool) (newId string, err error) {
	err = r.RemoveContainer(id)
	if err != nil {
		return newId, err
	}
	for {
		bytes, err := uuid.GenerateRandomBytes(64)
		if err != nil {
			return newId, err
		}
		rand := binary.BigEndian.Uint64(bytes)
		newId, err, code := r.createContainer(name+"-"+strconv.FormatUint(rand, 16), image, env, restart)
		if err != nil {
			return newId, err
		}
		if code != http.StatusUnprocessableEntity {
			// if  code == http.StatusUnprocessableEntity probably reuse of old id
			return newId, err
		}
	}
}

func (r Rancher) ContainerExists(id string) (exists bool, err error) {
	request := gorequest.New().SetBasicAuth(r.accessKey, r.secretKey)
	resp, _, errs := request.Get(r.url + "services/" + id).End()
	if len(errs) > 0 {
		return false, errs[0]
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return false, errors.New("unexpected status " + strconv.Itoa(resp.StatusCode))
	}
	return resp.StatusCode == http.StatusOK, nil
}

func (r Rancher) selfCheck() error {
	request := gorequest.New().SetBasicAuth(r.accessKey, r.secretKey)
	resp, _, _ := request.Get(r.url + "stacks/" + r.stackId).End()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return errors.New("rancher unexpected status code: " + strconv.Itoa(resp.StatusCode))
}
