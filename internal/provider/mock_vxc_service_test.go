package provider

import (
	"context"

	megaport "github.com/megaport/megaportgo"
)

// MockVXCService is a test double for megaport.VXCService.
type MockVXCService struct {
	GetVXCResult              *megaport.VXC
	GetVXCErr                 error
	GetVXCFunc                func(ctx context.Context, id string) (*megaport.VXC, error)
	ListVXCsResult            []*megaport.VXC
	ListVXCsErr               error
	ListVXCResourceTagsFunc   func(ctx context.Context, vxcID string) (map[string]string, error)
	ListVXCResourceTagsErr    error
	ListVXCResourceTagsResult map[string]string
	CapturedResourceTagVXCUID string
}

func (m *MockVXCService) GetVXC(ctx context.Context, id string) (*megaport.VXC, error) {
	if m.GetVXCFunc != nil {
		return m.GetVXCFunc(ctx, id)
	}
	if m.GetVXCErr != nil {
		return nil, m.GetVXCErr
	}
	return m.GetVXCResult, nil
}

func (m *MockVXCService) ListVXCs(_ context.Context, _ *megaport.ListVXCsRequest) ([]*megaport.VXC, error) {
	if m.ListVXCsErr != nil {
		return nil, m.ListVXCsErr
	}
	if m.ListVXCsResult != nil {
		return m.ListVXCsResult, nil
	}
	return []*megaport.VXC{}, nil
}

func (m *MockVXCService) ListVXCResourceTags(_ context.Context, vxcID string) (map[string]string, error) {
	m.CapturedResourceTagVXCUID = vxcID
	if m.ListVXCResourceTagsFunc != nil {
		return m.ListVXCResourceTagsFunc(context.Background(), vxcID)
	}
	if m.ListVXCResourceTagsErr != nil {
		return nil, m.ListVXCResourceTagsErr
	}
	if m.ListVXCResourceTagsResult != nil {
		return m.ListVXCResourceTagsResult, nil
	}
	return nil, nil
}

func (m *MockVXCService) BuyVXC(_ context.Context, _ *megaport.BuyVXCRequest) (*megaport.BuyVXCResponse, error) {
	return nil, nil
}

func (m *MockVXCService) ValidateVXCOrder(_ context.Context, _ *megaport.BuyVXCRequest) error {
	return nil
}

func (m *MockVXCService) DeleteVXC(_ context.Context, _ string, _ *megaport.DeleteVXCRequest) error {
	return nil
}

func (m *MockVXCService) UpdateVXC(_ context.Context, _ string, _ *megaport.UpdateVXCRequest) (*megaport.VXC, error) {
	return nil, nil
}

func (m *MockVXCService) LookupPartnerPorts(_ context.Context, _ *megaport.LookupPartnerPortsRequest) (*megaport.LookupPartnerPortsResponse, error) {
	return nil, nil
}

func (m *MockVXCService) ListPartnerPorts(_ context.Context, _ *megaport.ListPartnerPortsRequest) (*megaport.ListPartnerPortsResponse, error) {
	return nil, nil
}

func (m *MockVXCService) UpdateVXCResourceTags(_ context.Context, _ string, _ map[string]string) error {
	return nil
}
