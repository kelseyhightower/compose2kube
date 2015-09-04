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

	"github.com/docker/libcompose/project"
	"k8s.io/kubernetes/pkg/api"
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
		ProjectName: "kube",
		ComposeFile: composeFile,
	})

	err := p.Parse()
	if err != nil {
		log.Fatal(err)
	}
	os.MkdirAll(outputDir, 0755)

	for name, service := range p.Configs {
		rc := &api.ReplicationController{
			TypeMeta: api.TypeMeta{
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
								Name:  name,
								Image: service.Image,
							},
						},
					},
				},
			},
		}

		// Configure the container ports.
		ports := make([]api.ContainerPort, 0)
		for _, port := range service.Ports {
			if portNumber, err := strconv.Atoi(port); err == nil {
				ports = append(ports, api.ContainerPort{ContainerPort: portNumber})
				continue
			}
			log.Fatal("invalid container port %s for service %s", port, name)
		}

		rc.Spec.Template.Spec.Containers[0].Ports = ports

		// Configure the container restart policy.
		switch service.Restart {
		case "", "always":
			rc.Spec.Template.Spec.RestartPolicy = api.RestartPolicyAlways
		case "no":
			rc.Spec.Template.Spec.RestartPolicy = api.RestartPolicyNever
		case "on-failure":
			rc.Spec.Template.Spec.RestartPolicy = api.RestartPolicyOnFailure
		default:
			log.Fatalf("unknown restart policy %s for service %s", service.Restart, name)
		}

		data, err := json.MarshalIndent(rc, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		// Save the replication controller for the Docker compose service to the
		// configs directory.
		outputFileName := fmt.Sprintf("%s-rc.yaml", name)
		outputFilePath := filepath.Join(outputDir, outputFileName)
		if err := ioutil.WriteFile(outputFilePath, data, 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Println(outputFilePath)
	}
}
