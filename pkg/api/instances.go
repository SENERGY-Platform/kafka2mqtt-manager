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

package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/model"
	"github.com/julienschmidt/httprouter"
)

func init() {
	endpoints = append(endpoints, DeploymentEndpoints)
}

type instanceList struct {
	Instances model.Instances `json:"instances"`
	Count     int             `json:"count"`
	Total     int             `json:"total"`
}

const authHeader = "Authorization"

// Query godoc
// @Summary      Create an instance
// @Description  Creates an instance
// @Accept       json
// @Produce      json
// @Security Bearer
// @Param        instance body model.Instance true "Instance to create"
// @Success      200 {object}  model.Instance
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /instances [POST]
func PostInstances() {} // for doc generation

// Query godoc
// @Summary      Get instances
// @Description  Provides a list of instances
// @Produce      json
// @Security Bearer
// @Success      200 {array}  model.Instance
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /instances [GET]
func GetInstances() {} // for doc generation

// Query godoc
// @Summary      Get instance
// @Description  Provides a single instance
// @Produce      json
// @Security Bearer
// @Param        id path string true "ID of the requested instance"
// @Success      200 {object}  model.Instance
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /instances/{id} [GET]
func GetInstance() {} // for doc generation

// Query godoc
// @Summary      Update an instance
// @Description  Updates an instance
// @Accept       json
// @Produce      json
// @Security Bearer
// @Param        instance body model.Instance true "Instance to update"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /instances [PUT]
func PutInstances() {} // for doc generation

// Query godoc
// @Summary      Delete instance
// @Description  Deletes a single instance
// @Produce      json
// @Security Bearer
// @Param        id path string true "ID of the instance to delete"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /instances/{id} [DELETE]
func DeleteInstance() {} // for doc generation

// Query godoc
// @Summary      Delete instances
// @Description  Deletes a single instance
// @Produce      json
// @Security Bearer
// @Param        id body []string true "IDs of the instances to delete"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /instances [DELETE]
func DeleteInstances() {} // for doc generation

func DeploymentEndpoints(config config.Config, control Controller, router *httprouter.Router) {
	resource := "/instances"

	router.POST(resource, func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		instance := model.Instance{}
		err := json.NewDecoder(request.Body).Decode(&instance)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			log.Println("ERROR: unable to decode instance request: ", err)
			if config.Debug {
				b, _ := io.ReadAll(request.Body)
				log.Println("Payload: " + string(b))
			}
			return
		}
		result, err, code := control.CreateInstance(instance, getUserId(request), request.Header.Get(authHeader))
		if err != nil {
			http.Error(writer, err.Error(), code)
			log.Println("ERROR: cant create instance: ", err)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(code)
		err = json.NewEncoder(writer).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
			return
		}
		return
	})

	router.GET(resource, func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		limit := request.URL.Query().Get("limit")
		if limit == "" {
			limit = "100"
		}
		limitInt, err := strconv.ParseInt(limit, 10, 64)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		offset := request.URL.Query().Get("offset")
		if offset == "" {
			offset = "0"
		}
		offsetInt, err := strconv.ParseInt(offset, 10, 64)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		sort := request.URL.Query().Get("order")
		if sort == "" {
			sort = "name"
		}
		orderBy := strings.Split(sort, ":")[0]
		asc := !strings.HasSuffix(sort, ":desc")

		search := request.URL.Query().Get("search")

		includeGenerated := strings.ToLower(request.URL.Query().Get("generated")) != "false"
		results, total, err, errCode := control.ListInstances(request.Header.Get(authHeader), limitInt, offsetInt, orderBy, asc, search, includeGenerated)
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		r := instanceList{
			Instances: results,
			Count:     len(results),
			Total:     total,
		}
		if results == nil {
			r.Instances = []model.Instance{}
		}
		err = json.NewEncoder(writer).Encode(r)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
		return
	})

	router.GET(resource+"/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.ByName("id")
		result, err, errCode := control.ReadInstance(request.Header.Get(authHeader), id)
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
		return
	})

	router.DELETE(resource+"/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.ByName("id")
		err, errCode := control.DeleteInstances(request.Header.Get(authHeader), []string{id})
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		writer.WriteHeader(errCode)
		return
	})

	router.DELETE(resource, func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		var ids []string
		err := json.NewDecoder(request.Body).Decode(&ids)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		err, errCode := control.DeleteInstances(request.Header.Get(authHeader), ids)
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		writer.WriteHeader(errCode)
		return
	})

	router.PUT(resource+"/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.ByName("id")
		instance := model.Instance{}
		err := json.NewDecoder(request.Body).Decode(&instance)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		if id != instance.Id {
			http.Error(writer, "IDs don't match", http.StatusBadRequest)
			return
		}
		err, code := control.SetInstance(instance, getUserId(request), request.Header.Get(authHeader))
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
		return
	})

}

func getUserId(request *http.Request) string {
	user := request.Header.Get("X-UserId")
	if len(user) == 0 {
		log.Println("WARN: Could not extract UserId, replacing with 'developer'")
		user = "developer"
	}
	return user
}
