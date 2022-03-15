// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package oidc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/spacemonkeygo/monkit/v3"

	"storj.io/common/uuid"
	"storj.io/storj/satellite/console"
)

var (
	mon = monkit.Package()
)

// NewEndpoint constructs an OpenID identity provider.
func NewEndpoint(externalAddress string, oidcService *Service, service *console.Service, codeExpiry, accessTokenExpiry, refreshTokenExpiry time.Duration) *Endpoint {
	manager := manage.NewManager()

	tokenStore := oidcService.TokenStore()

	manager.MapClientStorage(oidcService.ClientStore())
	manager.MapTokenStorage(tokenStore)

	manager.MapAuthorizeGenerate(&UUIDAuthorizeGenerate{})
	manager.SetAuthorizeCodeExp(codeExpiry)

	manager.MapAccessGenerate(&MacaroonAccessGenerate{Service: service})
	manager.SetRefreshTokenCfg(&manage.RefreshingConfig{
		AccessTokenExp:    accessTokenExpiry,
		RefreshTokenExp:   refreshTokenExpiry,
		IsGenerateRefresh: refreshTokenExpiry > 0,
	})

	svr := server.NewDefaultServer(manager)

	svr.SetUserAuthorizationHandler(func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
		auth, err := console.GetAuth(r.Context())
		if err != nil {
			return "", console.ErrUnauthorized.Wrap(err)
		}

		return auth.User.ID.String(), nil
	})

	// externalAddress _should_ end with a '/' suffix based on the calling path
	return &Endpoint{
		tokenStore: tokenStore,
		service:    service,
		server:     svr,
		config: ProviderConfig{
			Issuer:      externalAddress,
			AuthURL:     externalAddress + "oauth/v2/authorize",
			TokenURL:    externalAddress + "oauth/v2/tokens",
			UserInfoURL: externalAddress + "oauth/v2/userinfo",
		},
	}
}

// Endpoint implements an OpenID Connect (OIDC) Identity Provider. It grants client applications access to resources
// in the Storj network on behalf of the end user.
//
// architecture: Endpoint
type Endpoint struct {
	tokenStore oauth2.TokenStore
	service    *console.Service
	server     *server.Server
	config     ProviderConfig
}

// WellKnownConfiguration renders the identity provider configuration that points clients to various endpoints.
func (e *Endpoint) WellKnownConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error
	defer mon.Task()(&ctx)(&err)

	data, err := json.Marshal(e.config)

	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	} else {
		http.ServeContent(w, r, "", time.Now(), bytes.NewReader(data))
	}
}

// AuthorizeUser is called from an authenticated context granting the requester access to the application. We redirect
// back to the client application with the provided state and obtained code.
func (e *Endpoint) AuthorizeUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error
	defer mon.Task()(&ctx)(&err)

	err = e.server.HandleAuthorizeRequest(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

// Tokens exchanges unexpired refresh tokens or codes provided by AuthorizeUser for the associated set of tokens.
func (e *Endpoint) Tokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error
	defer mon.Task()(&ctx)(&err)

	err = e.server.HandleTokenRequest(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

// UserInfo uses the provided access token to look up the associated user information.
func (e *Endpoint) UserInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error
	defer mon.Task()(&ctx)(&err)

	accessToken := r.Header.Get("Authorization")
	if !strings.HasPrefix(accessToken, "Bearer ") {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	accessToken = strings.TrimPrefix(accessToken, "Bearer ")

	info, err := e.tokenStore.GetByAccess(ctx, accessToken)
	if err != nil || info == nil {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	userInfo, _, err := parseScope(info.GetScope())
	if err != nil {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.FromString(info.GetUserID())
	if err != nil {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	user, err := e.service.GetUser(ctx, userID)
	if err != nil {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	if user.Status != console.Active {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	userInfo.Subject = user.ID
	userInfo.Email = user.Email
	userInfo.EmailVerified = true

	data, err := json.Marshal(userInfo)

	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	} else {
		http.ServeContent(w, r, "", time.Now(), bytes.NewReader(data))
	}
}

// ProviderConfig defines a subset of elements used by OIDC to auto-discover endpoints.
type ProviderConfig struct {
	Issuer      string `json:"issuer"`
	AuthURL     string `json:"authorization_endpoint"`
	TokenURL    string `json:"token_endpoint"`
	UserInfoURL string `json:"userinfo_endpoint"`
}

// UserInfo provides a semi-standard object for common user information. The "cubbyhole" value is used to share the
// derived encryption key between client applications. In order to obtain it, the requesting client must decrypt
// the value using the key they provided when redirecting the user to login.
type UserInfo struct {
	Subject       uuid.UUID `json:"sub"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`

	// custom values below

	Project   string   `json:"project"`
	Buckets   []string `json:"buckets"`
	Cubbyhole string   `json:"cubbyhole"`
}