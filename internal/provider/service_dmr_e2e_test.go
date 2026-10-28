package provider_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/labstack/gommon/random"
)

// The tests in this file deliberately run in BOTH modes: against the mocked API during
// `make test`, and against the real Solace Cloud API during `make testacc` and the staging
// acceptance job, which set SOLACE_BASE_URL. They therefore assert only what holds on any
// service package, whether or not it supports the DMR opt-in, and never assert a value that
// comes from the mock fixtures.
//
// Their purpose is the compatibility guarantee: adding dmr_enabled must not break service
// creation against a real API that predates the DMR off-by-default rollout.
//
// SOLACE_TEST_DATACENTER_ID and SOLACE_TEST_SERVICE_CLASS_ID point them at a datacenter and a
// service class that exist in whichever environment they run against, and
// SOLACE_TEST_EVENT_BROKER_VERSION selects the service package.
//
// TestAccDmrLifecycle additionally drives the enable and disable transitions against the real
// API. Each transition reconfigures a real broker, so against a real API it runs only when
// SOLACE_TEST_DMR_TRANSITIONS=true; the staging acceptance workflow sets that.

// e2eDatacenterId is the datacenter the e2e tests create services in: the one every test
// shares, which SOLACE_TEST_DATACENTER_ID overrides so the tests can run against any environment.
func e2eDatacenterId() string {
	return internal.TestDatacenterId()
}

// e2eServiceClassId is the service class the e2e tests create. It defaults to the class the
// mocked fixtures and the staging environment support, and SOLACE_TEST_SERVICE_CLASS_ID
// overrides it for environments whose datacenter offers a different set of classes.
func e2eServiceClassId() string {
	if override := os.Getenv("SOLACE_TEST_SERVICE_CLASS_ID"); override != "" {
		return override
	}
	return serviceClass1K
}

// e2eEventBrokerVersionHcl pins the event broker version when SOLACE_TEST_EVENT_BROKER_VERSION
// is set. That selects the service package, and with it whether DMR is off by default, so the
// same tests can be pointed at a package that predates the opt-in and at one that honours it.
func e2eEventBrokerVersionHcl() string {
	if version := os.Getenv("SOLACE_TEST_EVENT_BROKER_VERSION"); version != "" {
		return fmt.Sprintf("  event_broker_version = %q\n", version)
	}
	return ""
}

