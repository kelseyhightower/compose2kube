/*
Copyright 2015 German Ramos. All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

func parseRancherCompose() map[interface{}]interface{} {
	composeFile := composeFilePath + "rancher-compose.yml"
	file, err := ioutil.ReadFile(composeFile)
	if err != nil {
		log.Printf("error: %v", err)
		return nil
	}
	var f interface{}
	err = yaml.Unmarshal(file, &f)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return f.(map[interface{}]interface{})
}

func configureScale(name string, rancherCompose map[interface{}]interface{}) int {
	if rancherCompose[name] != nil && rancherCompose[name].(map[interface{}]interface{})["scale"] != nil {
		return rancherCompose[name].(map[interface{}]interface{})["scale"].(int)
	}
	return 1
}

func processRancherCompose(rancherCompose map[interface{}]interface{}) {
	if len(rancherCompose) == 0 {
		return
	}

	catalog := rancherCompose[".catalog"].(map[interface{}]interface{})
	var questions []interface{}
	if catalog["questions"] != nil {
		questions = catalog["questions"].([]interface{})
	}

	firstQuestion := make(map[string]interface{})
	firstQuestion["variable"] = "NAME"
	firstQuestion["default"] = catalog["name"].(string)
	firstQuestion["label"] = "Kubernetes Name (Max 24 characters)"
	firstQuestion["description"] = "at most 24 characters] = matching regex [a-z]([-a-z0-9]*[a-z0-9])?)"
	firstQuestion["required"] = true
	firstQuestion["type"] = "string"

	secondQuestion := make(map[string]interface{})
	secondQuestion["variable"] = "NAMESPACE"
	secondQuestion["default"] = "default"
	secondQuestion["label"] = "Kubernetes Namespace"
	secondQuestion["description"] = "Make sure the Namespace exists or you will not be able to see the service"
	secondQuestion["required"] = true
	secondQuestion["type"] = "string"

	newQuestionsArray := make([]interface{}, 2)
	newQuestionsArray[0] = firstQuestion
	newQuestionsArray[1] = secondQuestion

	newQuestions := Append(newQuestionsArray, questions...) // The '...' is essential!
	catalog["questions"] = newQuestions

	byteArray, _ := yaml.Marshal(rancherCompose)

	outputFilePath := filepath.Join(outputDir, "rancher-compose.yml")
	if err := ioutil.WriteFile(outputFilePath, byteArray, 0644); err != nil {
		log.Fatalf("Failed to write rancher-compose: %v", err)
	}
	fmt.Println(outputFilePath)
}
