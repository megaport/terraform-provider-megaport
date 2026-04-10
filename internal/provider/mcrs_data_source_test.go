package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	megaport "github.com/megaport/megaportgo"
)

// MockMCRService is a mock of the MCR service for testing
type MockMCRService struct {
	ListMCRsResult            []*megaport.MCR
	ListMCRsErr               error
	GetMCRResult              *megaport.MCR
	GetMCRErr                 error
	ListMCRResourceTagsFunc   func(ctx context.Context, mcrID string) (map[string]string, error)
	ListMCRResourceTagsErr    error
	ListMCRResourceTagsResult map[string]string
	CapturedResourceTagMCRUID string
}

func (m *MockMCRService) ListMCRs(ctx context.Context, req *megaport.ListMCRsRequest) ([]*megaport.MCR, error) {
	if m.ListMCRsErr != nil {
		return nil, m.ListMCRsErr
	}
	if m.ListMCRsResult != nil {
		return m.ListMCRsResult, nil
	}
	return []*megaport.MCR{}, nil
}

func (m *MockMCRService) GetMCR(ctx context.Context, mcrId string) (*megaport.MCR, error) {
	if m.GetMCRErr != nil {
		return nil, m.GetMCRErr
	}
	return m.GetMCRResult, nil
}

func (m *MockMCRService) ListMCRResourceTags(ctx context.Context, mcrID string) (map[string]string, error) {
	m.CapturedResourceTagMCRUID = mcrID
	if m.ListMCRResourceTagsFunc != nil {
		return m.ListMCRResourceTagsFunc(ctx, mcrID)
	}
	if m.ListMCRResourceTagsErr != nil {
		return nil, m.ListMCRResourceTagsErr
	}
	if m.ListMCRResourceTagsResult != nil {
		return m.ListMCRResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

// Implement other required methods of the MCRService interface with minimal stubs
func (m *MockMCRService) BuyMCR(ctx context.Context, req *megaport.BuyMCRRequest) (*megaport.BuyMCRResponse, error) {
	return nil, nil
}

func (m *MockMCRService) ValidateMCROrder(ctx context.Context, req *megaport.BuyMCRRequest) error {
	return nil
}

func (m *MockMCRService) CreatePrefixFilterList(ctx context.Context, req *megaport.CreateMCRPrefixFilterListRequest) (*megaport.CreateMCRPrefixFilterListResponse, error) {
	return nil, nil
}

func (m *MockMCRService) ListMCRPrefixFilterLists(ctx context.Context, mcrId string) ([]*megaport.PrefixFilterList, error) {
	return nil, nil
}

func (m *MockMCRService) GetMCRPrefixFilterList(ctx context.Context, mcrID string, prefixFilterListID int) (*megaport.MCRPrefixFilterList, error) {
	return nil, nil
}

func (m *MockMCRService) ModifyMCRPrefixFilterList(ctx context.Context, mcrID string, prefixFilterListID int, prefixFilterList *megaport.MCRPrefixFilterList) (*megaport.ModifyMCRPrefixFilterListResponse, error) {
	return nil, nil
}

func (m *MockMCRService) DeleteMCRPrefixFilterList(ctx context.Context, mcrID string, prefixFilterListID int) (*megaport.DeleteMCRPrefixFilterListResponse, error) {
	return nil, nil
}

func (m *MockMCRService) ModifyMCR(ctx context.Context, req *megaport.ModifyMCRRequest) (*megaport.ModifyMCRResponse, error) {
	return nil, nil
}

func (m *MockMCRService) DeleteMCR(ctx context.Context, req *megaport.DeleteMCRRequest) (*megaport.DeleteMCRResponse, error) {
	return nil, nil
}

func (m *MockMCRService) RestoreMCR(ctx context.Context, mcrId string) (*megaport.RestoreMCRResponse, error) {
	return nil, nil
}

func (m *MockMCRService) UpdateMCRResourceTags(ctx context.Context, mcrID string, tags map[string]string) error {
	return nil
}

func (m *MockMCRService) GetMCRPrefixFilterLists(ctx context.Context, mcrId string) ([]*megaport.PrefixFilterList, error) {
	return nil, nil
}

func (m *MockMCRService) UpdateMCRWithAddOn(ctx context.Context, mcrID string, req megaport.MCRAddOnRequest) error {
	return nil
}

func (m *MockMCRService) UpdateMCRIPsecAddOn(ctx context.Context, mcrID, addOnUID string, tunnelCount int) error {
	return nil
}

func TestReadMCRs_ListAll(t *testing.T) {
	mockMCRService := &MockMCRService{
		ListMCRsResult: []*megaport.MCR{
			{UID: "mcr-1", Name: "MCR One"},
			{UID: "mcr-2", Name: "MCR Two"},
		},
	}
	mockClient := &megaport.Client{MCRService: mockMCRService}
	ds := &mcrsDataSource{client: mockClient}

	mcrs, err := ds.client.MCRService.ListMCRs(context.Background(), &megaport.ListMCRsRequest{IncludeInactive: false})
	assert.NoError(t, err)
	assert.Len(t, mcrs, 2)
	assert.Equal(t, "mcr-1", mcrs[0].UID)
	assert.Equal(t, "mcr-2", mcrs[1].UID)
}

func TestReadMCRs_GetByUID(t *testing.T) {
	mockMCRService := &MockMCRService{
		GetMCRResult: &megaport.MCR{UID: "mcr-1", Name: "MCR One"},
	}
	mockClient := &megaport.Client{MCRService: mockMCRService}
	ds := &mcrsDataSource{client: mockClient}

	mcr, err := ds.client.MCRService.GetMCR(context.Background(), "mcr-1")
	assert.NoError(t, err)
	assert.Equal(t, "mcr-1", mcr.UID)
	assert.Equal(t, "MCR One", mcr.Name)
}

func TestReadMCRs_ListError(t *testing.T) {
	mockMCRService := &MockMCRService{
		ListMCRsErr: errors.New("API error"),
	}
	mockClient := &megaport.Client{MCRService: mockMCRService}
	ds := &mcrsDataSource{client: mockClient}

	mcrs, err := ds.client.MCRService.ListMCRs(context.Background(), &megaport.ListMCRsRequest{IncludeInactive: false})
	assert.Error(t, err)
	assert.Nil(t, mcrs)
	assert.Contains(t, err.Error(), "API error")
}

func TestReadMCRs_GetByUIDError(t *testing.T) {
	mockMCRService := &MockMCRService{
		GetMCRErr: errors.New("MCR not found"),
	}
	mockClient := &megaport.Client{MCRService: mockMCRService}
	ds := &mcrsDataSource{client: mockClient}

	mcr, err := ds.client.MCRService.GetMCR(context.Background(), "mcr-nonexistent")
	assert.Error(t, err)
	assert.Nil(t, mcr)
	assert.Contains(t, err.Error(), "MCR not found")
}

func TestFromAPIMCRDetail(t *testing.T) {
	t.Run("Maps all fields correctly", func(t *testing.T) {
		mcr := &megaport.MCR{
			UID:                   "mcr-abc-123",
			Name:                  "My Test MCR",
			ProvisioningStatus:    "LIVE",
			CreatedBy:             "user@example.com",
			CostCentre:            "CC-001",
			PortSpeed:             5000,
			Market:                "Sydney",
			LocationID:            65,
			CompanyUID:            "company-abc",
			CompanyName:           "Acme Corp",
			ContractTermMonths:    12,
			DiversityZone:         "zone-a",
			SecondaryName:         "Secondary MCR",
			VXCPermitted:          true,
			VXCAutoApproval:       false,
			MarketplaceVisibility: true,
			Locked:                false,
			AdminLocked:           true,
			Cancelable:            true,
			AttributeTags: map[string]string{
				"tag1": "val1",
				"tag2": "val2",
			},
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64512,
				},
			},
		}

		tags := map[string]string{
			"env":   "production",
			"owner": "team-a",
		}

		detail := fromAPIMCRDetail(mcr, tags)

		assert.Equal(t, "mcr-abc-123", detail.UID.ValueString())
		assert.Equal(t, "My Test MCR", detail.Name.ValueString())
		assert.Equal(t, "LIVE", detail.ProvisioningStatus.ValueString())
		assert.Equal(t, "user@example.com", detail.CreatedBy.ValueString())
		assert.Equal(t, "CC-001", detail.CostCentre.ValueString())
		assert.Equal(t, int64(5000), detail.PortSpeed.ValueInt64())
		assert.Equal(t, "Sydney", detail.Market.ValueString())
		assert.Equal(t, int64(65), detail.LocationID.ValueInt64())
		assert.Equal(t, "company-abc", detail.CompanyUID.ValueString())
		assert.Equal(t, "Acme Corp", detail.CompanyName.ValueString())
		assert.Equal(t, int64(12), detail.ContractTermMonths.ValueInt64())
		assert.Equal(t, "zone-a", detail.DiversityZone.ValueString())
		assert.Equal(t, "Secondary MCR", detail.SecondaryName.ValueString())
		assert.Equal(t, true, detail.VXCPermitted.ValueBool())
		assert.Equal(t, false, detail.VXCAutoApproval.ValueBool())
		assert.Equal(t, true, detail.MarketplaceVisibility.ValueBool())
		assert.Equal(t, int64(64512), detail.ASN.ValueInt64())
		assert.Equal(t, false, detail.Locked.ValueBool())
		assert.Equal(t, true, detail.AdminLocked.ValueBool())
		assert.Equal(t, true, detail.Cancelable.ValueBool())
		assert.False(t, detail.AttributeTags.IsNull())
		assert.False(t, detail.ResourceTags.IsNull())
	})

	t.Run("Nil time fields produce empty strings", func(t *testing.T) {
		mcr := &megaport.MCR{
			UID:               "mcr-nil-times",
			CreateDate:        nil,
			LiveDate:          nil,
			TerminateDate:     nil,
			ContractStartDate: nil,
			ContractEndDate:   nil,
		}

		detail := fromAPIMCRDetail(mcr, nil)

		assert.Equal(t, "", detail.CreateDate.ValueString())
		assert.Equal(t, "", detail.LiveDate.ValueString())
		assert.Equal(t, "", detail.TerminateDate.ValueString())
		assert.Equal(t, "", detail.ContractStartDate.ValueString())
		assert.Equal(t, "", detail.ContractEndDate.ValueString())
	})

	t.Run("Nil tags produce null maps", func(t *testing.T) {
		mcr := &megaport.MCR{
			UID:           "mcr-nil-tags",
			AttributeTags: nil,
		}

		detail := fromAPIMCRDetail(mcr, nil)

		assert.True(t, detail.AttributeTags.IsNull())
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Empty resource tags produce null map", func(t *testing.T) {
		mcr := &megaport.MCR{
			UID: "mcr-empty-tags",
		}

		detail := fromAPIMCRDetail(mcr, map[string]string{})

		assert.True(t, detail.ResourceTags.IsNull())
	})
}

// Ensure mcrsModel compiles with the new schema (no filter/tags fields).
func TestMCRsModel_Structure(t *testing.T) {
	model := mcrsModel{
		ProductUID: types.StringValue("mcr-123"),
		MCRs:       types.ListNull(types.ObjectType{AttrTypes: mcrDetailAttrs}),
	}
	assert.Equal(t, "mcr-123", model.ProductUID.ValueString())
	assert.True(t, model.MCRs.IsNull())
}
