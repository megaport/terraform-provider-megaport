// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	megaport "github.com/megaport/megaportgo"
)

func TestClientURLOverridesUnsetLeavesDefaults(t *testing.T) {
	t.Setenv("MEGAPORT_BASE_URL", "")
	t.Setenv("MEGAPORT_TOKEN_URL", "")

	if opts := clientURLOverrides(); len(opts) != 0 {
		t.Fatalf("expected no options when both vars are unset, got %d", len(opts))
	}
}

// The overrides have to beat WithEnvironment on BaseURL and supply a token URL that the host switch
// would otherwise reject. Authorizing against a local server proves both.
func TestClientURLOverridesRetargetTheClient(t *testing.T) {
	paths := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"bearer","expires_in":3600}`))
	}))
	t.Cleanup(server.Close)

	t.Setenv("MEGAPORT_BASE_URL", server.URL)
	t.Setenv("MEGAPORT_TOKEN_URL", server.URL+"/oauth2/token")

	// Mirrors the ordering in Configure: the overrides are appended last.
	clientOpts := append([]megaport.ClientOpt{
		megaport.WithEnvironment(megaport.EnvironmentStaging),
		megaport.WithCredentials("test-key", "test-secret"),
	}, clientURLOverrides()...)

	client, err := megaport.New(nil, clientOpts...)
	if err != nil {
		t.Fatalf("megaport.New: %v", err)
	}

	if got := client.BaseURL.String(); got != server.URL {
		t.Errorf("BaseURL = %q, want %q", got, server.URL)
	}

	if _, err := client.Authorize(context.Background()); err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if got := <-paths; got != "/oauth2/token" {
		t.Errorf("token request path = %q, want %q", got, "/oauth2/token")
	}
}

// Without a token URL the host switch rejects a non-standard base URL, so the pair is not optional.
func TestClientURLOverridesBaseURLAloneCannotAuthorize(t *testing.T) {
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	t.Setenv("MEGAPORT_BASE_URL", server.URL)
	t.Setenv("MEGAPORT_TOKEN_URL", "")

	clientOpts := append([]megaport.ClientOpt{
		megaport.WithCredentials("test-key", "test-secret"),
	}, clientURLOverrides()...)

	client, err := megaport.New(nil, clientOpts...)
	if err != nil {
		t.Fatalf("megaport.New: %v", err)
	}

	if _, err := client.Authorize(context.Background()); err == nil {
		t.Fatal("expected an error when only the base URL is overridden")
	}

	select {
	case path := <-requests:
		t.Errorf("token request reached %q; the host switch should reject it before any request", path)
	default:
	}
}
