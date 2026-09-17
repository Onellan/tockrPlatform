package httpserver

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/db/sqlite"
	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/platform/readauthority"
)

type readAuthorityHTTPFixture struct {
	store      *sqlite.Store
	handler    http.Handler
	privateKey map[string]ed25519.PrivateKey
}

func newReadAuthorityHTTPFixture(t *testing.T) readAuthorityHTTPFixture {
	t.Helper()
	ctx := context.Background()
	persistence, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatal(err)
	}
	keys := readauthority.PublicKeySet{}
	privateKeys := map[string]ed25519.PrivateKey{}
	for _, consumer := range []string{readauthority.ConsumerCTRL, readauthority.ConsumerIMS} {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		keys[consumer] = map[string]ed25519.PublicKey{"current": publicKey}
		privateKeys[consumer] = privateKey
	}
	server := NewServer(persistence, Config{AllowInsecureCookies: true, ReadAuthorityKeys: keys})
	t.Cleanup(func() { _ = persistence.Close() })
	return readAuthorityHTTPFixture{store: persistence, handler: server.Handler(), privateKey: privateKeys}
}

func TestReadAuthorityHTTPContractSupportsCTRLAndIMSAndRejectsBrowserAuth(t *testing.T) {
	fixture := newReadAuthorityHTTPFixture(t)
	body := `{"entity_kinds":["product"],"expires_in_seconds":3600}`
	for _, consumer := range []string{readauthority.ConsumerCTRL, readauthority.ConsumerIMS} {
		request := signedReadAuthorityRequest(t, http.MethodPost, "/api/v1/read-authority/snapshots", "", []byte(body), consumer, "current", fixture.privateKey[consumer], "nonce-"+consumer)
		response := httptest.NewRecorder()
		fixture.handler.ServeHTTP(response, request)
		if response.Code != http.StatusCreated || response.Header().Get(readauthority.VersionHeader) != readauthority.VersionV2 {
			t.Fatalf("%s snapshot = %d/%q", consumer, response.Code, response.Body.String())
		}
		var metadata readauthority.SnapshotMetadata
		if err := json.Unmarshal(response.Body.Bytes(), &metadata); err != nil || metadata.Consumer != consumer || metadata.ContractVersion != readauthority.VersionV2 || !metadata.Complete {
			t.Fatalf("%s metadata = %#v, err=%v", consumer, metadata, err)
		}
	}

	unauthenticated := httptest.NewRecorder()
	unauthenticatedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/read-authority/status", nil)
	unauthenticatedRequest.Header.Set(readauthority.VersionHeader, readauthority.VersionV2)
	fixture.handler.ServeHTTP(unauthenticated, unauthenticatedRequest)
	if unauthenticated.Code != http.StatusUnauthorized || strings.Contains(unauthenticated.Body.String(), "session") {
		t.Fatalf("browserless auth = %d/%q", unauthenticated.Code, unauthenticated.Body.String())
	}
	wrongVersion := signedReadAuthorityRequest(t, http.MethodGet, "/api/v1/read-authority/status", "", nil, readauthority.ConsumerCTRL, "current", fixture.privateKey[readauthority.ConsumerCTRL], "nonce-version")
	wrongVersion.Header.Set(readauthority.VersionHeader, readauthority.VersionV1)
	wrongVersionResponse := httptest.NewRecorder()
	fixture.handler.ServeHTTP(wrongVersionResponse, wrongVersion)
	if wrongVersionResponse.Code != http.StatusNotAcceptable {
		t.Fatalf("wrong version = %d/%q", wrongVersionResponse.Code, wrongVersionResponse.Body.String())
	}
}