// e2eServiceHcl builds a service configuration for the e2e tests.
func e2eServiceHcl(instance *internal.TestInstance, name string, dmrArguments string) string {
	return instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "%s"
  service_class_id = "%s"
%s%s}
`, name, name, e2eDatacenterId(), e2eServiceClassId(), e2eEventBrokerVersionHcl(), dmrArguments)
}

// TestAccServiceWithoutDmrArgumentsStillCreates is the compatibility check: a configuration
// written before dmr_enabled existed must keep applying unchanged, and must report the DMR
// state the service actually has, whichever default the service package applies.
func TestAccServiceWithoutDmrArgumentsStillCreates(t *testing.T) {
	instance := internal.NewTestInstance()
	serviceName := "E2E_DMR_Compat_" + random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: e2eServiceClassId(),
		ServiceName:  serviceName,
	}
	instance.Init(params)

	resourceName := "solacecloud_service." + serviceName

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: e2eServiceHcl(instance, serviceName, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, attrId),
					resource.TestCheckResourceAttr(resourceName, "name", serviceName),
					// dmr_enabled is computed from the service, so it is always known after
					// apply whether or not the package enables DMR by default.
					resource.TestCheckResourceAttrSet(resourceName, attrDmrEnabled),
				),
			},
			// The configuration mentions neither DMR argument, so re-planning it must be empty.
			// This is the promise that an existing configuration keeps working untouched.
			{
				Config:   e2eServiceHcl(instance, serviceName, ""),
				PlanOnly: true,
			},
		},
	})
}

// TestAccServiceWithDmrOptInCreates checks that the new argument is accepted by a real API and
// yields a service with DMR. On a package that always enables DMR the field is ignored and the
// service has DMR anyway, so the expected result is the same either way.
func TestAccServiceWithDmrOptInCreates(t *testing.T) {
	instance := internal.NewTestInstance()
	serviceName := "E2E_DMR_OptIn_" + random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: e2eServiceClassId(),
		ServiceName:  serviceName,
	}
	instance.Init(params)

	resourceName := "solacecloud_service." + serviceName

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: e2eServiceHcl(instance, serviceName, hclDmrEnabledTrue),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, attrId),
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueTrue),
					// Solace Cloud names the cluster when the configuration does not.
					resource.TestCheckResourceAttrSet(resourceName, attrClusterName),
					resource.TestCheckResourceAttrSet(resourceName, attrDmrClusterName),
				),
			},
			{
				Config:   e2eServiceHcl(instance, serviceName, hclDmrEnabledTrue),
				PlanOnly: true,
			},
		},
	})
}

// TestAccClusterNameWithoutDmrOptInFailsAtPlan checks the cross-attribute rule in both modes.
// It is plan-only, so it reaches no API and creates nothing.
func TestAccClusterNameWithoutDmrOptInFailsAtPlan(t *testing.T) {
	instance := internal.NewTestInstance()
	serviceName := "E2E_DMR_Reject_" + random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: e2eServiceClassId(),
		ServiceName:  serviceName,
	}
	instance.Init(params)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      e2eServiceHcl(instance, serviceName, "  cluster_name     = \"e2e-orphan-cluster\"\n"),
				ExpectError: regexp.MustCompile("cluster_name requires dmr_enabled = true"),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccDmrLifecycle drives the whole DMR lifecycle on one service: created without DMR,
// enabled, disabled, then enabled again with a chosen cluster name, and finally re-planned.
//
// It is written to pass on any service package. On a package that always enables DMR, the first
// step exercises the create-time convergence (the provider disables the cluster the package
// created); on a package with DMR off by default it is a plain create. Every later step is a
// genuine transition on both kinds of package.
//
// Against a real API this test runs only when SOLACE_TEST_DMR_TRANSITIONS=true, because each
// transition reconfigures a real broker. Mocked, it always runs.
func TestAccDmrLifecycle(t *testing.T) {
	instance := internal.NewTestInstance()
	serviceName := "E2E_DMR_Lifecycle_" + random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: e2eServiceClassId(),
		ServiceName:  serviceName,
		// Mocked: behave like a package that always enables DMR, so the first step also covers
		// the create-time convergence.
		DmrOff: false,
	}
	instance.Init(params)
	if instance.IsMocked() {
		instance.SetupDmrClusterMocks(params, "")
	} else if os.Getenv("SOLACE_TEST_DMR_TRANSITIONS") != "true" {
		t.Skip("set SOLACE_TEST_DMR_TRANSITIONS=true to run the DMR transitions against a real API")
	}

	resourceName := resourceTypePrefix + serviceName
	chosenName := "e2e-dmr-" + strings.ToLower(random.String(8))
	enableWithName := hclDmrEnabledTrue + "  cluster_name     = \"" + chosenName + "\"\n"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1. Created without DMR. On a package that always enables DMR the provider disables
			//    it during the create, with a warning; either way the service ends up without one.
			{
				Config: e2eServiceHcl(instance, serviceName, hclDmrEnabledFalse),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, attrId),
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueFalse),
					resource.TestCheckNoResourceAttr(resourceName, attrClusterName),
					resource.TestCheckNoResourceAttr(resourceName, attrDmrClusterName),
				),
			},
			// 2. Enabled in place; Solace Cloud names the cluster.
			{
				Config: e2eServiceHcl(instance, serviceName, hclDmrEnabledTrue),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueTrue),
					resource.TestCheckResourceAttrSet(resourceName, attrClusterName),
					resource.TestCheckResourceAttrSet(resourceName, attrDmrClusterName),
				),
			},
			// 3. Disabled in place; the cluster is gone.
			{
				Config: e2eServiceHcl(instance, serviceName, hclDmrEnabledFalse),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueFalse),
					resource.TestCheckNoResourceAttr(resourceName, attrClusterName),
					resource.TestCheckNoResourceAttr(resourceName, attrDmrClusterName),
				),
			},
			// 4. Enabled again with a chosen name.
			{
				Config: e2eServiceHcl(instance, serviceName, enableWithName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, attrDmrEnabled, valueTrue),
					resource.TestCheckResourceAttr(resourceName, attrClusterName, chosenName),
					resource.TestCheckResourceAttr(resourceName, attrDmrClusterName, chosenName),
				),
			},
			// 5. Nothing left to converge.
			{
				Config:   e2eServiceHcl(instance, serviceName, enableWithName),
				PlanOnly: true,
			},
		},
	})
}
