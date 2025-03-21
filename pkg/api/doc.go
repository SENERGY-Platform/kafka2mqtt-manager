/*
 *    Copyright 2020 InfAI (CC SES)
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package api

import (
	"net/http"
	"strings"

	_ "github.com/SENERGY-Platform/kafka2mqtt-manager/docs"

	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"

	"github.com/julienschmidt/httprouter"
	"github.com/swaggo/swag"
)

func init() {
	endpoints = append(endpoints, DocEndpoint)
}

//go:generate go tool swag init -o ../../docs --parseDependency -d .. -g api/api.go
func DocEndpoint(config config.Config, control Controller, router *httprouter.Router) {
	router.GET("/doc", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		doc, err := swag.ReadDoc()
		if err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		//remove empty host to enable developer-swagger-api service to replace it; can not use cleaner delete on json object, because developer-swagger-api is sensible to formatting; better alternative is refactoring of developer-swagger-api/apis/db/db.py
		doc = strings.Replace(doc, `"host": "",`, "", 1)
		_, _ = writer.Write([]byte(doc))
	})
}
