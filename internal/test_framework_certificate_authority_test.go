package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestSetupCertificateAuthorityMocksFollowsTheApiLifecycle(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	instance := &TestInstance{baseUrl: "https://api.solace.test", mockedApi: true}
	mock := instance.SetupCertificateAuthorityMocks("svc-mock-ca")
	service := instance.baseUrl + eventBrokerServicesPath + "svc-mock-ca"
	send := func(method, path, body string) (int, string) {
		t.Helper()
		req, err := http.NewRequest(method, service+path, strings.NewReader(body))
		if err != nil {
			t.Fatalf("could not build the %s request: %v", method, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s failed: %v", method, path, err)
		}
		data, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode, string(data)
	}
	expect := func(method, path, body string, want int) string {
		t.Helper()
		got, response := send(method, path, body)
		if got != want {
			t.Fatalf("%s %s %s returned %d, want %d: %s", method, path, body, got, want, response)
		}
		return response
	}
	jobs := 0
	finishJob := func(wantStatus string) {
		t.Helper()
		jobs++
		path := "/operations/ca-job-" + strconv.Itoa(jobs)
		if response := expect("GET", path, "", 200); !strings.Contains(response, `"INPROGRESS"`) {
			t.Fatalf("first poll of %s should be in progress: %s", path, response)
		}
		if response := expect("GET", path, "", 200); !strings.Contains(response, `"`+wantStatus+`"`) {
			t.Fatalf("second poll of %s should be %s: %s", path, wantStatus, response)
		}
	}
	cert := CertificatePem("mock-ca")
	certJson, _ := json.Marshal(cert)
	twoCertsJson, _ := json.Marshal(cert + cert)
	const ca = "/clientCertificateAuthorities/corp"

	expect("GET", ca, "", 404)
	expect("PATCH", ca, `{"ocspTimeout": 9}`, 404)
	expect("POST", "/clientCertificateAuthorities", `{"name": 7}`, 400)
	expect("POST", "/clientCertificateAuthorities", `{"name": "corp", "certContent": `+string(certJson)+`, "ocspTimeout": 5}`, 202)
	expect("POST", "/domainCertificateAuthorities", `{"name": "other", "certContent": `+string(certJson)+`}`, 409)
	finishJob("SUCCEEDED")
	expect("POST", "/clientCertificateAuthorities", `{"name": "corp", "certContent": `+string(certJson)+`}`, 409)

	for _, badCert := range []string{`5`, `"not a certificate"`, string(twoCertsJson),
		`"-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"`} {
		expect("PATCH", ca, `{"certContent": `+badCert+`}`, 400)
	}

	expect("PATCH", ca, `{"ocspOverrideUrl": "http://ocsp.example.com", "certContent": `+string(certJson)+`}`, 202)
	finishJob("SUCCEEDED")
	if response := expect("GET", ca, "", 200); !strings.Contains(response, "http://ocsp.example.com") || strings.Contains(response, "BEGIN CERTIFICATE") {
		t.Fatalf("GET should return the override URL and never the certificate: %s", response)
	}
	expect("PATCH", ca, `{"ocspOverrideUrl": ""}`, 202)
	finishJob("SUCCEEDED")
	if response := expect("GET", ca, "", 200); strings.Contains(response, "ocspOverrideUrl") {
		t.Fatalf("a cleared override URL should be left out of GET: %s", response)
	}

	mock.FailNextJob("broker rejected it")
	expect("PATCH", ca, `{"ocspTimeout": 9}`, 202)
	finishJob("FAILED")

	expect("DELETE", ca, "", 202)
	finishJob("SUCCEEDED")
	if mock.Exists(ClientCertificateAuthorities, "corp") {
		t.Fatal("expected DELETE to remove the certificate authority")
	}
	if writes := mock.Writes(); len(writes) != 5 || writes[0].Method != "POST" || writes[4].Method != "DELETE" {
		t.Fatalf("expected POST, three PATCHes and DELETE to be recorded, got %+v", writes)
	}

	mock.Put(DomainCertificateAuthorities, "seeded", `{"type": "domainCertificateAuthority"}`)
	if !mock.Exists(DomainCertificateAuthorities, "seeded") {
		t.Fatal("expected Put to create the certificate authority")
	}
	mock.Remove(DomainCertificateAuthorities, "seeded")
	expect("GET", "/domainCertificateAuthorities/seeded", "", 404)
}
