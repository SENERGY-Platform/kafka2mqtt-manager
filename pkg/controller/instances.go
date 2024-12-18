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

package controller

import (
	"errors"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/model"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/util"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/verification"
	"github.com/hashicorp/go-uuid"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const idPrefix = "urn:infai:ses:broker-export:"
const containerNamePrefix = "k2m-"
const filterDevice = "deviceId"
const filterImport = "import_id"
const filterOperator = "operatorId"

func (this *Controller) ListInstances(userId string, limit int64, offset int64, sort string, asc bool, search string, includeGenerated bool) (results []model.Instance, total int64, err error, errCode int) {
	ctx, _ := getTimeoutContext()
	results, total, err = this.db.ListInstances(ctx, limit, offset, sort, userId, asc, search, includeGenerated)
	if err != nil {
		return results, 0, err, http.StatusInternalServerError
	}
	return results, total, nil, http.StatusOK
}

func (this *Controller) ReadInstance(id string, userId string) (result model.Instance, err error, errCode int) {
	ctx, _ := getTimeoutContext()
	result, exists, err := this.db.GetInstance(ctx, id, userId)
	if !exists {
		return result, err, http.StatusNotFound
	}
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return result, nil, http.StatusOK
}

func (this *Controller) CreateInstance(instance model.Instance, userId string, token string) (result model.Instance, err error, code int) {
	if instance.Id != "" {
		return result, errors.New("explicit setting of id not allowed"), http.StatusBadRequest
	}
	id, err := uuid.GenerateUUID()
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	instance.Id = idPrefix + id
	instance.UserId = userId

	env, err, code := this.getEnv(&instance, token, userId, true)
	if err != nil {
		log.Println("Cant get env: " + err.Error())
		return result, err, code
	}

	instance.ServiceId, err = this.deploymentClient.CreateContainer(containerNamePrefix+strings.TrimPrefix(instance.Id, idPrefix), this.config.TransferImage, env, true)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	now := time.Now()
	instance.CreatedAt = now
	instance.UpdatedAt = now
	ctx, _ := getTimeoutContext()
	err = this.db.SetInstance(ctx, instance, userId)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return instance, nil, http.StatusOK
}

func (this *Controller) SetInstance(instance model.Instance, userId string, token string) (err error, code int) {
	ctx, _ := getTimeoutContext()
	existing, exists, err := this.db.GetInstance(ctx, instance.Id, userId)
	if !exists {
		return errors.New("not found"), http.StatusNotFound
	}
	if err != nil {
		return err, http.StatusInternalServerError
	}
	instance.UserId = userId

	env, err, code := this.getEnv(&instance, token, userId, true)
	if err != nil {
		return err, code
	}

	if (existing.Offset != instance.Offset) || (existing.Offset == "smallest" && !reflect.DeepEqual(existing.Values, instance.Values)) {
		err = refreshConsumerGroupId(instance, env)
		if err != nil {
			return err, http.StatusInternalServerError
		}
	}

	instance.ServiceId, err = this.deploymentClient.UpdateContainer(existing.ServiceId, containerNamePrefix+strings.TrimPrefix(instance.Id, idPrefix), this.config.TransferImage, env, true)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	instance.UpdatedAt = time.Now()
	ctx, _ = getTimeoutContext()
	err = this.db.SetInstance(ctx, instance, userId)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Controller) DeleteInstances(ids []string, userId string) (err error, errCode int) {
	ctx, _ := getTimeoutContext()
	instances, exists, err := this.db.GetInstances(ctx, ids, userId)
	if !exists {
		return errors.New("not found"), http.StatusNotFound
	}
	if err != nil {
		return err, http.StatusInternalServerError
	}
	for i := range instances {
		err = this.deploymentClient.RemoveContainer(instances[i].ServiceId)
		if err != nil {
			return err, http.StatusInternalServerError
		}
		ctx, _ := getTimeoutContext()
		err = this.db.RemoveInstances(ctx, []string{instances[i].Id}, userId)
		if err != nil {
			return err, http.StatusInternalServerError
		}
	}

	return nil, http.StatusNoContent
}

func (this *Controller) EnsureAllInstancesDeployed() (err error) {
	var offset int64 = 0
	var batchSize int64 = 100
	for {
		ctx, _ := util.GetTimeoutContext()
		instances, _, err := this.db.ListInstances(ctx, batchSize, offset, "name", "", true, "", true)
		if err != nil {
			return err
		}
		if len(instances) == 0 {
			return nil // done
		}
		offset += int64(len(instances))
		for _, instance := range instances {
			exists, err := this.deploymentClient.ContainerExists(instance.ServiceId)
			if err != nil {
				return err
			}
			if exists {
				log.Println(instance.Id + " still exists")
				continue
			}
			log.Println("Recreating " + instance.Id)
			env, err, _ := this.getEnv(&instance, "", instance.UserId, false)
			if err != nil {
				return err
			}
			instance.ServiceId, err = this.deploymentClient.CreateContainer(containerNamePrefix+strings.TrimPrefix(instance.Id, idPrefix), this.config.TransferImage, env, true)
			if err != nil {
				return err
			}
			ctx, _ := util.GetTimeoutContext()
			err = this.db.SetInstance(ctx, instance, instance.UserId)
			if err != nil {
				return err
			}
		}
	}
}

func (this *Controller) getEnv(instance *model.Instance, token string, userId string, verify bool) (m map[string]string, err error, code int) {
	m = map[string]string{}
	m["KAFKA_BOOTSTRAP"] = this.config.KafkaBootstrap
	m["KAFKA_TOPIC"] = instance.Topic
	m["KAFKA_GROUP_ID"] = instance.Id
	m["KAFKA_OFFSET"] = instance.Offset
	m["FILTER_QUERY"] = "."
	switch instance.FilterType {
	case filterDevice:
		if this.config.VerifyInput && verify {
			ok, err := verification.VerifyDevice(instance.Filter, token, &this.config)
			if err != nil {
				return nil, err, http.StatusInternalServerError
			}
			if !ok {
				return nil, errors.New("filtered device not found"), http.StatusNotFound
			}
		}
		m["FILTER_QUERY"] += "device_id==\"" + instance.Filter + "\""
	case filterOperator:
		parts := strings.Split(instance.Filter, ":")
		if len(parts) != 2 {
			return m, errors.New("filterType is operatorId, but filter has not exactly two parts"), http.StatusBadRequest
		}
		if this.config.VerifyInput && verify {
			ok, err := verification.VerifyPipeline(parts[0], token, userId, &this.config)
			if err != nil {
				return nil, err, http.StatusInternalServerError
			}
			if !ok {
				return nil, errors.New("filtered pipeline not found"), http.StatusNotFound
			}
		}
		m["FILTER_QUERY"] += "pipeline_id==\"" + parts[0] + "\"and.operator_id==\"" + parts[1] + "\""
	case filterImport:
		if this.config.VerifyInput && verify {
			ok, err := verification.VerifyImport(instance.Filter, token, userId, &this.config)
			if err != nil {
				return nil, err, http.StatusInternalServerError
			}
			if !ok {
				return nil, errors.New("filtered import not found"), http.StatusNotFound
			}
		}
		m["FILTER_QUERY"] += "import_id==\"" + instance.Filter + "\""
	default:
		return m, errors.New("unknown filterType"), http.StatusBadRequest
	}
	baseTopic := "export/" + instance.UserId + "/" + instance.Id + "/"
	if instance.CustomMqttBroker != nil {
		m["MQTT_BROKER"] = *instance.CustomMqttBroker
		if instance.CustomMqttUser != nil {
			m["MQTT_USER"] = *instance.CustomMqttUser
		}
		if instance.CustomMqttPassword != nil {
			m["MQTT_PW"] = *instance.CustomMqttPassword
		}
		if instance.CustomMqttBaseTopic != nil {
			baseTopic = *instance.CustomMqttBaseTopic
			if !strings.HasSuffix(baseTopic, "/") && len(baseTopic) > 0 {
				baseTopic += "/"
				instance.CustomMqttBaseTopic = &baseTopic
				log.Println("baseTopic", baseTopic, *instance.CustomMqttBaseTopic)
			}
		}
	} else {
		if instance.CustomMqttUser != nil || instance.CustomMqttPassword != nil || instance.CustomMqttBaseTopic != nil {
			return nil, errors.New("must not set custom mqtt options with default broker"), http.StatusBadRequest
		}
		m["MQTT_BROKER"] = this.config.MqttBroker
		m["MQTT_USER"] = this.config.MqttUser
		m["MQTT_PW"] = this.config.MqttPw
	}
	m["MQTT_CLIENT_ID"] = instance.Id
	m["MQTT_QOS"] = "1"
	m["MQTT_TOPIC_MAPPING"] = "["
	for i := range instance.Values {
		m["MQTT_TOPIC_MAPPING"] += "{\"query\":\"." + instance.Values[i].Path + "\",\"topic\":\"" + baseTopic + instance.Values[i].Name + "\"}"
		if i < len(instance.Values)-1 {
			m["MQTT_TOPIC_MAPPING"] += ","
		}
	}
	m["MQTT_TOPIC_MAPPING"] += "]"
	m["DEBUG"] = "true"

	return m, nil, http.StatusOK
}

func refreshConsumerGroupId(instance model.Instance, env map[string]string) error {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return err
	}
	env["KAFKA_GROUP_ID"] = instance.Id + "_" + id
	return nil
}
