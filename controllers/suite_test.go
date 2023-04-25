/*
Copyright 2023.

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

package controllers_test

import (
	"context"
	"errors"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/fhivemind/plant-operator/controllers"
	"github.com/fhivemind/plant-operator/controllers/workflow"
	"io"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"math/rand"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	operatorv1 "github.com/fhivemind/plant-operator/api/v1"
	//+kubebuilder:scaffold:imports
)

// These test use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	Cfg         *rest.Config
	PlantClient client.Client
	TestEnv     *envtest.Environment
	Ctx         context.Context
	Cancel      func()
	Timeout     = time.Second * 15
	Interval    = time.Millisecond * 250
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	Ctx, Cancel = context.WithCancel(context.TODO())
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")

	TestEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
		},
		CRDs: loadExternalCrds(
			filepath.Join("..", "config", "samples", "test", "crds"),
			"cert-manager.v1.11.0.crds.yaml",
		),
		ErrorIfCRDPathMissing: true,
	}

	// Cfg is defined in this file globally.
	cfg, err := TestEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// register test
	Expect(operatorv1.AddToScheme(scheme.Scheme)).NotTo(HaveOccurred())
	Expect(certv1.AddToScheme(scheme.Scheme)).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	// create manager
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	// configure reconciler
	err = (&controllers.PlantReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Workflow: workflow.NewManager(),
		Recorder: mgr.GetEventRecorderFor("plant-controller"),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	// set client
	PlantClient = mgr.GetClient()
	Expect(err).NotTo(HaveOccurred())
	Expect(PlantClient).NotTo(BeNil())

	// configure start
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(Ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	Cancel()

	err := TestEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func randString(n int) string {
	letters := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] //nolint:gosec
	}
	return string(b)
}

func loadExternalCrds(path string, files ...string) []*v1.CustomResourceDefinition {
	var crds []*v1.CustomResourceDefinition
	for _, file := range files {
		crdPath := filepath.Join(path, file)
		moduleFile, err := os.Open(crdPath)
		Expect(err).ToNot(HaveOccurred())
		decoder := yaml.NewYAMLOrJSONDecoder(moduleFile, 2048)
		for {
			crd := &v1.CustomResourceDefinition{}
			if err = decoder.Decode(crd); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				continue
			}
			crds = append(crds, crd)
		}
	}
	return crds
}
