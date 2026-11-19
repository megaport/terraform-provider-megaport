package provider

import (
	"context"

	megaport "github.com/megaport/megaportgo"
)

// MockSolutionService is a mock of the Solution Service for testing.
type MockSolutionService struct {
	BuyDCIResult              *megaport.BuyDCIResponse
	BuyDCIErr                 error
	CapturedBuyDCIRequest     *megaport.BuyDCIRequest
	GetSolutionResult         *megaport.Solution
	GetSolutionErr            error
	GetSolutionBillableResult *megaport.SolutionBillable
	GetSolutionBillableErr    error
}

var _ megaport.SolutionService = &MockSolutionService{}

// BuyDCI returns the result with the error, as the SDK does when the DCI is ordered but not ready in time.
func (m *MockSolutionService) BuyDCI(_ context.Context, req *megaport.BuyDCIRequest) (*megaport.BuyDCIResponse, error) {
	m.CapturedBuyDCIRequest = req
	return m.BuyDCIResult, m.BuyDCIErr
}

func (m *MockSolutionService) GetSolution(_ context.Context, _ string) (*megaport.Solution, error) {
	if m.GetSolutionErr != nil {
		return nil, m.GetSolutionErr
	}
	return m.GetSolutionResult, nil
}

func (m *MockSolutionService) GetSolutionBillable(_ context.Context, _ string) (*megaport.SolutionBillable, error) {
	if m.GetSolutionBillableErr != nil {
		return nil, m.GetSolutionBillableErr
	}
	return m.GetSolutionBillableResult, nil
}
