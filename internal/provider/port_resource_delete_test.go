// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	megaport "github.com/megaport/megaportgo"
)

// TestPortDeleteAlwaysUsesCancelNow asserts that the SDK contract used by
// both the Single Port and LAG Port resource Delete methods — calling
// DeletePort with DeleteNow=true — results in a CANCEL_NOW action against
// the products API. Delayed cancellation (CANCEL) was removed for ports in
// megaportgo PR #155, and the provider must never issue it.
func TestPortDeleteAlwaysUsesCancelNow(t *testing.T) {
	t.Parallel()

	const portUID = "test-port-uid"

	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok","data":{}}`))
	}))
	t.Cleanup(server.Close)

	client, err := megaport.New(nil,
		megaport.WithBaseURL(server.URL),
		megaport.WithAccessToken("test-token", time.Now().Add(time.Hour)),
	)
	if err != nil {
		t.Fatalf("megaport.New: %v", err)
	}

	_, err = client.PortService.DeletePort(context.Background(), &megaport.DeletePortRequest{
		PortID:     portUID,
		DeleteNow:  true,
		SafeDelete: true,
	})
	if err != nil {
		t.Fatalf("DeletePort returned error: %v", err)
	}

	wantPath := "/v3/product/" + portUID + "/action/CANCEL_NOW"
	if receivedPath != wantPath {
		t.Fatalf("DeletePort hit unexpected path: got %q want %q", receivedPath, wantPath)
	}
}
