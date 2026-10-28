package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
	"github.com/labstack/gommon/random"
)

// createRequestRecorder captures the JSON body the provider sends when it creates a
// service, so tests can assert exactly which DMR fields went over the wire.
type createRequestRecorder struct {
	mu   sync.Mutex
	body map[string]interface{}
}

func (rec *createRequestRecorder) install(instance *internal.TestInstance, params internal.ConfigurableParams) {
	httpmock.RegisterResponder("POST", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices",
		func(r *http.Request) (*http.Response, error) {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			var parsed map[string]interface{}
			if err := json.Unmarshal(raw, &parsed); err != nil {
				return nil, fmt.Errorf("create service request body is not JSON: %w", err)
			}
			rec.mu.Lock()
			rec.body = parsed
			rec.mu.Unlock()
			return internal.JsonResponder(202, internal.CreateServiceOperationResponse(params))(r)
		})
}

func (rec *createRequestRecorder) snapshot() map[string]interface{} {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return rec.body
}

// checkCreateRequest asserts the captured create request: a nil expectation means the
// field must be omitted from the request body.
func checkCreateRequest(rec *createRequestRecorder, wantDmrEnabled *bool, wantClusterName *string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		body := rec.snapshot()
		if body == nil {
			return fmt.Errorf("no create service request was captured")
		}

		dmrEnabled, hasDmrEnabled := body["dmrEnabled"]
		switch {
		case wantDmrEnabled == nil && hasDmrEnabled:
			return fmt.Errorf("expected dmrEnabled to be omitted from the create request, got %v", dmrEnabled)
		case wantDmrEnabled != nil && !hasDmrEnabled:
			return fmt.Errorf("expected dmrEnabled=%t in the create request, but it was omitted", *wantDmrEnabled)
		case wantDmrEnabled != nil && dmrEnabled != *wantDmrEnabled:
			return fmt.Errorf("expected dmrEnabled=%t in the create request, got %v", *wantDmrEnabled, dmrEnabled)
		}

		clusterName, hasClusterName := body["clusterName"]
		switch {
		case wantClusterName == nil && hasClusterName:
			return fmt.Errorf("expected clusterName to be omitted from the create request, got %v", clusterName)
		case wantClusterName != nil && !hasClusterName:
			return fmt.Errorf("expected clusterName=%q in the create request, but it was omitted", *wantClusterName)
		case wantClusterName != nil && clusterName != *wantClusterName:
			return fmt.Errorf("expected clusterName=%q in the create request, got %v", *wantClusterName, clusterName)
		}
		return nil
	}
}

func boolPtr(b bool) *bool { return &b }

// Attribute paths, values and fixtures shared by the DMR tests.
const (
	attrDmrEnabled     = "dmr_enabled"
	attrClusterName    = "cluster_name"
	attrDmrClusterName = "dmr_cluster.name"
	attrId             = "id"
	valueTrue          = "true"
	valueFalse         = "false"
	serviceClass1K     = "ENTERPRISE_1K_STANDALONE"
	resourceTypePrefix = "solacecloud_service."
)

// HCL fragments for the dmr_enabled / cluster_name arguments used across the tests.
const (
	hclDmrEnabledTrue  = "  dmr_enabled      = true\n"
	hclDmrEnabledFalse = "  dmr_enabled      = false\n"
	hclOrphanCluster   = "  cluster_name     = \"orphan-cluster\"\n"
)

