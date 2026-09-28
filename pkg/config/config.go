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

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var LogEnvConfig = true

type Config struct {
	ApiPort                   string `json:"api_port"`
	KafkaBootstrap            string `json:"kafka_bootstrap"`
	MqttBroker                string `json:"mqtt_broker"`
	MqttUser                  string `json:"mqtt_user"`
	MqttPw                    string `json:"mqtt_pw" config:"secret"`
	MongoUrl                  string `json:"mongo_url"`
	MongoUser                 string `json:"mongo_user"`
	MongoPassword             string `json:"mongo_password" config:"secret"`
	MongoAuthSource           string `json:"mongo_auth_source"`
	MongoDatabase             string `json:"mongo_database"`
	MongoReplSet              bool   `json:"mongo_repl_set"`
	MongoImportTypeCollection string `json:"mongo_import_type_collection"`
	TransferImage             string `json:"transfer_image"`
	DeployMode                string `json:"deploy_mode"`
	DockerNetwork             string `json:"docker_network"`
	DockerPull                bool   `json:"docker_pull"`
	RancherUrl                string `json:"rancher_url"`
	RancherAccessKey          string `json:"rancher_access_key"`
	RancherSecretKey          string `json:"rancher_secret_key" config:"secret"`
	RancherStackId            string `json:"rancher_stack_id"`
	RancherNamespaceId        string `json:"rancher_namespace_id"`
	RancherProjectId          string `json:"rancher_project_id"`
	VerifyInput               bool   `json:"verify_input"`
	ImportDeployUrl           string `json:"import_deploy_url"`
	AnalyticsPipelineUrl      string `json:"analytics_pipeline_url"`
	StartupEnsureDeployed     bool   `json:"startup_ensure_deployed"`
	PermissionsV2Url          string `json:"permissions_v2_url"`

	Debug      bool   `json:"debug"`
	LogHandler string `json:"log_handler"`
}

// loads config from json in location and used environment variables (e.g ZookeeperUrl --> ZOOKEEPER_URL)
func Load(location string) (config Config, err error) {
	file, err := os.Open(location)
	if err != nil {
		return config, err
	}
	config = Config{
		MongoUrl:        "mongodb://localhost:27017",
		MongoAuthSource: "admin",
		MongoDatabase:   "kafka2mqtt_manager",
	}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return config, err
	}
	handleEnvironmentVars(&config)
	return config, nil
}

func isSecret(field reflect.StructField) bool {
	return strings.Contains(field.Tag.Get("config"), "secret")
}

// plainConfig has none of Config's methods, so formatting it does not recurse.
type plainConfig Config

// masked returns a copy in which every non-empty field tagged config:"secret" is replaced.
func (c Config) masked() plainConfig {
	v := reflect.ValueOf(&c).Elem()
	for i := 0; i < v.NumField(); i++ {
		if isSecret(v.Type().Field(i)) && v.Field(i).Kind() == reflect.String && v.Field(i).String() != "" {
			v.Field(i).SetString("***")
		}
	}
	return plainConfig(c)
}

func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.masked())
}

func (c Config) String() string {
	return fmt.Sprintf("%+v", c.masked())
}

func (c Config) GoString() string {
	return fmt.Sprintf("%#v", c.masked())
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func fieldNameToEnvName(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToUpper(strings.Join(a, "_"))
}

func handleEnvironmentVars(config *Config) {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	configType := configValue.Type()
	for index := 0; index < configType.NumField(); index++ {
		fieldName := configType.Field(index).Name
		envName := fieldNameToEnvName(fieldName)
		envValue := os.Getenv(envName)
		if envValue != "" {
			if LogEnvConfig && !isSecret(configType.Field(index)) {
				fmt.Println("use environment variable: ", envName, " = ", envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Int64 {
				i, _ := strconv.ParseInt(envValue, 10, 64)
				configValue.FieldByName(fieldName).SetInt(i)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.String {
				configValue.FieldByName(fieldName).SetString(envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Bool {
				b, _ := strconv.ParseBool(envValue)
				configValue.FieldByName(fieldName).SetBool(b)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Float64 {
				f, _ := strconv.ParseFloat(envValue, 64)
				configValue.FieldByName(fieldName).SetFloat(f)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Slice {
				val := []string{}
				for _, element := range strings.Split(envValue, ",") {
					val = append(val, strings.TrimSpace(element))
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(val))
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Map {
				value := map[string]string{}
				for _, element := range strings.Split(envValue, ",") {
					keyVal := strings.Split(element, ":")
					key := strings.TrimSpace(keyVal[0])
					val := strings.TrimSpace(keyVal[1])
					value[key] = val
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(value))
			}
		}
	}
}
