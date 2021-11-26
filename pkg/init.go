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

package lib

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/api"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/controller"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/database/mongo"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/deploy"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/deploy/dockerClient"
	rancher1api "github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/deploy/rancher-api"
	rancher2api "github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/deploy/rancher2-api"
	"log"
	"sync"
)

func Start(conf config.Config, ctx context.Context) (wg *sync.WaitGroup, err error) {
	wg = &sync.WaitGroup{}

	data, err := mongo.New(conf, ctx, wg)
	if err != nil {
		return wg, err
	}

	var deploymentClient deploy.DeploymentClient

	switch conf.DeployMode {
	case "docker":
		deploymentClient, err = dockerClient.New(conf, ctx, wg)
		break
	case "rancher1":
		deploymentClient, err = rancher1api.New(conf)
		break
	case "rancher2":
		deploymentClient = rancher2api.New(conf)
		break
	default:
		return wg, errors.New("unknown deploy_mode")
	}
	if err != nil {
		return wg, err
	}

	ctrl := controller.New(conf, data, deploymentClient)

	if conf.StartupEnsureDeployed {
		log.Println("Restoring missing import containers")
		err = ctrl.EnsureAllInstancesDeployed()
		if err != nil {
			return wg, err
		}
	}

	err = api.Start(conf, ctx, ctrl)
	if err != nil {
		log.Println("ERROR: unable to start api", err)
		return wg, err
	}

	return wg, err
}