func dmrServiceHcl(instance *internal.TestInstance, params internal.ConfigurableParams, dmrArguments string) string {
	return instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "%s"
  service_class_id = "%s"
%s}
`, params.ServiceName, params.ServiceName, internal.TestDatacenterId(), params.ServiceClass, dmrArguments)
}

// TestServiceCreationWithDmrOptIn covers the request matrix for creating a service:
// opting in with and without a cluster name, omitting the opt-in, and opting out
// explicitly. It asserts both the request sent to Solace Cloud and the resulting state.
func TestServiceCreationWithDmrOptIn(t *testing.T) {
	const namedCluster = "my-dmr-cluster"
	const generatedCluster = "cluster-eks-eu-central-1a-3-62xaikge1vm"

	cases := []struct {
		name            string
		dmrArguments    string
		params          internal.ConfigurableParams
		wantDmrEnabled  *bool
		wantClusterName *string
		checks          func(resourceName string) []resource.TestCheckFunc
	}{
		{
			name:            "opt in with a cluster name",
			dmrArguments:    hclDmrEnabledTrue + "  cluster_name     = \"" + namedCluster + "\"\n",
			params:          internal.ConfigurableParams{ClusterName: namedCluster},
			wantDmrEnabled:  boolPtr(true),
			wantClusterName: stringPtr(namedCluster),
			checks: func(resourceName string) []resource.TestCheckFunc {
				return []resource.TestCheckFunc{
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueTrue),
					resource.TestCheckResourceAttr(resourceName, attrClusterName, namedCluster),
					resource.TestCheckResourceAttr(resourceName, attrDmrClusterName, namedCluster),
				}
			},
		},
		{
			name:            "opt in and let Solace Cloud name the cluster",
			dmrArguments:    hclDmrEnabledTrue,
			params:          internal.ConfigurableParams{},
			wantDmrEnabled:  boolPtr(true),
			wantClusterName: nil,
			checks: func(resourceName string) []resource.TestCheckFunc {
				return []resource.TestCheckFunc{
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueTrue),
					resource.TestCheckResourceAttr(resourceName, attrClusterName, generatedCluster),
					resource.TestCheckResourceAttr(resourceName, attrDmrClusterName, generatedCluster),
				}
			},
		},
		{
			name:            "omit dmr_enabled creates a service without DMR",
			dmrArguments:    "",
			params:          internal.ConfigurableParams{DmrOff: true},
			wantDmrEnabled:  nil,
			wantClusterName: nil,
			checks: func(resourceName string) []resource.TestCheckFunc {
				return []resource.TestCheckFunc{
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueFalse),
					resource.TestCheckNoResourceAttr(resourceName, attrClusterName),
					resource.TestCheckNoResourceAttr(resourceName, attrDmrClusterName),
				}
			},
		},
		{
			name:            "dmr_enabled false is sent explicitly",
			dmrArguments:    hclDmrEnabledFalse,
			params:          internal.ConfigurableParams{DmrOff: true},
			wantDmrEnabled:  boolPtr(false),
			wantClusterName: nil,
			checks: func(resourceName string) []resource.TestCheckFunc {
				return []resource.TestCheckFunc{
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueFalse),
					resource.TestCheckNoResourceAttr(resourceName, attrClusterName),
					resource.TestCheckNoResourceAttr(resourceName, attrDmrClusterName),
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instance := internal.NewTestInstance()
			params := tc.params
			params.ServiceClass = serviceClass1K
			params.ServiceName = "Test_DMR_" + random.String(8)
			instance.Init(params)
			if !instance.IsMocked() {
				return
			}
			recorder := &createRequestRecorder{}
			recorder.install(instance, params)

			resourceName := resourceTypePrefix + params.ServiceName
			checks := append(tc.checks(resourceName),
				resource.TestCheckResourceAttrSet(resourceName, attrId),
				checkCreateRequest(recorder, tc.wantDmrEnabled, tc.wantClusterName),
			)

			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: dmrServiceHcl(instance, params, tc.dmrArguments),
						Check:  resource.ComposeTestCheckFunc(checks...),
					},
					// Re-applying the same configuration must not produce a diff.
					{
						Config:   dmrServiceHcl(instance, params, tc.dmrArguments),
						PlanOnly: true,
					},
				},
			})
		})
	}
}

// TestServiceWithoutDmrArgumentsReadsDmrStateFromService checks the compatibility promise:
// a configuration that sets neither dmr_enabled nor cluster_name applies against an
// existing DMR-enabled service and keeps applying without a plan diff.
func TestServiceWithoutDmrArgumentsReadsDmrStateFromService(t *testing.T) {
	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{
		ServiceClass: serviceClass1K,
		ServiceName:  "Test_DMR_Legacy_" + random.String(8),
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}
	resourceName := resourceTypePrefix + params.ServiceName

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: dmrServiceHcl(instance, params, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueTrue),
					resource.TestCheckResourceAttrSet(resourceName, attrClusterName),
					resource.TestCheckResourceAttrSet(resourceName, attrDmrClusterName),
				),
			},
			{
				Config:   dmrServiceHcl(instance, params, ""),
				PlanOnly: true,
			},
		},
	})
}

// TestClusterNameWithoutDmrOptInIsRejected covers the contradictory combination: naming a
// cluster without opting in to DMR fails at plan time, before any request is sent.
func TestClusterNameWithoutDmrOptInIsRejected(t *testing.T) {
	cases := map[string]string{
		"dmr_enabled omitted": hclOrphanCluster,
		"dmr_enabled false":   hclDmrEnabledFalse + hclOrphanCluster,
	}

	for name, dmrArguments := range cases {
		t.Run(name, func(t *testing.T) {
			instance := internal.NewTestInstance()
			params := internal.ConfigurableParams{
				ServiceClass: serviceClass1K,
				ServiceName:  "Test_DMR_Reject_" + random.String(8),
			}
			instance.Init(params)
			if !instance.IsMocked() {
				return
			}
			recorder := &createRequestRecorder{}
			recorder.install(instance, params)

			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      dmrServiceHcl(instance, params, dmrArguments),
						ExpectError: regexp.MustCompile("cluster_name requires dmr_enabled = true"),
					},
				},
			})

			if recorder.snapshot() != nil {
				t.Errorf("expected no create request to be sent when cluster_name is set without dmr_enabled = true")
			}
		})
	}
}

// TestDmrEnabledCanBeEnabledAndDisabledInPlace verifies the update path: changing dmr_enabled
// on an existing service enables or disables DMR through the Solace Cloud API rather than
// replacing the service.
func TestDmrEnabledCanBeEnabledAndDisabledInPlace(t *testing.T) {
	cases := []struct {
		name          string
		startDmrOff   bool
		createArgs    string
		changedArgs   string
		createdValue  string
		expectedValue string
		expectCluster bool
	}{
		{
			name:          "enabling DMR on a service created without it",
			startDmrOff:   true,
			createArgs:    "",
			changedArgs:   hclDmrEnabledTrue,
			createdValue:  valueFalse,
			expectedValue: valueTrue,
			expectCluster: true,
		},
		{
			name:          "disabling DMR on a service created with it",
			startDmrOff:   false,
			createArgs:    hclDmrEnabledTrue,
			changedArgs:   hclDmrEnabledFalse,
			createdValue:  valueTrue,
			expectedValue: valueFalse,
			expectCluster: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instance := internal.NewTestInstance()
			params := internal.ConfigurableParams{
				ServiceClass: serviceClass1K,
				ServiceName:  "Test_DMR_Toggle_" + random.String(8),
				DmrOff:       tc.startDmrOff,
			}
			instance.Init(params)
			if !instance.IsMocked() {
				return
			}
			instance.SetupDmrClusterMocks(params, "")

			resourceName := resourceTypePrefix + params.ServiceName
			clusterCheck := resource.TestCheckResourceAttrSet(resourceName, attrDmrClusterName)
			if !tc.expectCluster {
				clusterCheck = resource.TestCheckNoResourceAttr(resourceName, attrDmrClusterName)
			}

			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: dmrServiceHcl(instance, params, tc.createArgs),
						Check:  resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, tc.createdValue),
					},
					{
						Config: dmrServiceHcl(instance, params, tc.changedArgs),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, tc.expectedValue),
							clusterCheck,
						),
					},
					// The change is applied in place, so a re-plan is empty.
					{
						Config:   dmrServiceHcl(instance, params, tc.changedArgs),
						PlanOnly: true,
					},
				},
			})
		})
	}
}

// TestEnablingDmrWithAClusterName verifies that a cluster name supplied while enabling DMR is
// sent to Solace Cloud and read back into state.
func TestEnablingDmrWithAClusterName(t *testing.T) {
	const chosenName = "chosen-dmr-cluster"

	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{
		ServiceClass: serviceClass1K,
		ServiceName:  "Test_DMR_Named_" + random.String(8),
		DmrOff:       true,
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}
	instance.SetupDmrClusterMocks(params, "")

	resourceName := resourceTypePrefix + params.ServiceName
	enableWithName := hclDmrEnabledTrue + "  cluster_name     = \"" + chosenName + "\"\n"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: dmrServiceHcl(instance, params, ""),
				Check:  resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueFalse),
			},
			{
				Config: dmrServiceHcl(instance, params, enableWithName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueTrue),
					resource.TestCheckResourceAttr(resourceName, attrClusterName, chosenName),
					resource.TestCheckResourceAttr(resourceName, attrDmrClusterName, chosenName),
				),
			},
		},
	})
}

// TestDisablingDmrSurfacesTheApiRefusal verifies that a refusal from Solace Cloud, such as
// disabling DMR on a service that belongs to an event mesh, is reported as a Terraform error
// rather than silently leaving the service unchanged.
func TestDisablingDmrSurfacesTheApiRefusal(t *testing.T) {
	const refusal = "DMR cannot be disabled on a service that is part of an event mesh."

	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{
		ServiceClass: serviceClass1K,
		ServiceName:  "Test_DMR_Refused_" + random.String(8),
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}
	instance.SetupDmrClusterMocks(params, refusal)

	resourceName := resourceTypePrefix + params.ServiceName

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: dmrServiceHcl(instance, params, hclDmrEnabledTrue),
				Check:  resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueTrue),
			},
			{
				Config:      dmrServiceHcl(instance, params, hclDmrEnabledFalse),
				ExpectError: regexp.MustCompile("part of an event mesh"),
			},
		},
	})
}

// TestAddingClusterNameToADmrOffServiceIsRejected covers the update case of the cross-attribute
// rule: a cluster name added to an existing service without DMR is refused at plan time, rather
// than producing an apply that reports a result inconsistent with the plan.
func TestAddingClusterNameToADmrOffServiceIsRejected(t *testing.T) {
	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{
		ServiceClass: serviceClass1K,
		ServiceName:  "Test_DMR_LateCluster_" + random.String(8),
		DmrOff:       true,
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	resourceName := resourceTypePrefix + params.ServiceName

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: dmrServiceHcl(instance, params, ""),
				Check:  resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueFalse),
			},
			{
				Config:      dmrServiceHcl(instance, params, hclOrphanCluster),
				ExpectError: regexp.MustCompile("cluster_name requires dmr_enabled = true"),
			},
		},
	})
}

// TestPreUpgradeConfigNamingTheClusterKeepsApplying is the compatibility case the reviewer
// raised: a configuration written before dmr_enabled existed names the DMR cluster and says
// nothing about dmr_enabled. Against a service that has DMR it must plan clean after upgrading.
func TestPreUpgradeConfigNamingTheClusterKeepsApplying(t *testing.T) {
	const namedCluster = "pre-upgrade-cluster"

	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{
		ServiceClass: serviceClass1K,
		ServiceName:  "Test_DMR_PreUpgrade_" + random.String(8),
		ClusterName:  namedCluster,
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	resourceName := resourceTypePrefix + params.ServiceName
	clusterOnly := "  cluster_name     = \"" + namedCluster + "\"\n"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// The service exists with DMR and a named cluster (created with the opt-in).
			{
				Config: dmrServiceHcl(instance, params, hclDmrEnabledTrue+clusterOnly),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueTrue),
					resource.TestCheckResourceAttr(resourceName, attrClusterName, namedCluster),
				),
			},
			// The pre-upgrade shape of the configuration: cluster_name only. No error, no diff.
			{
				Config:   dmrServiceHcl(instance, params, clusterOnly),
				PlanOnly: true,
			},
		},
	})
}

// TestLegacyPackageWithDmrDisabledConvergesAtCreate reproduces the destructive scenario the
// reviewer traced and asserts the corrected behaviour. A service package that always enables DMR
// ignores dmr_enabled = false at creation. The provider must then disable DMR as part of the same
// apply, record the service as it really is, and plan clean afterwards, so no later apply tears
// down a cluster by surprise.
func TestLegacyPackageWithDmrDisabledConvergesAtCreate(t *testing.T) {
	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{
		ServiceClass: serviceClass1K,
		ServiceName:  "Test_DMR_Legacy_Off_" + random.String(8),
		// DmrOff false: the mocked service comes up with a cluster, as a legacy package would.
		DmrOff: false,
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}
	instance.SetupDmrClusterMocks(params, "")

	resourceName := resourceTypePrefix + params.ServiceName
	deleteClusterCall := "DELETE " + instance.GetBaseURL() + "/api/v2/missionControl/eventBrokerServices/6q1p55o6ovr/broker/cluster"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: dmrServiceHcl(instance, params, hclDmrEnabledFalse),
				Check: resource.ComposeTestCheckFunc(
					// State records the converged service, not the planned value.
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueFalse),
					resource.TestCheckNoResourceAttr(resourceName, attrClusterName),
					resource.TestCheckNoResourceAttr(resourceName, attrDmrClusterName),
					// DMR was disabled during the create, exactly once.
					func(*terraform.State) error {
						if calls := httpmock.GetCallCountInfo()[deleteClusterCall]; calls != 1 {
							return fmt.Errorf("expected DMR to be disabled once during create, got %d delete calls", calls)
						}
						return nil
					},
				),
			},
			// Nothing is left to converge, so no later apply deletes anything.
			{
				Config:   dmrServiceHcl(instance, params, hclDmrEnabledFalse),
				PlanOnly: true,
				Check: func(*terraform.State) error {
					if calls := httpmock.GetCallCountInfo()[deleteClusterCall]; calls != 1 {
						return fmt.Errorf("expected no further delete calls after create, got %d in total", calls)
					}
					return nil
				},
			},
		},
	})
}
