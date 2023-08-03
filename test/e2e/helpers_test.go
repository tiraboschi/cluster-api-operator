//go:build e2e
// +build e2e

/*
Copyright 2022 The Kubernetes Authors.

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

package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ctx = context.Background()
)

const (
	timeout           = 5 * time.Minute
	operatorNamespace = "capi-operator-system"

	previousCAPIVersion = "v1.4.2"

	coreProviderName           = "cluster-api"
	coreProviderDeploymentName = "capi-controller-manager"

	bootstrapProviderName           = "kubeadm"
	bootstrapProviderDeploymentName = "capi-kubeadm-bootstrap-controller-manager"

	cpProviderName           = "kubeadm"
	cpProviderDeploymentName = "capi-kubeadm-control-plane-controller-manager"

	infraProviderName           = "docker"
	infraProviderDeploymentName = "capd-controller-manager"

	customManifestsFolder = "resources/"
)

func WaitForObjectToBeDeleted(cl client.Client, ctx context.Context, key client.ObjectKey, obj client.Object) (bool, error) {
	if err := cl.Get(ctx, key, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	}

	return false, nil
}

type HelmChartHelper struct {
	BinaryPath      string
	Path            string
	Name            string
	Kubeconfig      string
	DryRun          bool
	Wait            bool
	AdditionalFlags []string
}

// InstallChart performs an install of the helm chart. Install returns the rendered manifest
// with some additional data that can't be parsed as yaml. This function processes the output and returns only the optional resources,
// marked as post install hooks.
func (h *HelmChartHelper) InstallChart(values map[string]string) (string, error) {
	args := []string{"install", "--kubeconfig", h.Kubeconfig, h.Name, h.Path}
	if h.DryRun {
		args = append(args, "--dry-run")
	}
	if h.Wait {
		args = append(args, "--wait")
	}
	for key, value := range values {
		args = append(args, "--set")
		args = append(args, fmt.Sprintf("%s=%s", key, value))
	}
	if h.AdditionalFlags != nil {
		args = append(args, h.AdditionalFlags...)
	}

	cmd := exec.Command(h.BinaryPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run helm install: %w, output: %s", err, string(out))
	}

	outString := string(out)
	startIndex := strings.Index(outString, "HOOKS:")
	endIndex := strings.Index(outString, "MANIFEST:")

	if startIndex != -1 && endIndex != -1 {
		res := outString[startIndex+len("HOOKS:") : endIndex]
		res = strings.TrimPrefix(res, "\n")
		res = strings.TrimSuffix(res, "\n")
		return res, nil
	}

	return "", fmt.Errorf("failed to parse helm output")
}
