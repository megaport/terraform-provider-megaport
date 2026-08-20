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

	opts, err := clientURLOverrides()
	if err != nil {
		t.Fatalf("clientURLOverrides: %v", err)
	}
	if len(opts) != 0 {
		t.Fatalf("expected no options when both vars are unset, got %d", len(opts))
	}
}

// Setting one var without the other splits authentication from the API calls, so both directions
// have to be rejected before a client is built.
func TestClientURLOverridesRejectAnUnpairedVar(t *testing.T) {
	for name, env := range map[string][2]string{
		"base URL only":  {"https://api-ci.megaport.com", ""},
		"token URL only": {"", "https://api-ci.megaport.com/oauth2/token"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("MEGAPORT_BASE_URL", env[0])
			t.Setenv("MEGAPORT_TOKEN_URL", env[1])

			if _, err := clientURLOverrides(); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
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

	urlOverrides, err := clientURLOverrides()
	if err != nil {
		t.Fatalf("clientURLOverrides: %v", err)
	}

	// Mirrors the ordering in Configure: the overrides are appended last.
	clientOpts := append([]megaport.ClientOpt{
		megaport.WithEnvironment(megaport.EnvironmentStaging),
		megaport.WithCredentials("test-key", "test-secret"),
	}, urlOverrides...)

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
