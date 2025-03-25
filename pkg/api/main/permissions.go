/*
 * Copyright 2025 InfAI (CC SES)
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

package main

import (
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/api"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/controller"
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
)

//go:generate go run permissions.go
//go:generate go tool swag init -o ../../../docs --parseDependency -d .. -g api.go

// generates lib/api/permissions.go
// which enables swag init to generate documentation for permissions endpoints
// which are added by 'permForward := client.New(config.PermissionsV2Url).EmbedPermissionsRequestForwarding("/permissions/", router)'
func main() {
	err := client.GenerateGoFileWithSwaggoCommentsForEmbeddedPermissionsClient(
		"api",
		"permissions",
		"../generated_permissions.go",
		[]string{controller.Permv2topic},
		api.ForwardPermissions,
	)
	if err != nil {
		panic(err)
	}
}
