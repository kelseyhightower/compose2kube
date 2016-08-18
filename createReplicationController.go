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
	"log"
	"strconv"
	"strings"

	"github.com/docker/libcompose/config"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

func createReplicationController(shortName string, service *config.ServiceConfig, scale int) *api.ReplicationController {
	rc := &api.ReplicationController{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "ReplicationController",
			APIVersion: "v1",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      shortName,
			Namespace: "${NAMESPACE}",
			Labels:    map[string]string{"service": shortName},
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
