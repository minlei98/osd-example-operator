// DO NOT REMOVE TAGS BELOW. IF ANY NEW TEST FILES ARE CREATED UNDER /test/e2e, PLEASE ADD THESE TAGS TO THEM IN ORDER TO BE EXCLUDED FROM UNIT TESTS.
//go:build osde2e

package osde2etests

import (
	"context"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	ocme2e "github.com/openshift/osde2e-common/pkg/clients/ocm"
	"github.com/openshift/osde2e-common/pkg/clients/openshift"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("osd-example-operator", func() {
	It("asserts success", func(ctx context.Context) {
		Expect(true).To(BeTrue(), "True should be true")
	})

	It("should connect to stage ocm client", func(ctx context.Context) {
		By("Getting ocm creds")
		clientID := os.Getenv("OCM_CLIENT_ID")
		clientSecret := os.Getenv("OCM_CLIENT_SECRET")
		Expect(clientID).NotTo(BeEmpty(), "OCM_CLIENT_ID must be set")
		Expect(clientSecret).NotTo(BeEmpty(), "OCM_CLIENT_SECRET must be set")
		ocmEnv := ocme2e.Stage
		_, err := ocme2e.New(ctx, "", clientID, clientSecret, ocmEnv)
		Expect(err).ShouldNot(HaveOccurred(), "Unable to setup stage OCM Client")
	})

	// Failing test for log analysis demo
	// It("should fail on purpose", func() {
	// 	fmt.Println("Running Intentional Failure Test")
	// 	Expect(true).To(BeFalse(), "This test is designed to fail intentionally")
	// })
})

var _ = Describe("MC Simulation", Ordered, func() {
	var (
		k8s       *openshift.Client
		dynClient dynamic.Interface
		hcpName   string
	)
	const hcpNamespace = "default"

	BeforeAll(func(ctx context.Context) {
		if !isMCSimulationEnabled() {
			Skip("MC_SIMULATION_ENABLED is not set; skipping MC Simulation tests")
		}
		hcpName = fmt.Sprintf("mc-sim-e2e-%x", GinkgoRandomSeed())
		log.SetLogger(GinkgoLogr)
		var err error
		k8s, err = openshift.New(GinkgoLogr)
		Expect(err).ShouldNot(HaveOccurred(), "unable to setup k8s client")
		Expect(hypershiftv1beta1.AddToScheme(k8s.GetScheme())).Should(Succeed())
		dynClient, err = dynamic.NewForConfig(k8s.GetConfig())
		Expect(err).ShouldNot(HaveOccurred(), "unable to create dynamic client")
	})

	It("should have MC_SIMULATION_ENABLED env var propagated", func(_ context.Context) {
		Expect(isMCSimulationEnabled()).To(BeTrue())
	})

	It("should have MC_SIMULATION_SKIP_INFRA_CHECK env var propagated", func(_ context.Context) {
		Expect(strings.EqualFold(os.Getenv("MC_SIMULATION_SKIP_INFRA_CHECK"), "true")).To(BeTrue())
	})

	It("should find HostedControlPlane CRD installed by osde2e", func(ctx context.Context) {
		hcpRes := hypershiftv1beta1.GroupVersion.WithResource("hostedcontrolplanes")
		_, err := dynClient.Resource(hcpRes).List(ctx, metav1.ListOptions{Limit: 1})
		Expect(err).ShouldNot(HaveOccurred(), "HCP CRD should be installed by osde2e orchestrator")
	})

	It("should create and status-update a fake HCP CR", func(ctx context.Context) {
		hcp := createMCStyleHCP(hcpName, hcpNamespace)
		err := k8s.Create(ctx, hcp)
		Expect(err).ShouldNot(HaveOccurred(), "failed to create HCP CR")

		err = setHostedControlPlaneAvailable(ctx, dynClient, hcp)
		Expect(err).ShouldNot(HaveOccurred(), "failed to set HCP status to Available")
	})

	AfterAll(func(ctx context.Context) {
		if k8s == nil {
			return
		}
		hcp := &hypershiftv1beta1.HostedControlPlane{
			ObjectMeta: metav1.ObjectMeta{Name: hcpName, Namespace: hcpNamespace},
		}
		if err := k8s.Delete(ctx, hcp); err != nil && !apierrors.IsNotFound(err) {
			GinkgoLogr.Error(err, "failed to delete HCP CR during cleanup")
		}
	})
})
