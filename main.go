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

import "flag"

var (
	composeFilePath string
	outputDir       string
	asJSON          bool
)

func init() {
	flag.StringVar(&composeFilePath, "compose-file-path", "./", "Specify an alternate path for compose files")
	flag.StringVar(&outputDir, "output-dir", "output", "Kubernetes configs output `directory`")
	flag.BoolVar(&asJSON, "json", false, "output json instead of yaml")
}

func main() {
	flag.Parse()
	dockerCompose := parseDockerCompose()
	rancherCompose := parseRancherCompose()
	processDockerCompose(dockerCompose, rancherCompose)
	processRancherCompose(rancherCompose)
}
