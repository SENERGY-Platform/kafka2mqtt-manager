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
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	_log "github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/log"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/model"
	"github.com/gin-gonic/gin"
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

func DeploymentEndpoints(config config.Config, control Controller, router *gin.Engine) {
	resource := "/instances"

	router.POST(resource, func(c *gin.Context) {
		instance := model.Instance{}
		err := c.ShouldBind(&instance)
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(http.StatusBadRequest)))
			_log.Logger.Error("unable to decode instance request", attributes.ErrorKey, err)
			return
		}
		result, err, code := control.CreateInstance(instance, getUserId(c), c.GetHeader(authHeader))
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(code)))
			_log.Logger.Error("can't create instance", attributes.ErrorKey, err)
			return
		}
		c.JSON(code, result)
	})

	router.GET(resource, func(c *gin.Context) {
		limit := c.Query("limit")
		if limit == "" {
			limit = "100"
		}
		limitInt, err := strconv.ParseInt(limit, 10, 64)
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(http.StatusBadRequest)))
			return
		}
		offset := c.Query("offset")
		if offset == "" {
			offset = "0"
		}
		offsetInt, err := strconv.ParseInt(offset, 10, 64)
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(http.StatusBadRequest)))
			return
		}
		sort := c.Query("order")
		if sort == "" {
			sort = "name"
		}
		orderBy := strings.Split(sort, ":")[0]
		asc := !strings.HasSuffix(sort, ":desc")

		search := c.Query("search")

		includeGenerated := strings.ToLower(c.Query("generated")) != "false"
		results, total, err, errCode := control.ListInstances(c.GetHeader(authHeader), limitInt, offsetInt, orderBy, asc, search, includeGenerated)
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(errCode)))
			return
		}
		r := instanceList{
			Instances: results,
			Count:     len(results),
			Total:     total,
		}
		if results == nil {
			r.Instances = []model.Instance{}
		}
		c.JSON(http.StatusOK, r)
	})

	router.GET(resource+"/:id", func(c *gin.Context) {
		id := c.Param("id")
		result, err, errCode := control.ReadInstance(c.GetHeader(authHeader), id)
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(errCode)))
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.DELETE(resource+"/:id", func(c *gin.Context) {
		id := c.Param("id")
		err, errCode := control.DeleteInstances(c.GetHeader(authHeader), []string{id})
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(errCode)))
			return
		}
		c.Status(errCode)
	})

	router.DELETE(resource, func(c *gin.Context) {
		var ids []string
		err := c.ShouldBind(&ids)
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(http.StatusBadRequest)))
			return
		}
		err, errCode := control.DeleteInstances(c.GetHeader(authHeader), ids)
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(errCode)))
			return
		}
		c.Status(errCode)
	})

	router.PUT(resource+"/:id", func(c *gin.Context) {
		id := c.Param("id")
		instance := model.Instance{}
		err := c.ShouldBind(&instance)
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(http.StatusBadRequest)))
			return
		}

		if id != instance.Id {
			_ = c.Error(errors.Join(errors.New("IDs don't match"), model.GetError(http.StatusBadRequest)))
			return
		}
		err, code := control.SetInstance(instance, getUserId(c), c.GetHeader(authHeader))
		if err != nil {
			_ = c.Error(errors.Join(err, model.GetError(code)))
			return
		}
		c.Status(http.StatusOK)
	})

}

func getUserId(c *gin.Context) string {
	user := c.GetHeader("X-UserId")
	if len(user) == 0 {
		_log.Logger.Warn("could not extract user id, replacing with developer")
		user = "developer"
	}
	return user
}
