package artifactstore

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// s3KeyPattern is the only key shape Put ever produces. Open refuses
// anything else so a corrupted database row cannot address other objects.
var s3KeyPattern = regexp.MustCompile(`^[0-9a-f]{2}/[0-9a-f]{64}$`)

// S3 stores artifacts as content-addressed objects in an S3-compatible
// bucket (Supabase Storage, Cloudflare R2, MinIO, AWS S3) using path-style
// addressing and AWS Signature Version 4. Artifacts are bounded by MaxBytes
// and small, so uploads are buffered in memory to learn the hash (and thus
// the key) before the single PUT.
type S3 struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	MaxBytes        int64
	Client          *http.Client
	// Now is overridable for tests; defaults to time.Now.
	Now func() time.Time
}

func (s S3) validate() error {
	endpoint, err := url.Parse(s.Endpoint)
	if err != nil || endpoint.Host == "" {
		return errors.New("s3 endpoint must be an absolute URL")
	}
	local := endpoint.Hostname() == "127.0.0.1" || endpoint.Hostname() == "localhost"
	if endpoint.Scheme != "https" && !(endpoint.Scheme == "http" && local) {
		return errors.New("s3 endpoint must be HTTPS or local HTTP")
	}
	if s.Region == "" || s.Bucket == "" || s.AccessKeyID == "" || s.SecretAccessKey == "" {
		return errors.New("s3 region, bucket and credentials are required")
	}
	if s.MaxBytes < 1 {
		return errors.New("maximum artifact size must be positive")
	}
	return nil
}

func (s S3) client() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (s S3) objectURL(key string) string {
	return strings.TrimRight(s.Endpoint, "/") + "/" + s.Bucket + "/" + key
}

// Put buffers input (bounded by MaxBytes), hashes it server-side, and PUTs
// it at <first two hex chars>/<sha256>. Identical content maps to the same
// key, so retries and duplicate uploads are idempotent.
func (s S3) Put(ctx context.Context, input io.Reader) (Stored, error) {
	if err := s.validate(); err != nil {
		return Stored{}, err
	}
	reader := &contextReader{ctx: ctx, reader: io.LimitReader(input, s.MaxBytes+1)}
	body, err := io.ReadAll(reader)
	if err != nil {
		return Stored{}, fmt.Errorf("read artifact: %w", err)
	}
	if int64(len(body)) > s.MaxBytes {
		return Stored{}, ErrTooLarge
	}
	if len(body) == 0 {
		return Stored{}, ErrEmpty
	}
	sum := sha256.Sum256(body)
	encoded := hex.EncodeToString(sum[:])
	key := encoded[:2] + "/" + encoded

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, s.objectURL(key), bytes.NewReader(body))
	if err != nil {
		return Stored{}, fmt.Errorf("build artifact request: %w", err)
	}
	request.Header.Set("Content-Type", "application/wasm")
	s.sign(request, encoded)
	response, err := s.client().Do(request)
	if err != nil {
		return Stored{}, fmt.Errorf("store artifact: %w", sanitizeTransportError(err))
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return Stored{}, fmt.Errorf("store artifact: object store returned status %d", response.StatusCode)
	}
	return Stored{SHA256: encoded, Key: key, Size: int64(len(body)), ContentType: "application/wasm"}, nil
}

// Open streams a stored object. Callers verify the hash with VerifyHash
// when integrity matters.
func (s S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if !s3KeyPattern.MatchString(key) {
		return nil, errors.New("artifact storage key is invalid")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(key), nil)
	if err != nil {
		return nil, fmt.Errorf("build artifact request: %w", err)
	}
	s.sign(request, emptyPayloadHash)
	response, err := s.client().Do(request)
	if err != nil {
		return nil, fmt.Errorf("open artifact: %w", sanitizeTransportError(err))
	}
	switch {
	case response.StatusCode == http.StatusNotFound:
		response.Body.Close()
		return nil, fmt.Errorf("open artifact: %w", fs.ErrNotExist)
	case response.StatusCode < 200 || response.StatusCode > 299:
		response.Body.Close()
		return nil, fmt.Errorf("open artifact: object store returned status %d", response.StatusCode)
	}
	return response.Body, nil
}

// sanitizeTransportError reduces a transport failure to a message that
// names neither the endpoint nor the object key. Context cancellation and
// deadline errors keep their identity so callers can still test for them.
func sanitizeTransportError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	default:
		return errors.New("object store unreachable")
	}
}

const emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// sign adds AWS Signature Version 4 headers to request. Every header
// already on the request (plus Host) is signed, so callers set headers
// before signing.
func (s S3) sign(request *http.Request, payloadHash string) {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	timestamp := now().UTC()
	amzDate := timestamp.Format("20060102T150405Z")
	day := timestamp.Format("20060102")
	request.Header.Set("X-Amz-Date", amzDate)
	request.Header.Set("X-Amz-Content-Sha256", payloadHash)

	headers := map[string]string{"host": request.URL.Host}
	for name, values := range request.Header {
		headers[strings.ToLower(name)] = strings.TrimSpace(strings.Join(values, ","))
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	var canonicalHeaders strings.Builder
	for _, name := range names {
		canonicalHeaders.WriteString(name + ":" + headers[name] + "\n")
	}
	signedHeaders := strings.Join(names, ";")

	canonicalRequest := strings.Join([]string{
		request.Method,
		canonicalPath(request.URL.Path),
		request.URL.RawQuery,
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")
	scope := day + "/" + s.Region + "/s3/aws4_request"
	hashed := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := strings.Join([]string{"AWS4-HMAC-SHA256", amzDate, scope, hex.EncodeToString(hashed[:])}, "\n")

	key := hmacSHA256([]byte("AWS4"+s.SecretAccessKey), day)
	key = hmacSHA256(key, s.Region)
	key = hmacSHA256(key, "s3")
	key = hmacSHA256(key, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(key, stringToSign))
	request.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", s.AccessKeyID, scope, signedHeaders, signature))
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

// canonicalPath URI-encodes each path segment per SigV4 (S3 does not
// double-encode).
func canonicalPath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = awsEscape(segment)
	}
	return strings.Join(segments, "/")
}

func awsEscape(value string) string {
	var out strings.Builder
	for _, b := range []byte(value) {
		switch {
		case b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z', b >= '0' && b <= '9', b == '-', b == '_', b == '.', b == '~':
			out.WriteByte(b)
		default:
			fmt.Fprintf(&out, "%%%02X", b)
		}
	}
	return out.String()
}
