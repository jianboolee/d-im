package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	userRepository "d-im/internal/user/repository"
	"d-im/pkg/crypto"
	"d-im/pkg/model"
)

type userSnapshotRepoStub struct {
	got *model.User
	err error
}

func (r *userSnapshotRepoStub) UpsertSnapshot(_ context.Context, user *model.User) error {
	r.got = user
	return r.err
}

func TestUserSyncHandlerPutUserSnapshot(t *testing.T) {
	repo := &userSnapshotRepoStub{}
	handler := newTestUserSyncHandler(repo)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/management/users/user-a", strings.NewReader(`{
		"nickname":"Alice",
		"avatar_url":"https://example.com/avatar.png",
		"status":"active",
		"version":2,
		"ext":{"tenant":"acme"}
	}`))
	req.SetPathValue("userID", "user-a")
	req.Header.Set("X-API-Key", "test-api-key")
	recorder := httptest.NewRecorder()

	handler.PutUserSnapshot(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if repo.got == nil || repo.got.ID != "user-a" || repo.got.Version != 2 || repo.got.Status != "active" {
		t.Fatalf("unexpected snapshot: %+v", repo.got)
	}
}

func TestUserSyncHandlerRejectsInvalidSnapshot(t *testing.T) {
	repo := &userSnapshotRepoStub{}
	handler := newTestUserSyncHandler(repo)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/management/users/user-a", strings.NewReader(`{"status":"deleted","version":0}`))
	req.SetPathValue("userID", "user-a")
	req.Header.Set("X-API-Key", "test-api-key")
	recorder := httptest.NewRecorder()

	handler.PutUserSnapshot(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if repo.got != nil {
		t.Fatalf("repository should not be called: %+v", repo.got)
	}
}

func TestUserSyncHandlerRejectsStaleVersion(t *testing.T) {
	repo := &userSnapshotRepoStub{err: userRepository.ErrStaleUserVersion}
	handler := newTestUserSyncHandler(repo)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/management/users/user-a", strings.NewReader(`{"nickname":"Alice","status":"active","version":1}`))
	req.SetPathValue("userID", "user-a")
	req.Header.Set("X-API-Key", "test-api-key")
	recorder := httptest.NewRecorder()

	handler.PutUserSnapshot(recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}
}

func newTestUserSyncHandler(repo *userSnapshotRepoStub) *UserSyncHandler {
	jwtManager := crypto.NewJWTManager("test-secret", time.Minute, time.Hour, time.Minute, "test-api-key")
	return NewUserSyncHandler(jwtManager, repo)
}

var _ userRepo = (*userSnapshotRepoStub)(nil)
