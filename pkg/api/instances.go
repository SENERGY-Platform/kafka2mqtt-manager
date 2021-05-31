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
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/model"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func init() {
	endpoints = append(endpoints, DeploymentEndpoints)
}

type instanceList struct {
	Instances model.Instances `json:"instances"`
	Count     int             `json:"count"`
	Total     int64           `json:"total"`
}

const authHeader = "Authorization"

func DeploymentEndpoints(_ config.Config, control Controller, router *httprouter.Router) {
	resource := "/instances"

	router.POST(resource, func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		instance := model.Instance{}
		err := json.NewDecoder(request.Body).Decode(&instance)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		result, err, code := control.CreateInstance(instance, getUserId(request), request.Header.Get(authHeader))
		if err != nil {
			http.Error(writer, err.Error(), code)
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
		results, total, err, errCode := control.ListInstances(getUserId(request), limitInt, offsetInt, orderBy, asc, search, includeGenerated)
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
		result, err, errCode := control.ReadInstance(id, getUserId(request))
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
		err, errCode := control.DeleteInstances([]string{id}, getUserId(request))
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
		err, errCode := control.DeleteInstances(ids, getUserId(request))
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
