// DO NOT REMOVE TAGS BELOW. IF ANY NEW TEST FILES ARE CREATED UNDER /test/e2e, PLEASE ADD THESE TAGS TO THEM IN ORDER TO BE EXCLUDED FROM UNIT TESTS.
//go:build osde2e

package osde2etests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func isMCSimulationEnabled() bool {
	v := os.Getenv("MC_SIMULATION_ENABLED")
	return strings.EqualFold(v, "true")
}

func createMCStyleHCP(name, namespace string) *hypershiftv1beta1.HostedControlPlane {
	hostname := fmt.Sprintf("api.%s.test.devshift.org", name)
	return &hypershiftv1beta1.HostedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"api.openshift.com/name":        name,
				"api.openshift.com/environment": "staging",
				"test-type":                     "osde2e-mc-simulation",
			},
		},
		Spec: hypershiftv1beta1.HostedControlPlaneSpec{
			Platform: hypershiftv1beta1.PlatformSpec{
				Type: hypershiftv1beta1.AWSPlatform,
				AWS: &hypershiftv1beta1.AWSPlatformSpec{
					Region: "us-west-2",
				},
			},
			Services: []hypershiftv1beta1.ServicePublishingStrategyMapping{
				{
					Service: hypershiftv1beta1.APIServer,
					ServicePublishingStrategy: hypershiftv1beta1.ServicePublishingStrategy{
						Type: hypershiftv1beta1.Route,
						Route: &hypershiftv1beta1.RoutePublishingStrategy{
							Hostname: hostname,
						},
					},
				},
			},
		},
	}
}

// setHostedControlPlaneAvailable sets the HCP status to Available via the
// status subresource using the dynamic client, matching the pattern an
// operator would observe on a real Management Cluster.
func setHostedControlPlaneAvailable(ctx context.Context, dynClient dynamic.Interface, hcp *hypershiftv1beta1.HostedControlPlane) error {
	hcpGVR := schema.GroupVersionResource{
		Group:    hypershiftv1beta1.GroupVersion.Group,
		Version:  hypershiftv1beta1.GroupVersion.Version,
		Resource: "hostedcontrolplanes",
	}
	statusClient := dynClient.Resource(hcpGVR).Namespace(hcp.Namespace)

	// Fetch the latest version to avoid conflict errors.
	u, err := statusClient.Get(ctx, hcp.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get HCP: %w", err)
	}

	now := metav1.Now()
	conditions := []interface{}{
		map[string]interface{}{
			"type":               string(hypershiftv1beta1.HostedControlPlaneAvailable),
			"status":             string(metav1.ConditionTrue),
			"reason":             "AsExpected",
			"message":            "HostedControlPlane is available",
			"lastTransitionTime": now.UTC().Format(time.RFC3339),
		},
	}
	if err := unstructured.SetNestedSlice(u.Object, conditions, "status", "conditions"); err != nil {
		return fmt.Errorf("set status field: %w", err)
	}

	if _, err := statusClient.UpdateStatus(ctx, u, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update HCP status: %w", err)
	}
	return nil
}
