/*
Copyright 2015 Kelsey Hightower All rights reserved.
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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/docker/libcompose/config"
	"github.com/docker/libcompose/project"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

var (
	composeFilePath string
	outputDir       string
	asYml           bool
)

func init() {
	flag.StringVar(&composeFilePath, "compose-file-path", "./", "Specify an alternate path for compose files")
	flag.StringVar(&outputDir, "output-dir", "output", "Kubernetes configs output `directory`")
	flag.BoolVar(&asYml, "yaml", false, "output yaml instead of json")
}

func parseDockerCompose() *project.Project {
	composeFile := composeFilePath + "docker-compose.yml"
	p := project.NewProject(&project.Context{
		ProjectName:  "kube",
		ComposeFiles: []string{composeFile},
	}, nil, &config.ParseOptions{})

	if err := p.Parse(); err != nil {
		log.Fatalf("Failed to parse the compose project from %s: %v", composeFile, err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create the output directory %s: %v", outputDir, err)
	}

	if p.ServiceConfigs == nil {
		log.Fatalf("No service config found, aborting")
	}
	return p
}

func createReplicationController(shortName string, service *config.ServiceConfig, scale int) *api.ReplicationController {
	rc := &api.ReplicationController{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "ReplicationController",
			APIVersion: "v1",
		},
		ObjectMeta: api.ObjectMeta{
			Name:   shortName,
			Labels: map[string]string{"service": shortName},
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: int32(scale),
			Selector: map[string]string{"service": shortName},
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{"service": shortName},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:    shortName,
							Image:   service.Image,
							Command: service.Command,
						},
					},
				},
			},
		},
	}
	return rc
}

func configureVariables(service *config.ServiceConfig) []api.EnvVar {
	// Configure the container ENV variables
	var envs []api.EnvVar
	for _, env := range service.Environment {
		if strings.Contains(env, "=") {
			parts := strings.Split(env, "=")
			ename := parts[0]
			evalue := parts[1]
			envs = append(envs, api.EnvVar{Name: ename, Value: evalue})
		}
	}
	return envs
}

func configurePorts(name string, service *config.ServiceConfig) []api.ContainerPort {
	var ports []api.ContainerPort
	for _, port := range service.Ports {
		// Check if we have to deal with a mapped port
		port = strings.Trim(port, "\"")
		port = strings.TrimSpace(port)
		if strings.Contains(port, ":") {
			parts := strings.Split(port, ":")
			port = parts[1]
		}
		portNumber, err := strconv.ParseInt(port, 10, 32)
		if err != nil {
			log.Fatalf("Invalid container port %s for service %s", port, name)
		}
		ports = append(ports, api.ContainerPort{ContainerPort: int32(portNumber)})
	}
	return ports
}

// func configureVolumes(service *config.ServiceConfig) ([]api.VolumeMount, []api.Volume) {
// 	var volumemounts []api.VolumeMount
// 	var volumes []api.Volume
// 	for _, volumestr := range service.Volumes {
// 		parts := strings.Split(volumestr, ":")
// 		partHostDir := parts[0]
// 		partContainerDir := parts[1]
// 		partReadOnly := false
// 		if len(parts) > 2 {
// 			for _, partOpt := range parts[2:] {
// 				switch partOpt {
// 				case "ro":
// 					partReadOnly = true
// 					break
// 				case "rw":
// 					partReadOnly = false
// 					break
// 				}
// 			}
// 		}
// 		partName := strings.Replace(partHostDir, "/", "", -1)
// 		if len(parts) > 2 {
// 			volumemounts = append(volumemounts, api.VolumeMount{Name: partName, ReadOnly: partReadOnly, MountPath: partContainerDir})
// 		} else {
// 			volumemounts = append(volumemounts, api.VolumeMount{Name: partName, ReadOnly: partReadOnly, MountPath: partContainerDir})
// 		}
// 		source := &api.HostPathVolumeSource{
// 			Path: partHostDir,
// 		}
// 		vsource := api.VolumeSource{HostPath: source}
// 		volumes = append(volumes, api.Volume{Name: partName, VolumeSource: vsource})
// 	}
// 	return volumemounts, volumes
// }

func configureRestartPolicy(name string, service *config.ServiceConfig) api.RestartPolicy {
	restartPolicy := api.RestartPolicyAlways
	switch service.Restart {
	case "", "always":
		restartPolicy = api.RestartPolicyAlways
	case "no":
		restartPolicy = api.RestartPolicyNever
	case "on-failure":
		restartPolicy = api.RestartPolicyOnFailure
	default:
		log.Fatalf("Unknown restart policy %s for service %s", service.Restart, name)
	}
	return restartPolicy
}

func writeOuputFile(shortName string, rc *api.ReplicationController) {
	data, err := json.MarshalIndent(rc, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal the replication controller: %v", err)
	}
	if !asYml {
		// Save the replication controller for the Docker compose service to the
		// configs directory.
		outputFileName := fmt.Sprintf("%s-rc.json", shortName)
		outputFilePath := filepath.Join(outputDir, outputFileName)
		if err := ioutil.WriteFile(outputFilePath, data, 0644); err != nil {
			log.Fatalf("Failed to write replication controller %s: %v", outputFileName, err)
		}
		fmt.Println(outputFilePath)
	} else {
		// Save the replication controller to Yaml file
		var exp interface{}
		// because yaml is not directly usable from api, we can unmarshal json to interface{}
		// and then write yaml file
		// yaml segfaults on serializing rc directly
		json.Unmarshal(data, &exp)
		data, err = yaml.Marshal(exp)
		if err != nil {
			log.Fatalf("Failed to marshal the replication controller to yaml: %v", err)
		}
		// Save the replication controller for the Docker compose service to the
		// configs directory.
		outputFileName := fmt.Sprintf("%s-rc.yml", shortName)
		outputFilePath := filepath.Join(outputDir, outputFileName)
		if err := ioutil.WriteFile(outputFilePath, data, 0644); err != nil {
			log.Fatalf("Failed to write replication controller %s: %v", outputFileName, err)
		}
		fmt.Println(outputFilePath)
	}
}

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

func processDockerCompose(dockerCompose *project.Project, rancherCompose map[interface{}]interface{}) {
	for _, name := range dockerCompose.ServiceConfigs.Keys() {
		service, ok := dockerCompose.ServiceConfigs.Get(name)
		if !ok {
			log.Fatalf("Failed to get key %s from config", name)
		}

		shortName := name
		if len(name) > 24 {
			shortName = name[0:24]
		}
		scale := configureScale(name, rancherCompose)
		rc := createReplicationController(shortName, service, scale)
		rc.Spec.Template.Spec.Containers[0].Ports = configurePorts(name, service)
		rc.Spec.Template.Spec.Containers[0].Env = configureVariables(service)
		// rc.Spec.Template.Spec.Containers[0].VolumeMounts, rc.Spec.Template.Spec.Volumes = configureVolumes(service)
		rc.Spec.Template.Spec.RestartPolicy = configureRestartPolicy(name, service)
		writeOuputFile(shortName, rc)
	}
}

func getIndentator(line string) string {
	//result := ""
	var index int
	var value rune
	for index, value = range line {
		if value != 32 {
			break
		}
	}
	return line[0:index]
}

func processRancherCompose(rancherCompose map[interface{}]interface{}) {
	catalog := rancherCompose[".catalog"].(map[interface{}]interface{})
	var questions []interface{}
	if catalog["questions"] != nil {
		questions = catalog["questions"].([]interface{})
	}

	firstQuestion := make(map[string]string)
	firstQuestion["variable"] = "NAME"
	firstQuestion["default"] = catalog["name"].(string)
	firstQuestion["label"] = "Kubernetes Name (Max 24 characters)"
	firstQuestion["description"] = "at most 24 characters] = matching regex [a-z]([-a-z0-9]*[a-z0-9])?)"
	firstQuestion["required"] = "true"
	firstQuestion["type"] = "string"

	secondQuestion := make(map[string]string)
	secondQuestion["variable"] = "NAMESPACE"
	secondQuestion["default"] = "default"
	secondQuestion["label"] = "Kubernetes Namespace"
	secondQuestion["description"] = "Make sure the Namespace exists or you will not be able to see the service"
	secondQuestion["required"] = "true"
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

func main() {
	flag.Parse()
	dockerCompose := parseDockerCompose()
	rancherCompose := parseRancherCompose()
	processDockerCompose(dockerCompose, rancherCompose)
	processRancherCompose(rancherCompose)
}
