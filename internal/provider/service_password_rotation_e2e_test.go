package provider_test

import (
	"fmt"
	"os"
	"strings"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/labstack/gommon/random"
)

func rotationE2eHcl(instance *internal.TestInstance, name, trigger string, users ...string) string {
	quoted := make([]string, 0, len(users))
	for _, u := range users {
		quoted = append(quoted, fmt.Sprintf("%q", u))
	}
	block := fmt.Sprintf(`  broker_password_rotation = {
    users   = [%s]
    trigger = %q
  }
`, strings.Join(quoted, ", "), trigger)
	return e2eServiceHcl(instance, name, block)
}

// Only some broker versions have a manager user, so a real API needs the flag to opt in.
func rotationManagerEnabled(instance *internal.TestInstance) bool {
	return instance.IsMocked() || os.Getenv("SOLACE_TEST_PASSWORD_ROTATION_MANAGER") == "true"
}

// A real API has no recorder, so this checks only the mock. It adds a "no request was sent"
// check that comparing passwords can't make.
func mockedResetRequestsAre(rec *internal.PasswordResetRecorder, want [][]string) resource.TestCheckFunc {
	if rec == nil {
		return func(*terraform.State) error { return nil }
	}
	return resetRequestsAre(rec, want)
}

// passwordTracker compares passwords across steps without the test knowing or printing them.
type passwordTracker struct {
	seen map[string]string
}

func (p *passwordTracker) value(s *terraform.State, resourceName, attribute string) (string, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return "", fmt.Errorf("resource %s not found in state", resourceName)
	}
	v := rs.Primary.Attributes[attribute]
	if v == "" {
		return "", fmt.Errorf("%s has no value for %s", resourceName, attribute)
	}
	return v, nil
}

func (p *passwordTracker) record(resourceName, attribute string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		v, err := p.value(s, resourceName, attribute)
		if err != nil {
			return err
		}
		p.seen[attribute] = v
		return nil
	}
}

func (p *passwordTracker) changed(resourceName, attribute string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		v, err := p.value(s, resourceName, attribute)
		if err != nil {
			return err
		}
		if v == p.seen[attribute] {
			return fmt.Errorf("expected %s to have been rotated, but it is unchanged", attribute)
		}
		return nil
	}
}

func (p *passwordTracker) unchanged(resourceName, attribute string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		v, err := p.value(s, resourceName, attribute)
		if err != nil {
			return err
		}
		if v != p.seen[attribute] {
			return fmt.Errorf("expected %s not to be rotated, but it changed", attribute)
		}
		return nil
	}
}

// Runs against both the mock and the real API, so it checks only whether a password changed,
// never its value. Against a real API, mc-manager runs only with SOLACE_TEST_PASSWORD_ROTATION_MANAGER=true.
func TestAccBrokerPasswordRotation(t *testing.T) {
	instance := internal.NewTestInstance()
	serviceName := "E2E_PwRotation_" + random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: e2eServiceClassId(),
		ServiceName:  serviceName,
	}
	instance.Init(params)
	var rec *internal.PasswordResetRecorder // set only against the mocked API
	if instance.IsMocked() {
		rec = instance.SetupPasswordResetMocks(params, 0, "")
	}

	resourceName := resourceTypePrefix + serviceName
	tracker := &passwordTracker{seen: map[string]string{}}
	withManager := rotationManagerEnabled(instance)
	// v4 if the manager step ran, else v3. The final step must reuse it: a different trigger rotates.
	lastTrigger := "v3"
	if withManager {
		lastTrigger = "v4"
	}
	sentBeforeUsersAlone := [][]string{{internal.PasswordUserViewer}, {internal.PasswordUserEditor, internal.PasswordUserViewer}}
	if withManager {
		sentBeforeUsersAlone = append(sentBeforeUsersAlone, []string{internal.PasswordUserManager, internal.PasswordUserViewer})
	}
	record := func() resource.TestCheckFunc {
		checks := []resource.TestCheckFunc{
			tracker.record(resourceName, attrEditorPassword),
			tracker.record(resourceName, attrViewerPassword),
		}
		if withManager {
			checks = append(checks, tracker.record(resourceName, attrManagerPassword))
		}
		return resource.ComposeTestCheckFunc(checks...)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create with the block: the first trigger is only recorded, nothing is rotated.
				Config: rotationE2eHcl(instance, serviceName, "v1", internal.PasswordUserViewer),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, attrRotationTrigger, "v1"),
					mockedResetRequestsAre(rec, nil),
					record(),
				),
			},
			{
				// Changing the trigger rotates the listed user only.
				Config: rotationE2eHcl(instance, serviceName, "v2", internal.PasswordUserViewer),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, attrRotationTrigger, "v2"),
					tracker.changed(resourceName, attrViewerPassword),
					tracker.unchanged(resourceName, attrEditorPassword),
					mockedResetRequestsAre(rec, [][]string{{internal.PasswordUserViewer}}),
					record(),
				),
			},
			{
				// The new password was read back in the same apply, so a re-plan is empty.
				Config:   rotationE2eHcl(instance, serviceName, "v2", internal.PasswordUserViewer),
				PlanOnly: true,
			},
			{
				// Rotating several users at once changes each of them.
				Config: rotationE2eHcl(instance, serviceName, "v3", internal.PasswordUserViewer, internal.PasswordUserEditor),
				Check: resource.ComposeTestCheckFunc(
					tracker.changed(resourceName, attrViewerPassword),
					tracker.changed(resourceName, attrEditorPassword),
					mockedResetRequestsAre(rec, [][]string{{internal.PasswordUserViewer}, {internal.PasswordUserEditor, internal.PasswordUserViewer}}),
					record(),
				),
			},
			{
				// mc-manager rotates with the others, and only the listed users change.
				SkipFunc: func() (bool, error) { return !withManager, nil },
				Config:   rotationE2eHcl(instance, serviceName, "v4", internal.PasswordUserViewer, internal.PasswordUserManager),
				Check: resource.ComposeTestCheckFunc(
					tracker.changed(resourceName, attrViewerPassword),
					tracker.changed(resourceName, attrManagerPassword),
					tracker.unchanged(resourceName, attrEditorPassword),
					mockedResetRequestsAre(rec, [][]string{{internal.PasswordUserViewer}, {internal.PasswordUserEditor, internal.PasswordUserViewer}, {internal.PasswordUserManager, internal.PasswordUserViewer}}),
					record(),
				),
			},
			{
				// Editing users alone, with the same trigger, rotates nothing.
				Config: rotationE2eHcl(instance, serviceName, lastTrigger, internal.PasswordUserViewer),
				Check: resource.ComposeTestCheckFunc(
					tracker.unchanged(resourceName, attrViewerPassword),
					tracker.unchanged(resourceName, attrEditorPassword),
					mockedResetRequestsAre(rec, sentBeforeUsersAlone),
				),
			},
		},
	})
}