func TestReadAuthorityHTTPReplayRotationAndCrossConsumerBinding(t *testing.T) {
	fixture := newReadAuthorityHTTPFixture(t)
	request := signedReadAuthorityRequest(t, http.MethodGet, "/api/v1/read-authority/status", "", nil, readauthority.ConsumerCTRL, "current", fixture.privateKey[readauthority.ConsumerCTRL], "nonce-replay")
	first := httptest.NewRecorder()
	fixture.handler.ServeHTTP(first, request)
	if first.Code != http.StatusOK {
		t.Fatalf("first signed status = %d/%q", first.Code, first.Body.String())
	}
	request.Body = http.NoBody
	replay := httptest.NewRecorder()
	fixture.handler.ServeHTTP(replay, request)
	if replay.Code != http.StatusUnauthorized {
		t.Fatalf("replayed signed status = %d/%q", replay.Code, replay.Body.String())
	}
	crossConsumer := signedReadAuthorityRequest(t, http.MethodGet, "/api/v1/read-authority/status", "", nil, readauthority.ConsumerIMS, "current", fixture.privateKey[readauthority.ConsumerCTRL], "nonce-cross")
	crossResponse := httptest.NewRecorder()
	fixture.handler.ServeHTTP(crossResponse, crossConsumer)
	if crossResponse.Code != http.StatusUnauthorized {
		t.Fatalf("cross-consumer key = %d/%q", crossResponse.Code, crossResponse.Body.String())
	}
}

func TestReadAuthorityHTTPAcceptsOverlappingKeyAndEnforcesConsumerRateLimit(t *testing.T) {
	fixture := newReadAuthorityHTTPFixture(t)
	overlapPublic, overlapPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keys := readauthority.PublicKeySet{
		readauthority.ConsumerCTRL: {
			"current": fixture.privateKey[readauthority.ConsumerCTRL].Public().(ed25519.PublicKey),
			"overlap": overlapPublic,
		},
	}
	server := NewServer(fixture.store, Config{AllowInsecureCookies: true, ReadAuthorityKeys: keys})
	overlap := signedReadAuthorityRequest(t, http.MethodGet, "/api/v1/read-authority/status", "", nil, readauthority.ConsumerCTRL, "overlap", overlapPrivate, "nonce-overlap")
	overlapResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(overlapResponse, overlap)
	if overlapResponse.Code != http.StatusOK {
		t.Fatalf("overlap key = %d/%q", overlapResponse.Code, overlapResponse.Body.String())
	}

	for index := 0; index < readauthority.RequestsPerMinute-1; index++ {
		request := signedReadAuthorityRequest(t, http.MethodGet, "/api/v1/read-authority/status", "", nil, readauthority.ConsumerCTRL, "current", fixture.privateKey[readauthority.ConsumerCTRL], "nonce-rate-"+strconv.Itoa(index))
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("rate request %d = %d/%q", index, response.Code, response.Body.String())
		}
	}
	limited := signedReadAuthorityRequest(t, http.MethodGet, "/api/v1/read-authority/status", "", nil, readauthority.ConsumerCTRL, "current", fixture.privateKey[readauthority.ConsumerCTRL], "nonce-rate-limit")
	limitedResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(limitedResponse, limited)
	if limitedResponse.Code != http.StatusBadRequest || !strings.Contains(limitedResponse.Body.String(), `"code":"limit_exceeded"`) {
		t.Fatalf("rate limit = %d/%q", limitedResponse.Code, limitedResponse.Body.String())
	}
}

