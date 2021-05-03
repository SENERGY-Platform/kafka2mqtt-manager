/*
 * Copyright 2020 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/docker/docker/api/types"
)

func (this *DockerClient) listAllContainers() (containers []types.Container, err error) {
	return this.cli.ContainerList(context.Background(), types.ContainerListOptions{})
}

func (this *DockerClient) stopAllContainers() (err error) {
	ctx := context.Background()

	containers, err := this.listAllContainers()
	if err != nil {
		return err
	}

	for _, ct := range containers {
		if err = this.cli.ContainerStop(ctx, ct.ID, nil); err != nil {
			return err
		}
	}
	return nil
}

func (this *DockerClient) removeAllContainers() (err error) {
	ctx := context.Background()
	containers, err := this.listAllContainers()
	if err != nil {
		return err
	}

	removeOptions := types.ContainerRemoveOptions{Force: true}

	for _, ct := range containers {
		if err = this.cli.ContainerRemove(ctx, ct.ID, removeOptions); err != nil {
			return err
		}
	}
	return nil
}

func (this *DockerClient) stopContainer(id string) (err error) {
	ctx := context.Background()
	err = this.cli.ContainerStop(ctx, id, nil)
	return err
}

func (this *DockerClient) removeContainer(id string) (err error) {
	ctx := context.Background()
	removeOptions := types.ContainerRemoveOptions{Force: true}

	return this.cli.ContainerRemove(ctx, id, removeOptions)

}
