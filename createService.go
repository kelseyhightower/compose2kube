/*
Copyright 2016 German Ramos. All rights reserved.
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
	"github.com/docker/libcompose/config"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

func createService(shortName string, service *config.ServiceConfig, rc *api.ReplicationController) *api.Service {
	ports := make([]api.ServicePort, len(rc.Spec.Template.Spec.Containers[0].Ports))
	for i, port := range rc.Spec.Template.Spec.Containers[0].Ports {
		ports[i].Port = port.ContainerPort
	}

	srv := &api.Service{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      shortName,
			Namespace: "default",
		},
		Spec: api.ServiceSpec{
			Selector: map[string]string{"service": shortName},
			Ports:    ports,
		},
	}

	return srv
}
