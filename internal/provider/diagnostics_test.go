package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildErrorDetail_FullAPIError(t *testing.T) {
	apiErr := &megaport.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusBadRequest},
		Message:  "Could not find a service with UID abc-123",
		Data:     "Service has been decommissioned",
		TraceID:  "req-xyz-789",
	}

	detail := buildErrorDetail(apiErr)

	assert.Contains(t, detail, "Could not find a service with UID abc-123")
	assert.Contains(t, detail, "Detail: Service has been decommissioned")
	assert.Contains(t, detail, "HTTP Status: 400 (Bad Request)")
	assert.Contains(t, detail, "Trace ID: req-xyz-789")
	assert.Contains(t, detail, "contact Megaport support")
}

func TestBuildErrorDetail_APIErrorNoTraceID(t *testing.T) {
	apiErr := &megaport.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusConflict},
		Message:  "Resource is locked",
	}

	detail := buildErrorDetail(apiErr)

	assert.Contains(t, detail, "Resource is locked")
	assert.Contains(t, detail, "HTTP Status: 409 (Conflict)")
	assert.NotContains(t, detail, "Trace ID")
	assert.Contains(t, detail, "contact Megaport support")
}

func TestBuildErrorDetail_APIErrorNoData(t *testing.T) {
	apiErr := &megaport.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusInternalServerError},
		Message:  "Internal error",
		TraceID:  "trace-abc",
	}

	detail := buildErrorDetail(apiErr)

	assert.Contains(t, detail, "Internal error")
	assert.NotContains(t, detail, "Detail:")
	assert.Contains(t, detail, "HTTP Status: 500 (Internal Server Error)")
	assert.Contains(t, detail, "Trace ID: trace-abc")
}

func TestBuildErrorDetail_PlainError(t *testing.T) {
	err := errors.New("something went wrong")
	detail := buildErrorDetail(err)
	assert.Equal(t, "something went wrong", detail)
}

func TestBuildErrorDetail_WrappedAPIError(t *testing.T) {
	apiErr := &megaport.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusNotFound},
		Message:  "Not found",
		TraceID:  "trace-wrapped",
	}
	wrapped := fmt.Errorf("outer context: %w", apiErr)

	detail := buildErrorDetail(wrapped)

	assert.Contains(t, detail, "Not found")
	assert.Contains(t, detail, "HTTP Status: 404 (Not Found)")
	assert.Contains(t, detail, "Trace ID: trace-wrapped")
}

func TestAddAPIError(t *testing.T) {
	var diags diag.Diagnostics
	apiErr := &megaport.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusBadRequest},
		Message:  "Bad request",
		TraceID:  "trace-123",
	}

	addAPIError(&diags, "Error Creating Port (test-port)", apiErr)

	require.Len(t, diags, 1)
	assert.Equal(t, "Error Creating Port (test-port)", diags[0].Summary())
	assert.True(t, strings.Contains(diags[0].Detail(), "HTTP Status: 400"))
}

func TestSummaryHelpers(t *testing.T) {
	assert.Equal(t, "Error Creating Port (my-port)", createErrorSummary("Port", "my-port"))
	assert.Equal(t, "Error Reading MCR (mcr-uid-123)", readErrorSummary("MCR", "mcr-uid-123"))
	assert.Equal(t, "Error Updating VXC (vxc-uid)", updateErrorSummary("VXC", "vxc-uid"))
	assert.Equal(t, "Error Deleting IX (ix-uid)", deleteErrorSummary("IX", "ix-uid"))
}
