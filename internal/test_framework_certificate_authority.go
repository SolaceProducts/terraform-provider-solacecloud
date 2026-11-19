package internal

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/jarcoal/httpmock"
)

const (
	ClientCertificateAuthorities = "clientCertificateAuthorities"
	DomainCertificateAuthorities = "domainCertificateAuthorities"
)

type CertificateAuthorityMock struct {
	serviceId     string
	mu            sync.Mutex
	cas           map[certificateAuthorityKey]map[string]json.RawMessage
	jobInProgress string
	jobs          int
	polls         map[string]int
	failNext      string
	failures      map[string]string
	writes        []CertificateAuthorityWrite
}

type certificateAuthorityKey struct {
	kind, name string
}

type CertificateAuthorityWrite struct {
	Method string
	Kind   string
	Name   string
	Body   map[string]json.RawMessage
}

var certificateAuthorityPath = regexp.MustCompile(`/(clientCertificateAuthorities|domainCertificateAuthorities)(?:/([^/]+))?$`)

func (provider *TestInstance) SetupCertificateAuthorityMocks(serviceId string) *CertificateAuthorityMock {
	mock := &CertificateAuthorityMock{
		serviceId: serviceId,
		cas:       map[certificateAuthorityKey]map[string]json.RawMessage{},
		polls:     map[string]int{},
		failures:  map[string]string{},
	}
	service := regexp.QuoteMeta(provider.baseUrl + eventBrokerServicesPath + serviceId)
	for _, method := range []string{"GET", "POST", "PATCH", "PUT", "DELETE"} {
		httpmock.RegisterResponder(method, "=~^"+service+`/(client|domain)CertificateAuthorities(/[^/]+)?$`, mock.handle)
	}
	httpmock.RegisterResponder("GET", "=~^"+service+`/operations/ca-job-\d+$`, mock.operation)
	return mock
}

func (m *CertificateAuthorityMock) Exists(kind, name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cas[certificateAuthorityKey{kind, name}] != nil
}

func (m *CertificateAuthorityMock) Put(kind, name, fieldsJson string) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(fieldsJson), &fields); err != nil {
		panic(err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := certificateAuthorityKey{kind, name}
	ca := m.cas[key]
	if ca == nil {
		ca = map[string]json.RawMessage{"id": json.RawMessage(`"` + name + `"`), "name": json.RawMessage(`"` + name + `"`)}
		m.cas[key] = ca
	}
	for field, value := range fields {
		ca[field] = value
	}
}

func (m *CertificateAuthorityMock) Remove(kind, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cas, certificateAuthorityKey{kind, name})
}

// FailNextJob makes the next accepted write's operation end as FAILED with message.
func (m *CertificateAuthorityMock) FailNextJob(message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = message
}

func (m *CertificateAuthorityMock) Writes() []CertificateAuthorityWrite {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]CertificateAuthorityWrite(nil), m.writes...)
}

func (m *CertificateAuthorityMock) handle(r *http.Request) (*http.Response, error) {
	match := certificateAuthorityPath.FindStringSubmatch(r.URL.Path)
	kind, name := match[1], match[2]
	body, _ := readJsonObject(r)
	if r.Method == http.MethodPost {
		if json.Unmarshal(body["name"], &name) != nil {
			return JsonResponder(400, `{"message": "name must be a string", "errorId": "mock"}`)(r)
		}
	}
	notFound := JsonResponder(404, `{"message": "Could not find certificate authority `+name+` on event broker service `+m.serviceId+`", "errorId": "mock"}`)

	m.mu.Lock()
	defer m.mu.Unlock()
	key := certificateAuthorityKey{kind, name}
	ca, exists := m.cas[key]
	if r.Method == http.MethodGet {
		if !exists {
			return notFound(r)
		}
		data, _ := json.Marshal(ca)
		return JsonResponder(200, `{"data": `+string(data)+`, "meta": {}}`)(r)
	}
	if m.jobInProgress != "" {
		return JsonResponder(409, `{"message": "Job `+m.jobInProgress+` is still in progress", "errorId": "mock"}`)(r)
	}
	if r.Method == http.MethodPost && exists {
		return JsonResponder(409, `{"message": "A certificate authority named `+name+` already exists on event broker service `+m.serviceId+`", "errorId": "mock"}`)(r)
	}
	if r.Method != http.MethodPost && !exists {
		return notFound(r)
	}
	if cert, sent := body["certContent"]; sent {
		if problem := pemProblem(cert); problem != "" {
			return JsonResponder(400, `{"message": "`+problem+`", "errorId": "mock", "validationDetails": {"certContent": ["`+problem+`"]}}`)(r)
		}
	}

	m.store(r.Method, key, body)
	m.writes = append(m.writes, CertificateAuthorityWrite{Method: r.Method, Kind: kind, Name: name, Body: body})

	m.jobs++
	m.jobInProgress = fmt.Sprintf("ca-job-%d", m.jobs)
	if m.failNext != "" {
		m.failures[m.jobInProgress], m.failNext = m.failNext, ""
	}
	return JsonResponder(202, serviceOperationResponse(m.jobInProgress, m.serviceId, "certificateAuthority"))(r)
}

// store applies an accepted write the way the backend stores it. The caller holds m.mu.
func (m *CertificateAuthorityMock) store(method string, key certificateAuthorityKey, body map[string]json.RawMessage) {
	if method == http.MethodDelete {
		delete(m.cas, key)
		return
	}
	if method == http.MethodPost {
		m.cas[key] = map[string]json.RawMessage{"id": json.RawMessage(`"` + key.name + `"`)}
	}
	ca := m.cas[key]
	for field, value := range body {
		switch {
		case field == "certContent": // write-only, never returned
		case field == "ocspOverrideUrl" && string(value) == `""`: // the backend stores "" as null and GET omits it
			delete(ca, field)
		default:
			ca[field] = value
		}
	}
}

func (m *CertificateAuthorityMock) operation(r *http.Request) (*http.Response, error) {
	id := path.Base(r.URL.Path)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.polls[id]++
	status, errorJson := "INPROGRESS", "null"
	if m.polls[id] > 1 {
		status = "SUCCEEDED"
		if message, failed := m.failures[id]; failed {
			status, errorJson = "FAILED", `{"message": "`+message+`"}`
		}
		if m.jobInProgress == id {
			m.jobInProgress = ""
		}
	}
	return JsonResponder(200, `{"data": {"id": "`+id+`", "type": "operation", "resourceId": "`+m.serviceId+`", "resourceType": "service", "status": "`+status+`", "error": `+errorJson+`}}`)(r)
}

// The API accepts exactly one PEM-encoded X.509 certificate.
func pemProblem(raw json.RawMessage) string {
	var content string
	if json.Unmarshal(raw, &content) != nil {
		return "certContent must be a string"
	}
	block, rest := pem.Decode([]byte(content))
	if block == nil || block.Type != "CERTIFICATE" {
		return "certContent must be a PEM-encoded X.509 certificate"
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return "certContent is not a valid X.509 certificate: " + err.Error()
	}
	if next, _ := pem.Decode(rest); next != nil {
		return "certContent must contain exactly one certificate"
	}
	return ""
}

// CertificatePem returns a new self-signed CA certificate in PEM format, valid for one day.
func CertificatePem(commonName string) string {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}
