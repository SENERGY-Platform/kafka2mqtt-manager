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
	"log"
	"net/http"
	"reflect"
	"runtime"
	"time"

	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/api/util"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/julienschmidt/httprouter"
)

var endpoints []func(config config.Config, control Controller, router *httprouter.Router)

func Start(config config.Config, ctx context.Context, control Controller, permv2 client.Client) (err error) {
	log.Println("start api on " + config.ApiPort)
	router := Router(config, control)
	handler := util.NewLogger(util.NewCors(router))
	client.EmbedPermissionsClientIntoRouter(permv2, handler, "/permissions/")
	server := &http.Server{Addr: ":" + config.ApiPort, Handler: handler, WriteTimeout: 10 * time.Second, ReadTimeout: 2 * time.Second, ReadHeaderTimeout: 2 * time.Second}
	go func() {
		log.Println("listening on ", server.Addr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("ERROR: api server error", err)
		}
	}()
	go func() {
		<-ctx.Done()
		err = server.Shutdown(context.Background())
		if config.Debug {
			log.Println("DEBUG: api shutdown", err)
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
	router := httprouter.New()
	log.Println("add heart beat endpoint")
	router.GET("/", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		writer.WriteHeader(http.StatusOK)
	})
	for _, e := range endpoints {
		log.Println("add endpoints: " + runtime.FuncForPC(reflect.ValueOf(e).Pointer()).Name())
		e(config, control, router)
	}
	return router
}
