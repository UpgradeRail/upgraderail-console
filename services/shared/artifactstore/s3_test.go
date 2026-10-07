package artifactstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestSigV4MatchesAWSDocumentedVector checks the signer against the worked
// GET Object example in the AWS Signature Version 4 documentation.
func TestSigV4MatchesAWSDocumentedVector(t *testing.T) {
	s := S3{
		Region:          "us-east-1",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Now:             func() time.Time { return time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC) },
	}
	request, err := http.NewRequest(http.MethodGet, "https://examplebucket.s3.amazonaws.com/test.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Range", "bytes=0-9")
	s.sign(request, emptyPayloadHash)
	want := "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request, " +
		"SignedHeaders=host;range;x-amz-content-sha256;x-amz-date, " +
		"Signature=f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41"
	if got := request.Header.Get("Authorization"); got != want {
		t.Fatalf("signature mismatch:\n got %s\nwant %s", got, want)
	}
}

type fakeBucket struct {
	mu      sync.Mutex
	objects map[string][]byte
	seen    []string
}

func (b *fakeBucket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=test-access/") || r.Header.Get("X-Amz-Date") == "" {
		http.Error(w, "unsigned", http.StatusForbidden)
		return
	}
	b.seen = append(b.seen, r.Method+" "+r.URL.Path)
	switch r.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != r.Header.Get("X-Amz-Content-Sha256") {
			http.Error(w, "payload hash mismatch", http.StatusBadRequest)
			return
		}
		b.objects[r.URL.Path] = body
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		body, ok := b.objects[r.URL.Path]
		if !ok {
			http.Error(w, "NoSuchKey", http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	}
}

func newFakeS3(t *testing.T, maxBytes int64) (S3, *fakeBucket) {
	t.Helper()
	bucket := &fakeBucket{objects: map[string][]byte{}}
	server := httptest.NewServer(bucket)
	t.Cleanup(server.Close)
	return S3{Endpoint: server.URL, Region: "eu-west-1", Bucket: "artifacts", AccessKeyID: "test-access", SecretAccessKey: "test-secret-do-not-log", MaxBytes: maxBytes}, bucket
}

func TestS3PutOpenVerifyRoundTrip(t *testing.T) {
	store, bucket := newFakeS3(t, 1024)
	ctx := context.Background()
	stored, err := store.Put(ctx, bytes.NewReader([]byte("wasm-bytes")))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("wasm-bytes"))
	if stored.SHA256 != hex.EncodeToString(sum[:]) || stored.Key != stored.SHA256[:2]+"/"+stored.SHA256 || stored.Size != 10 {
		t.Fatalf("unexpected stored descriptor: %+v", stored)
	}
	if _, ok := bucket.objects["/artifacts/"+stored.Key]; !ok {
		t.Fatalf("object not written at content-addressed key; saw %v", bucket.seen)
	}
	if err := VerifyHash(ctx, store, stored.Key, stored.SHA256); err != nil {
		t.Fatalf("expected hash to verify: %v", err)
	}
	if err := VerifyHash(ctx, store, stored.Key, "not-the-real-hash"); err == nil {
		t.Fatal("expected mismatched hash to fail")
	}
	again, err := store.Put(ctx, bytes.NewReader([]byte("wasm-bytes")))
	if err != nil || again.Key != stored.Key {
		t.Fatalf("duplicate upload must be idempotent: %+v %v", again, err)
	}
}

func TestS3RejectsOversizedAndEmpty(t *testing.T) {
	store, bucket := newFakeS3(t, 3)
	if _, err := store.Put(context.Background(), bytes.NewReader([]byte("wasm"))); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
	if _, err := store.Put(context.Background(), bytes.NewReader(nil)); !errors.Is(err, ErrEmpty) {
		t.Fatalf("expected ErrEmpty, got %v", err)
	}
	if len(bucket.seen) != 0 {
		t.Fatalf("rejected uploads must not reach the bucket, saw %v", bucket.seen)
	}
}

func TestS3MissingObjectWrapsNotExist(t *testing.T) {
	store, _ := newFakeS3(t, 1024)
	key := strings.Repeat("ab", 1) + "/" + strings.Repeat("ab", 32)
	if _, err := store.Open(context.Background(), key); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected fs.ErrNotExist, got %v", err)
	}
}

func TestS3OpenRejectsMalformedKeys(t *testing.T) {
	store, bucket := newFakeS3(t, 1024)
	for _, key := range []string{"../../etc/passwd", "/abs", "ab/../cd", "", "ab/short", "AB/" + strings.Repeat("AB", 32)} {
		if _, err := store.Open(context.Background(), key); err == nil {
			t.Fatalf("expected key %q to be rejected", key)
		}
	}
	if len(bucket.seen) != 0 {
		t.Fatalf("malformed keys must not reach the bucket, saw %v", bucket.seen)
	}
}

