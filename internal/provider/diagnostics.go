package provider

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	megaport "github.com/megaport/megaportgo"
)

// addAPIError adds a structured error diagnostic for Megaport API errors.
// It extracts HTTP status, trace ID, and message from *megaport.ErrorResponse
// when available, falling back to err.Error() for non-API errors.
func addAPIError(diags *diag.Diagnostics, summary string, err error) {
	diags.AddError(summary, buildErrorDetail(err))
}

// buildErrorDetail extracts structured information from an error.
// For *megaport.ErrorResponse it returns a multi-line detail with message,
// optional data, HTTP status, trace ID, and a support hint.
// For other errors it returns err.Error().
func buildErrorDetail(err error) string {
	var apiErr *megaport.ErrorResponse
	if !errors.As(err, &apiErr) {
		return err.Error()
	}

	detail := apiErr.Message
	if apiErr.Data != "" {
		detail += "\n  Detail: " + apiErr.Data
	}
	if apiErr.Response != nil {
		detail += fmt.Sprintf("\n  HTTP Status: %d (%s)",
			apiErr.Response.StatusCode,
			http.StatusText(apiErr.Response.StatusCode))
	}
	if apiErr.TraceID != "" {
		detail += "\n  Trace ID: " + apiErr.TraceID
	}
	detail += "\n\n  If this issue persists, contact Megaport support with the above details."
	return detail
}

func createErrorSummary(resourceType, name string) string {
	return fmt.Sprintf("Error Creating %s (%s)", resourceType, name)
}

func readErrorSummary(resourceType, id string) string {
	return fmt.Sprintf("Error Reading %s (%s)", resourceType, id)
}

func updateErrorSummary(resourceType, id string) string {
	return fmt.Sprintf("Error Updating %s (%s)", resourceType, id)
}

func deleteErrorSummary(resourceType, id string) string {
	return fmt.Sprintf("Error Deleting %s (%s)", resourceType, id)
}
