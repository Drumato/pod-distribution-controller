/*
Copyright 2024.

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
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/Drumato/pod-distribution-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const kubectlPath = "../../bin/k8s/1.29.0-linux-amd64/kubectl"
const projectImage = "example.com/pod-distribution-controller:v0.0.1"

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	fmt.Fprintf(GinkgoWriter, "Starting pod-distribution-controller suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	By("creating kind-test cluster")
	err := exec.Command("kind", "create", "cluster", "--name", "kind-test").Run()
	Expect(err).To(Succeed())

	By("setting kubectl context to kind-test cluster")
	err = exec.Command(kubectlPath, "cluster-info", "--context", "kind-kind-test").Run()
	Expect(err).To(Succeed())

	By("testing get nodes")
	err = exec.Command(kubectlPath, "get", "nodes").Run()
	Expect(err).To(Succeed())

	By("installing prometheus operator")
	Expect(utils.InstallPrometheusOperator()).To(Succeed())

	By("installing the cert-manager")
	Expect(utils.InstallCertManager()).To(Succeed())

	By("installing ArgoCD operator")
	Expect(utils.InstallArgoCDOperator()).To(Succeed())

	By("creating manager namespace")
	cmd := exec.Command(kubectlPath, "create", "ns", namespace)
	_, _ = utils.Run(cmd)

	// projectimage stores the name of the image used in the example

	By("building the manager(Operator) image")
	cmd = exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectImage))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("loading the the manager(Operator) image on Kind")
	err = utils.LoadImageToKindClusterWithName("kind-test", projectImage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("installing CRDs")
	cmd = exec.Command("make", "install")
	_, err = utils.Run(cmd)
	Expect(err).To(Succeed())

	By("deploying the controller-manager")
	cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("validating that the controller-manager pod is running as expected")
	verifyControllerUp := func() error {
		// Get pod name

		const kubectlPathInFn = "./bin/k8s/1.29.0-linux-amd64/kubectl"
		cmd = exec.Command(kubectlPathInFn, "get",
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}"+
				"{{ if not .metadata.deletionTimestamp }}"+
				"{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}",
			"-n", namespace,
		)

		podOutput, err := utils.Run(cmd)
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		podNames := utils.GetNonEmptyLines(string(podOutput))
		if len(podNames) != 1 {
			return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
		}
		controllerPodName := podNames[0]
		ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

		// Validate pod status
		cmd = exec.Command(kubectlPathInFn, "get",
			"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
			"-n", namespace,
		)
		status, err := utils.Run(cmd)
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if string(status) != "Running" {
			return fmt.Errorf("controller pod in %s status", status)
		}
		return nil
	}
	EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())

})

var _ = AfterSuite(func() {
	By("uninstalling the ArgoCD")
	utils.UninstallArgoCD()

	By("uninstalling the Prometheus manager bundle")
	utils.UninstallPrometheusOperator()

	By("uninstalling the cert-manager bundle")
	utils.UninstallCertManager()

	By("removing manager namespace")
	cmd := exec.Command(kubectlPath, "delete", "ns", namespace)
	_, _ = utils.Run(cmd)

	err := exec.Command("kind", "delete", "cluster", "--name", "kind-test").Run()
	Expect(err).To(Succeed())
})
