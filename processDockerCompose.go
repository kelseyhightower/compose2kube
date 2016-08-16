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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/libcompose/config"
	"github.com/docker/libcompose/project"
	"gopkg.in/yaml.v2"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

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
