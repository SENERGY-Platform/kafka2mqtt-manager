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

package database

import (
	"context"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/model"
)

type Database interface {
	ListInstances(ctx context.Context, limit int64, offset int64, sort string, owner string, asc bool, search string, includeGenerated bool) (result []model.Instance, total int64, err error)
	GetInstance(ctx context.Context, id string, owner string) (instance model.Instance, exists bool, err error)
	SetInstance(ctx context.Context, instance model.Instance, owner string) error
	GetInstances(ctx context.Context, ids []string, owner string) (result []model.Instance, allExist bool, err error)
	RemoveInstances(ctx context.Context, ids []string, owner string) error
}
