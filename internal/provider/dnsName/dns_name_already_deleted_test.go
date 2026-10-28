package dnsName

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-solacecloud/internal/model"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/missioncontrol"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// DATAGO-120147: the DNS name was deleted outside Terraform (console).
// The API returns 404 for the DELETE and no longer lists it on GET.
// With listStatus 404 the GET fails too, as when the service or connection endpoint is gone.
func newAlreadyDeletedDnsServer(t *testing.T, listStatus int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/dnsNames/"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"dns name not found","errorId":"x"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/dnsNames"):
			w.WriteHeader(listStatus)
			if listStatus == http.StatusOK {
				_, _ = w.Write([]byte(`{"data":[]}`))
			} else {
				_, _ = w.Write([]byte(`{"message":"not found","errorId":"x"}`))
			}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
}

func newAlreadyDeletedDnsResource(t *testing.T, url string) (*DnsNameResource, tfsdk.State) {
	ctx := context.Background()
	client, err := missioncontrol.NewClientWithResponses(url)
	if err != nil {
		t.Fatal(err)
	}
	r := &DnsNameResource{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: shared.ProviderConfig{APIClient: client, APIPollingInterval: 1}}, &resource.ConfigureResponse{})

	sch := model.DnsNameResourceSchema()
	state := tfsdk.State{Schema: sch, Raw: tftypes.NewValue(sch.Type().TerraformType(ctx), nil)}
	diags := state.Set(ctx, &model.DnsNameModel{
		Id:                   types.StringValue("dns-id"),
		ServiceId:            types.StringValue("svc-id"),
		ConnectionEndpointId: types.StringValue("ep-id"),
		DnsName:              types.StringValue("gone.example.com"),
		DomainType:           types.StringValue("CUSTOM"),
	})
	if diags.HasError() {
		t.Fatalf("state set: %v", diags)
	}
	return r, state
}

func TestDnsNameDeleteWhenAlreadyDeleted_DATAGO120147(t *testing.T) {
	srv := newAlreadyDeletedDnsServer(t, http.StatusOK)
	defer srv.Close()
	r, state := newAlreadyDeletedDnsResource(t, srv.URL)

	resp := &resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Delete on a 404 should succeed, got errors: %v", resp.Diagnostics.Errors())
	}
}

func TestDnsNameReadWhenAlreadyDeleted_DATAGO120147(t *testing.T) {
	for name, listStatus := range map[string]int{
		"dns name not listed":           http.StatusOK,
		"service or endpoint not found": http.StatusNotFound,
	} {
		t.Run(name, func(t *testing.T) {
			srv := newAlreadyDeletedDnsServer(t, listStatus)
			defer srv.Close()
			r, state := newAlreadyDeletedDnsResource(t, srv.URL)

			resp := &resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("Read of a DNS name gone from the API should not error, got: %v", resp.Diagnostics.Errors())
			}
			if !resp.State.Raw.IsNull() {
				t.Errorf("Read should remove the DNS name from state (RemoveResource), state is still set")
			}
		})
	}
}