func TestReadAuthorityHTTPSnapshotPagingChangesAndResync(t *testing.T) {
	fixture := newReadAuthorityHTTPFixture(t)
	ctx := context.Background()
	userID, err := domain.NewUserID()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.CreateUser(ctx, domain.User{ID: userID, Email: "feed@example.test", DisplayName: "Feed User", Active: true}, "correct horse battery staple"); err != nil {
		t.Fatal(err)
	}
	consumer := readauthority.ConsumerCTRL
	body := `{"entity_kinds":["product"],"expires_in_seconds":3600}`
	snapshotRequest := signedReadAuthorityRequest(t, http.MethodPost, "/api/v1/read-authority/snapshots", "", []byte(body), consumer, "current", fixture.privateKey[consumer], "nonce-snapshot")
	snapshotResponse := httptest.NewRecorder()
	fixture.handler.ServeHTTP(snapshotResponse, snapshotRequest)
	if snapshotResponse.Code != http.StatusCreated {
		t.Fatalf("snapshot create = %d/%q", snapshotResponse.Code, snapshotResponse.Body.String())
	}
	var metadata readauthority.SnapshotMetadata
	if err := json.Unmarshal(snapshotResponse.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	query := "entity_kind=product&limit=1"
	recordsRequest := signedReadAuthorityRequest(t, http.MethodGet, "/api/v1/read-authority/snapshots/"+metadata.SnapshotID+"/records", query, nil, consumer, "current", fixture.privateKey[consumer], "nonce-records")
	recordsResponse := httptest.NewRecorder()
	fixture.handler.ServeHTTP(recordsResponse, recordsRequest)
	if recordsResponse.Code != http.StatusOK || !strings.Contains(recordsResponse.Body.String(), `"provenance_kind":"migration_seed"`) {
		t.Fatalf("snapshot records = %d/%q", recordsResponse.Code, recordsResponse.Body.String())
	}

	changesQuery := "after=cur_0&limit=1"
	changesRequest := signedReadAuthorityRequest(t, http.MethodGet, "/api/v1/read-authority/changes", changesQuery, nil, consumer, "current", fixture.privateKey[consumer], "nonce-changes")
	changesResponse := httptest.NewRecorder()
	fixture.handler.ServeHTTP(changesResponse, changesRequest)
	if changesResponse.Code != http.StatusOK || !strings.Contains(changesResponse.Body.String(), `"cursor":"cur_1"`) {
		t.Fatalf("changes = %d/%q", changesResponse.Code, changesResponse.Body.String())
	}
	resyncRequest := signedReadAuthorityRequest(t, http.MethodGet, "/api/v1/read-authority/changes", "after=cur_999", nil, consumer, "current", fixture.privateKey[consumer], "nonce-resync")
	resyncResponse := httptest.NewRecorder()
	fixture.handler.ServeHTTP(resyncResponse, resyncRequest)
	if resyncResponse.Code != http.StatusConflict || !strings.Contains(resyncResponse.Body.String(), `"code":"resync_required"`) {
		t.Fatalf("resync = %d/%q", resyncResponse.Code, resyncResponse.Body.String())
	}
}

func TestReadAuthorityHTTPRejectsOversizedBodyAndRetiredKey(t *testing.T) {
	fixture := newReadAuthorityHTTPFixture(t)
	overSized := signedReadAuthorityRequest(t, http.MethodPost, "/api/v1/read-authority/snapshots", "", []byte(`{"entity_kinds":["product"],"expires_in_seconds":3600,"padding":"`+strings.Repeat("x", readauthority.RequestMaxBodyBytes)+`"}`), readauthority.ConsumerCTRL, "current", fixture.privateKey[readauthority.ConsumerCTRL], "nonce-large")
	largeResponse := httptest.NewRecorder()
	fixture.handler.ServeHTTP(largeResponse, overSized)
	if largeResponse.Code != http.StatusBadRequest {
		t.Fatalf("oversized body = %d/%q", largeResponse.Code, largeResponse.Body.String())
	}
	retired := signedReadAuthorityRequest(t, http.MethodGet, "/api/v1/read-authority/status", "", nil, readauthority.ConsumerCTRL, "retired", fixture.privateKey[readauthority.ConsumerCTRL], "nonce-retired")
	retiredResponse := httptest.NewRecorder()
	fixture.handler.ServeHTTP(retiredResponse, retired)
	if retiredResponse.Code != http.StatusUnauthorized {
		t.Fatalf("retired key = %d/%q", retiredResponse.Code, retiredResponse.Body.String())
	}
}

func signedReadAuthorityRequest(t *testing.T, method, path, query string, body []byte, consumer, keyID string, privateKey ed25519.PrivateKey, nonce string) *http.Request {
	t.Helper()
	if body == nil {
		body = []byte{}
	}
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	values, err := url.ParseQuery(query)
	if err != nil {
		t.Fatal(err)
	}
	canonicalPath := path
	if encoded := values.Encode(); encoded != "" {
		canonicalPath += "?" + encoded
	}
	canonical, err := readauthority.CanonicalRequest(method, canonicalPath, readauthority.BodyDigest(body), consumer, keyID, timestamp, nonce)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, []byte(canonical))
	request := httptest.NewRequest(method, path+func() string {
		if query == "" {
			return ""
		}
		return "?" + query
	}(), strings.NewReader(string(body)))
	request.Header.Set(readauthority.VersionHeader, readauthority.VersionV2)
	request.Header.Set(readauthority.ConsumerHeader, consumer)
	request.Header.Set(readauthority.KeyIDHeader, keyID)
	request.Header.Set(readauthority.TimestampHeader, timestamp)
	request.Header.Set(readauthority.NonceHeader, nonce)
	request.Header.Set(readauthority.SignatureHeader, base64.RawURLEncoding.EncodeToString(signature))
	return request
}
