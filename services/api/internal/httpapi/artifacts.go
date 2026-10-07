package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/UpgradeRail/upgraderail-console/services/api/internal/store"
	"github.com/UpgradeRail/upgraderail-console/services/shared/artifactstore"
)

// wasmMagic is the four-byte header every WASM binary module starts with.
// It is not proof the module is well-formed, but it is enough to reject
// obviously non-WASM uploads (text files, images, empty garbage) before
// they are hashed and stored. Content-Type headers are never trusted for
// this: a client can claim anything, so the bytes are checked directly.
var wasmMagic = []byte{0x00, 'a', 's', 'm'}

// MaxArtifactUploadBytes bounds a single uploaded artifact. Real Soroban
// contract WASM is generally well under 1 MiB; this leaves meaningful
// headroom while still bounding worst-case memory and disk use per upload.
// It is exported so main() can configure the storage backend with the same
// limit the HTTP layer enforces, rather than duplicating the number.
const MaxArtifactUploadBytes int64 = 8 << 20 // 8 MiB

const maxArtifactUploadBytes = MaxArtifactUploadBytes

// multipartOverheadBytes is extra budget on top of maxArtifactUploadBytes
// for multipart form framing (boundaries, headers, the field name) so a
// file at exactly the limit is not rejected for the envelope around it.
const multipartOverheadBytes int64 = 64 << 10 // 64 KiB

func registerArtifactRoutes(mux *http.ServeMux, repository Repository, artifacts artifactstore.Store) {
	mux.HandleFunc("POST /api/v1/artifacts", func(w http.ResponseWriter, request *http.Request) {
		session, ok := requireSession(w, request, repository)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, maxArtifactUploadBytes+multipartOverheadBytes)
		if err := request.ParseMultipartForm(multipartOverheadBytes); err != nil {
			errorResponse(w, http.StatusRequestEntityTooLarge, "artifact_too_large", "The upload exceeds the maximum artifact size.")
			return
		}
		defer func() {
			if request.MultipartForm != nil {
				_ = request.MultipartForm.RemoveAll()
			}
		}()
		file, _, err := request.FormFile("file")
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "missing_file", "A WASM file is required in the \"file\" field.")
			return
		}
		defer file.Close()

		peeked := bufio.NewReader(file)
		magic, err := peeked.Peek(len(wasmMagic))
		if err != nil || !bytes.Equal(magic, wasmMagic) {
			errorResponse(w, http.StatusBadRequest, "invalid_wasm", "The uploaded file does not look like a WASM module.")
			return
		}

		stored, err := artifacts.Put(request.Context(), peeked)
		switch {
		case errors.Is(err, artifactstore.ErrTooLarge):
			errorResponse(w, http.StatusRequestEntityTooLarge, "artifact_too_large", "The upload exceeds the maximum artifact size.")
			return
		case errors.Is(err, artifactstore.ErrEmpty):
			errorResponse(w, http.StatusBadRequest, "empty_artifact", "The uploaded file is empty.")
			return
		case err != nil:
			// Never log the artifact bytes or the underlying path; the error
			// itself only describes a storage failure, not file contents.
			errorResponse(w, http.StatusInternalServerError, "artifact_store_unavailable", "The artifact could not be stored.")
			return
		}

		uploader := session.Address
		record, err := repository.CreateArtifact(request.Context(), store.ArtifactInput{
			ID:          randomID(),
			SHA256:      stored.SHA256,
			StorageKey:  stored.Key,
			ContentType: stored.ContentType,
			SizeBytes:   stored.Size,
			Uploader:    &uploader,
		})
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "artifact_unavailable", "The artifact could not be recorded.")
			return
		}
		write(w, http.StatusCreated, record)
	})
}

// verifyArtifactExists confirms an artifact ID resolves to a real row and
// that the bytes still on disk hash to what was recorded when it was
// uploaded. The hash was already computed server-side at upload time
// (never trusted from the client), so this re-read is not re-establishing
// trust in a client value — it is a cheap integrity check (WASM artifacts
// are small) that catches a corrupted or tampered-with file on disk before
// a job is queued for it, rather than only discovering that when the
// worker tries to run the Engine much later.
func verifyArtifactExists(ctx context.Context, repository Repository, artifacts artifactstore.Store, id string) (store.Artifact, error) {
	record, err := repository.GetArtifact(ctx, id)
	if err != nil {
		return store.Artifact{}, fmt.Errorf("%w: %s", errArtifactNotFound, id)
	}
	if err := artifactstore.VerifyHash(ctx, artifacts, record.StorageKey, record.SHA256); err != nil {
		return store.Artifact{}, fmt.Errorf("%w: %s: %s", errArtifactCorrupt, id, err)
	}
	return record, nil
}

var (
	errArtifactNotFound = errors.New("artifact not found")
	errArtifactCorrupt  = errors.New("artifact bytes no longer match the recorded hash")
)
