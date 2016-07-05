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

	"github.com/docker/libcompose/config"
	"github.com/docker/libcompose/project"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

var (
	composeFile string
	outputDir   string
)

func init() {
	flag.StringVar(&composeFile, "compose-file", "docker-compose.yml", "Specify an alternate compose `file`")
	flag.StringVar(&outputDir, "output-dir", "output", "Kubernetes configs output `directory`")
}

func main() {
	flag.Parse()

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
	keys := p.ServiceConfigs.Keys()

	for _, name := range keys {
		service, ok := p.ServiceConfigs.Get(name)
		if !ok {
			log.Fatalf("Failed to get key %s from config", name)
		}

		rc := &api.ReplicationController{
			TypeMeta: unversioned.TypeMeta{
				Kind:       "ReplicationController",
				APIVersion: "v1",
			},
			ObjectMeta: api.ObjectMeta{
				Name:   name,
				Labels: map[string]string{"service": name},
			},
			Spec: api.ReplicationControllerSpec{
				Replicas: 1,
				Selector: map[string]string{"service": name},
				Template: &api.PodTemplateSpec{
					ObjectMeta: api.ObjectMeta{
						Labels: map[string]string{"service": name},
					},
					Spec: api.PodSpec{
						Containers: []api.Container{
							{
								Name:    name,
								Image:   service.Image,
								Command: service.Command,
							},
						},
					},
				},
			},
		}

		// Configure the container ports.
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
		rc.Spec.Template.Spec.Containers[0].Ports = ports

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
		rc.Spec.Template.Spec.Containers[0].Env = envs

		// Configure the volumes
		var volumemounts []api.VolumeMount
		var volumes []api.Volume
		for _, volumestr := range service.Volumes {
			parts := strings.Split(volumestr, ":")
			partHostDir := parts[0]
			partContainerDir := parts[1]
			partReadOnly := false
			if len(parts) > 2 {
				for _, partOpt := range parts[2:] {
					switch partOpt {
					case "ro":
						partReadOnly = true
						break
					case "rw":
						partReadOnly = false
						break
					}
				}
			}
			partName := strings.Replace(partHostDir, "/", "", -1)
			if len(parts) > 2 {
				volumemounts = append(volumemounts, api.VolumeMount{Name: partName, ReadOnly: partReadOnly, MountPath: partContainerDir})
			} else {
				volumemounts = append(volumemounts, api.VolumeMount{Name: partName, ReadOnly: partReadOnly, MountPath: partContainerDir})
			}
			source := &api.HostPathVolumeSource{
				Path: partHostDir,
			}
			vsource := api.VolumeSource{HostPath: source}
			volumes = append(volumes, api.Volume{Name: partName, VolumeSource: vsource})
		}
		rc.Spec.Template.Spec.Containers[0].VolumeMounts = volumemounts
		rc.Spec.Template.Spec.Volumes = volumes

		// Configure the container restart policy.
		switch service.Restart {
		case "", "always":
			rc.Spec.Template.Spec.RestartPolicy = api.RestartPolicyAlways
		case "no":
			rc.Spec.Template.Spec.RestartPolicy = api.RestartPolicyNever
		case "on-failure":
			rc.Spec.Template.Spec.RestartPolicy = api.RestartPolicyOnFailure
		default:
			log.Fatalf("Unknown restart policy %s for service %s", service.Restart, name)
		}

		data, err := json.MarshalIndent(rc, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal the replication controller: %v", err)
		}

		// Save the replication controller for the Docker compose service to the
		// configs directory.
		outputFileName := fmt.Sprintf("%s-rc.json", name)
		outputFilePath := filepath.Join(outputDir, outputFileName)
		if err := ioutil.WriteFile(outputFilePath, data, 0644); err != nil {
			log.Fatalf("Failed to write replication controller %s: %v", outputFileName, err)
		}
		fmt.Println(outputFilePath)
	}
}
