package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	authsvc "github.com/friend-diaries/backend/internal/application/auth"
	"github.com/friend-diaries/backend/internal/interfaces/dto"
	"github.com/friend-diaries/backend/internal/interfaces/http/middleware"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
	"github.com/friend-diaries/backend/pkg/token"
)

type AuthHandler struct {
	authService  *authsvc.Service
	tokenManager *token.Manager
}

func NewAuthHandler(authService *authsvc.Service, tokenManager *token.Manager) *AuthHandler {
	return &AuthHandler{authService: authService, tokenManager: tokenManager}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, err)
		return
	}

	u, err := h.authService.Register(r.Context(), authsvc.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, dto.ToUserResponse(u))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, err)
		return
	}

	pair, err := h.authService.Login(r.Context(), authsvc.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	h.setRefreshCookie(w, pair.RefreshToken)

	response.OK(w, dto.AuthResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: nativeRefreshToken(r, pair.RefreshToken),
		User:         dto.ToUserResponse(pair.User),
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := h.refreshTokenFromRequest(r)
	if err != nil {
		response.Error(w, apperror.Unauthorized("missing refresh token"))
		return
	}

	pair, err := h.authService.Refresh(r.Context(), refreshToken)
	if err != nil {
		h.clearRefreshCookie(w)
		response.Error(w, err)
		return
	}

	h.setRefreshCookie(w, pair.RefreshToken)

	response.OK(w, dto.AuthResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: nativeRefreshToken(r, pair.RefreshToken),
		User:         dto.ToUserResponse(pair.User),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.clearRefreshCookie(w)
	response.NoContent(w)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}
	_ = userID
	// User already validated by auth middleware; return from context in real impl
	response.OK(w, map[string]string{"user_id": userID.String()})
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   int(h.tokenManager.RefreshExpiry().Seconds()),
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (h *AuthHandler) refreshTokenFromRequest(r *http.Request) (string, error) {
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	if token := r.Header.Get("X-Refresh-Token"); isNativeClient(r) && token != "" {
		return token, nil
	}

	if !isNativeClient(r) || r.Body == nil {
		return "", errors.New("missing refresh token")
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if req.RefreshToken == "" {
		return "", errors.New("missing refresh token")
	}
	return req.RefreshToken, nil
}

func nativeRefreshToken(r *http.Request, token string) *string {
	if !isNativeClient(r) {
		return nil
	}
	return &token
}

func isNativeClient(r *http.Request) bool {
	return r.Header.Get("X-Friend-Diaries-Client") == "tauri"
}
