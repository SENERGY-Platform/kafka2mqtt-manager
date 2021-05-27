/*
 * Copyright 2021 InfAI (CC SES)
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

package model

import "time"

type Instances []Instance

type Instance struct {
	FilterType          string    `json:"FilterType,omitempty" validate:"required"`
	Filter              string    `json:"Filter,omitempty" validate:"required"`
	Name                string    `json:"Name,omitempty" validate:"required"`
	EntityName          string    `json:"EntityName,omitempty" validate:"required"`
	ServiceName         string    `json:"ServiceName,omitempty" validate:"required"`
	Description         string    `json:"Description,omitempty"`
	Topic               string    `json:"Topic,omitempty" validate:"required"`
	Generated           bool      `json:"generated,omitempty"`
	Offset              string    `json:"Offset,omitempty" validate:"required"`
	Values              []Value   `json:"Values,omitempty"`
	UserId              string    `json:"-"`
	ServiceId           string    `json:"-"`
	CustomMqttBroker    *string   `json:"CustomMqttBroker,omitempty"`
	CustomMqttUser      *string   `json:"CustomMqttUser,omitempty"`
	CustomMqttPassword  *string   `json:"CustomMqttPassword,omitempty"`
	CustomMqttBaseTopic *string   `json:"CustomMqttBaseTopic,omitempty"`
	Id                  string    `json:"ID"`
	CreatedAt           time.Time `json:"CreatedAt"`
	UpdatedAt           time.Time `json:"UpdatedAt"`
}

type InstancesResponse struct {
	Total     int64     `json:"total,omitempty"`
	Count     int       `json:"count,omitempty"`
	Instances Instances `json:"instances,omitempty"`
}

type Value struct {
	Name string `json:"Name"`
	Path string `json:"Path"`
}