// Adding the block to an existing service records the trigger; only a later change rotates.
func TestAccBrokerPasswordRotationAddedToExistingService(t *testing.T) {
	instance := internal.NewTestInstance()
	serviceName := "E2E_PwRotation_Adopt_" + random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: e2eServiceClassId(),
		ServiceName:  serviceName,
	}
	instance.Init(params)
	var rec *internal.PasswordResetRecorder // set only against the mocked API
	if instance.IsMocked() {
		rec = instance.SetupPasswordResetMocks(params, 0, "")
	}

	resourceName := resourceTypePrefix + serviceName
	tracker := &passwordTracker{seen: map[string]string{}}
	record := func() resource.TestCheckFunc {
		return resource.ComposeTestCheckFunc(
			tracker.record(resourceName, attrEditorPassword),
			tracker.record(resourceName, attrViewerPassword),
		)
	}
	unchanged := resource.ComposeTestCheckFunc(
		tracker.unchanged(resourceName, attrEditorPassword),
		tracker.unchanged(resourceName, attrViewerPassword),
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// An ordinary service with no rotation block.
				Config: e2eServiceHcl(instance, serviceName, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, attrRotationTrigger),
					mockedResetRequestsAre(rec, nil),
					record(),
				),
			},
			{
				// Adding the block to the existing service records the trigger and rotates nothing.
				Config: rotationE2eHcl(instance, serviceName, "v1", internal.PasswordUserViewer, internal.PasswordUserEditor),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, attrRotationTrigger, "v1"),
					resource.TestCheckResourceAttr(resourceName, attrRotationUsers, "2"),
					unchanged,
					mockedResetRequestsAre(rec, nil),
				),
			},
			{
				Config:   rotationE2eHcl(instance, serviceName, "v1", internal.PasswordUserViewer, internal.PasswordUserEditor),
				PlanOnly: true,
			},
			{
				// A later change to the trigger now rotates the listed users.
				Config: rotationE2eHcl(instance, serviceName, "v2", internal.PasswordUserViewer, internal.PasswordUserEditor),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, attrRotationTrigger, "v2"),
					tracker.changed(resourceName, attrViewerPassword),
					tracker.changed(resourceName, attrEditorPassword),
					mockedResetRequestsAre(rec, [][]string{{internal.PasswordUserViewer, internal.PasswordUserEditor}}),
					record(),
				),
			},
			{
				// Removing the block rotates nothing and clears it from state.
				Config: e2eServiceHcl(instance, serviceName, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, attrRotationTrigger),
					unchanged,
					mockedResetRequestsAre(rec, [][]string{{internal.PasswordUserViewer, internal.PasswordUserEditor}}),
				),
			},
			{
				Config:   e2eServiceHcl(instance, serviceName, ""),
				PlanOnly: true,
			},
		},
	})
}
