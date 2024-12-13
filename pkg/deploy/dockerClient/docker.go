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

package dockerClient

import (
	"context"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	"log"
	"sync"
)

type DockerClient struct {
	config config.Config
	cli    *docker.Client
}

func New(config config.Config, ctx context.Context, wg *sync.WaitGroup) (client *DockerClient, err error) {
	cli, err := docker.NewEnvClient()
	if err != nil {
		return nil, err
	}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		_ = cli.Close()
		wg.Done()
	}()
	return &DockerClient{config: config, cli: cli}, nil
}

func (this *DockerClient) CreateContainer(name string, image string, env map[string]string, restart bool) (id string, err error) {
	ctx, _ := util.GetTimeoutContext()
	if this.config.DockerPull == true {
		_, err = this.cli.ImagePull(ctx, image, types.ImagePullOptions{})
		if err != nil {
			log.Println("Cant pull image: " + err.Error())
			return id, err
		}
	}
	dockerEnv := []string{}
	for k, v := range env {
		dockerEnv = append(dockerEnv, k+"="+v)
	}
	var restartPolicy container.RestartPolicy
	if restart {
		restartPolicy = container.RestartPolicy{Name: "always"}
	} else {
		restartPolicy = container.RestartPolicy{Name: "no"}
	}
	resp, err := this.cli.ContainerCreate(ctx, &container.Config{
		Image: image,
		Env:   dockerEnv,
	}, &container.HostConfig{
		NetworkMode:   container.NetworkMode(this.config.DockerNetwork),
		RestartPolicy: restartPolicy,
	}, nil, nil, name)
	if err != nil {
		log.Println("Cant create container: " + err.Error())
		return id, err
	}

	err = this.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Println("Cant start container: " + err.Error())
		return id, err
	}
	return resp.ID, err
}

func (this *DockerClient) UpdateContainer(id string, name string, image string, env map[string]string, restart bool) (newId string, err error) {
	err = this.RemoveContainer(id)
	if err != nil {
		return newId, err
	}
	return this.CreateContainer(name, image, env, restart)
}

func (this *DockerClient) RemoveContainer(id string) (err error) {
	err = this.stopContainer(id)
	if err != nil {
		return err
	}
	err = this.removeContainer(id)
	if err != nil {
		return err
	}
	return nil
}

func (this *DockerClient) ContainerExists(id string) (exists bool, err error) {
	ctx, _ := util.GetTimeoutContext()
	_, err = this.cli.ContainerInspect(ctx, id)
	if err != nil {
		if docker.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
