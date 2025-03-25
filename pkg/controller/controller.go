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

package controller

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/verification"
	permv2 "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
)

type Controller struct {
	db               Database
	deploymentClient DeploymentClient
	config           config.Config
	verifier         *verification.Verifier
	permv2           permv2.Client
}

const Permv2topic = "kafka2mqtt"

func New(config config.Config, db Database, deploymentClient DeploymentClient, verifier *verification.Verifier, permv2 permv2.Client) (*Controller, error) {
	controller := &Controller{
		db:               db,
		deploymentClient: deploymentClient,
		config:           config,
		verifier:         verifier,
		permv2:           permv2,
	}

	err := controller.migrate()
	if err != nil {
		return nil, err
	}
	return controller, nil
}

func (c *Controller) migrate() error {
	_, err, _ := c.permv2.SetTopic(permv2.InternalAdminToken, permv2.Topic{
		Id: Permv2topic,
		DefaultPermissions: model.ResourcePermissions{
			RolePermissions: map[string]model.PermissionsMap{
				"admin": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	var offset int64 = 0
	var batchSize int64 = 100
	dbInstanceIds := []string{}
	for {
		ctx, _ := getTimeoutContext()
		instances, err := c.db.ListInstances(ctx, batchSize, offset, "name", true, "", true, nil)
		if err != nil {
			return err
		}
		offset += int64(len(instances))
		for _, instance := range instances {
			dbInstanceIds = append(dbInstanceIds, instance.Id)
			_, err, code := c.permv2.GetResource(permv2.InternalAdminToken, Permv2topic, instance.Id)
			if err != nil {
				if code == http.StatusNotFound {
					_, err, _ = c.permv2.SetPermission(permv2.InternalAdminToken, Permv2topic, instance.Id, model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							instance.UserId: {
								Read:         true,
								Write:        true,
								Execute:      true,
								Administrate: true,
							},
						},
						RolePermissions: map[string]model.PermissionsMap{
							"admin": {
								Read:         true,
								Write:        true,
								Execute:      true,
								Administrate: true,
							},
						},
					})
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}
		}
		if len(instances) < int(batchSize) {
			break // done
		}
	}
	permv2Ids, err, _ := c.permv2.AdminListResourceIds(permv2.InternalAdminToken, Permv2topic, model.ListOptions{})
	if err != nil {
		return err
	}
	slices.Sort(dbInstanceIds) // enables binary search
	for _, permv2Id := range permv2Ids {
		_, ok := slices.BinarySearch(dbInstanceIds, permv2Id)
		if !ok {
			err, _ = c.permv2.RemoveResource(permv2.InternalAdminToken, Permv2topic, permv2Id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}