func TestS3ErrorsDoNotLeakSecretsOrEndpoint(t *testing.T) {
	store, _ := newFakeS3(t, 1024)
	store.SecretAccessKey = "super-secret-value"
	store.AccessKeyID = "wrong-access" // fake bucket answers 403
	_, err := store.Put(context.Background(), bytes.NewReader([]byte("wasm")))
	if err == nil {
		t.Fatal("expected a forbidden response to fail")
	}
	for _, leaked := range []string{"super-secret-value", "wrong-access", store.Endpoint} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("error leaks %q: %v", leaked, err)
		}
	}
	store.Endpoint = "http://127.0.0.1:1" // connection refused
	_, err = store.Put(context.Background(), bytes.NewReader([]byte("wasm")))
	if err == nil || strings.Contains(err.Error(), "127.0.0.1:1") {
		t.Fatalf("transport error must not echo the endpoint: %v", err)
	}
}

func TestS3ValidateRequiresSecureEndpoint(t *testing.T) {
	base := S3{Endpoint: "http://example.com", Region: "r", Bucket: "b", AccessKeyID: "a", SecretAccessKey: "s", MaxBytes: 1}
	if err := base.validate(); err == nil {
		t.Fatal("plain HTTP to a remote host must be rejected")
	}
	base.Endpoint = "https://example.com/storage/v1/s3"
	if err := base.validate(); err != nil {
		t.Fatal(err)
	}
}

func TestFromEnvSelectsBackend(t *testing.T) {
	t.Setenv("ARTIFACT_BACKEND", "")
	t.Setenv("ARTIFACT_LOCAL_DIR", t.TempDir())
	if store, err := FromEnv(10); err != nil {
		t.Fatal(err)
	} else if _, ok := store.(Filesystem); !ok {
		t.Fatalf("expected filesystem, got %T", store)
	}
	t.Setenv("ARTIFACT_BACKEND", "s3")
	for _, name := range []string{"ARTIFACT_S3_ENDPOINT", "ARTIFACT_S3_REGION", "ARTIFACT_S3_BUCKET", "ARTIFACT_S3_ACCESS_KEY_ID", "ARTIFACT_S3_SECRET_ACCESS_KEY"} {
		t.Setenv(name, "")
	}
	if _, err := FromEnv(10); err == nil {
		t.Fatal("s3 backend without configuration must fail")
	}
	t.Setenv("ARTIFACT_S3_ENDPOINT", "https://example.supabase.co/storage/v1/s3")
	t.Setenv("ARTIFACT_S3_REGION", "eu-west-1")
	t.Setenv("ARTIFACT_S3_BUCKET", "artifacts")
	t.Setenv("ARTIFACT_S3_ACCESS_KEY_ID", "id")
	t.Setenv("ARTIFACT_S3_SECRET_ACCESS_KEY", "secret")
	if store, err := FromEnv(10); err != nil {
		t.Fatal(err)
	} else if _, ok := store.(S3); !ok {
		t.Fatalf("expected s3, got %T", store)
	}
	t.Setenv("ARTIFACT_BACKEND", "carrier-pigeon")
	if _, err := FromEnv(10); err == nil {
		t.Fatal("unknown backend must fail")
	}
}

// TestS3AgainstRealServer runs against any real S3-compatible endpoint
// (for example a local MinIO) so SigV4 is verified by an independent
// implementation. It is skipped unless UPGRADERAIL_TEST_S3_ENDPOINT is set.
func TestS3AgainstRealServer(t *testing.T) {
	endpoint := os.Getenv("UPGRADERAIL_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("UPGRADERAIL_TEST_S3_ENDPOINT not set")
	}
	store := S3{
		Endpoint:        endpoint,
		Region:          os.Getenv("UPGRADERAIL_TEST_S3_REGION"),
		Bucket:          os.Getenv("UPGRADERAIL_TEST_S3_BUCKET"),
		AccessKeyID:     os.Getenv("UPGRADERAIL_TEST_S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("UPGRADERAIL_TEST_S3_SECRET_ACCESS_KEY"),
		MaxBytes:        1 << 20,
	}
	ctx := context.Background()
	payload := []byte("\x00asm\x01\x00\x00\x00 real-server round trip")
	stored, err := store.Put(ctx, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyHash(ctx, store, stored.Key, stored.SHA256); err != nil {
		t.Fatal(err)
	}
	reader, err := store.Open(ctx, stored.Key)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	got, _ := io.ReadAll(reader)
	if !bytes.Equal(got, payload) {
		t.Fatal("readback bytes differ")
	}
	missing := strings.Repeat("0", 2) + "/" + strings.Repeat("0", 64)
	if _, err := store.Open(ctx, missing); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected not-exist for missing object, got %v", err)
	}
}
