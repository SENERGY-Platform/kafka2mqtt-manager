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

package verification

import (
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"io"
	"log"
	"net/http"
	"strconv"
)

type pipeline struct {
	Id string `json:"id"`
}

func (*Verifier) VerifyPipeline(id string, token string, userId string, config *config.Config) (bool, error) {
	req, err := http.NewRequest("GET", config.AnalyticsPipelineUrl+"/pipeline/"+id, nil)
	if err != nil {
		log.Println("Cant create request to get pipeline: ", err.Error())
		return false, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("X-UserId", userId)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Cant get pipeline information: ", err.Error())
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		log.Println("Cant get pipeline information, code: " + strconv.Itoa(resp.StatusCode) + " response: " + string(payload))
		return false, nil
	}
	return true, nil
}
