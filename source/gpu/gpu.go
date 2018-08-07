/*
Copyright 2017 The Kubernetes Authors.

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

package gpu

import (
	"os/exec"
        "bytes"
	"fmt"
)

func ExecCommand(cmdName string, arg ...string) (bytes.Buffer, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(cmdName, arg...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("CMD--" + cmdName + ": " + fmt.Sprint(err) + ": " + stderr.String())
	}

	return out, err
}


type Source struct{}

func (s Source) Name() string { return "gpu" }

func (s Source) Discover() ([]string, error) {
	features := []string{}
	out, err := ExecCommand("find_nvidia_display_adapter.sh")
	if err != nil {
		return nil, fmt.Errorf("Failed to detect a gpu, please check if the system has a gpu: %s %s", err.Error(), out)
	}
	features = append(features, "present")
	return features, nil
}
