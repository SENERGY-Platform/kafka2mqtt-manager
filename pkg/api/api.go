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
	"context"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	gin_mw "github.com/SENERGY-Platform/gin-middleware"
	"github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/model"

	_log "github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/log"
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

var endpoints []func(config config.Config, control Controller, router *gin.Engine)

func ForwardPermissions(method string, path string) bool {
	if method == http.MethodDelete {
		return false
	}
	if strings.Contains(path, "import") {
		return false
	}
	if strings.Contains(path, "export") {
		return false
	}
	if strings.Contains(path, "admin") {
		return false
	}
	return true
}

func Start(config config.Config, ctx context.Context, control Controller, permv2 client.Client) (err error) {
	_log.Logger.Info("start api", "port", config.ApiPort)
	router := Router(config, control)
	router = client.EmbedPermissionsClientIntoRouter(permv2, router, "/permissions/", ForwardPermissions)
	server := &http.Server{Addr: ":" + config.ApiPort, Handler: router, WriteTimeout: 10 * time.Second, ReadTimeout: 2 * time.Second, ReadHeaderTimeout: 2 * time.Second}
	go func() {
		_log.Logger.Info("api listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			_log.Logger.Error("api server error", attributes.ErrorKey, err)
			panic(err)
		}
	}()
	go func() {
		<-ctx.Done()
		err = server.Shutdown(context.Background())
		if config.Debug {
			if err != nil {
				_log.Logger.Debug("api shutdown", attributes.ErrorKey, err)
			} else {
				_log.Logger.Debug("api shutdown")
			}
		}
	}()
	return nil
}

// GetRouter doc
// @title         Kafka2MQTT API
// @version       0.1
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath  /
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func Router(config config.Config, control Controller) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		gin_mw.StructLoggerHandlerWithDefaultGenerators(
			_log.Logger.With(attributes.LogRecordTypeKey, attributes.HttpAccessLogRecordTypeVal),
			attributes.Provider,
			[]string{},
			nil,
		),
		requestid.New(requestid.WithCustomHeaderStrKey("X-Request-ID")),
		gin_mw.ErrorHandler(model.GetStatusCode, ", "),
		gin_mw.StructRecoveryHandler(_log.Logger, gin_mw.DefaultRecoveryFunc),
	)
	for _, e := range endpoints {
		_log.Logger.Info("add endpoint", "name", runtime.FuncForPC(reflect.ValueOf(e).Pointer()).Name())
		e(config, control, router)
	}
	router.GET("/", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})
	return router
}
