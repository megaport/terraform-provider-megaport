package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	megaport "github.com/megaport/megaportgo"
)

// MockMVEService is a mock of the MVE service for testing
type MockMVEService struct {
	ListMVEsResult            []*megaport.MVE
	ListMVEsErr               error
	GetMVEResult              *megaport.MVE
	GetMVEErr                 error
	ListMVEResourceTagsFunc   func(ctx context.Context, mveID string) (map[string]string, error)
	ListMVEResourceTagsErr    error
	ListMVEResourceTagsResult map[string]string
	CapturedResourceTagMVEUID string
}

func (m *MockMVEService) ListMVEs(ctx context.Context, req *megaport.ListMVEsRequest) ([]*megaport.MVE, error) {
	if m.ListMVEsErr != nil {
		return nil, m.ListMVEsErr
	}
	if m.ListMVEsResult != nil {
		return m.ListMVEsResult, nil
	}
	return []*megaport.MVE{}, nil
}

func (m *MockMVEService) GetMVE(ctx context.Context, mveId string) (*megaport.MVE, error) {
	if m.GetMVEErr != nil {
		return nil, m.GetMVEErr
	}
	return m.GetMVEResult, nil
}

func (m *MockMVEService) ListMVEResourceTags(ctx context.Context, mveID string) (map[string]string, error) {
	m.CapturedResourceTagMVEUID = mveID
	if m.ListMVEResourceTagsFunc != nil {
		return m.ListMVEResourceTagsFunc(ctx, mveID)
	}
	if m.ListMVEResourceTagsErr != nil {
		return nil, m.ListMVEResourceTagsErr
	}
	if m.ListMVEResourceTagsResult != nil {
		return m.ListMVEResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

// Implement other required methods of the MVEService interface with minimal stubs
func (m *MockMVEService) BuyMVE(ctx context.Context, req *megaport.BuyMVERequest) (*megaport.BuyMVEResponse, error) {
	return nil, nil
}

func (m *MockMVEService) ValidateMVEOrder(ctx context.Context, req *megaport.BuyMVERequest) error {
	return nil
}

func (m *MockMVEService) ModifyMVE(ctx context.Context, req *megaport.ModifyMVERequest) (*megaport.ModifyMVEResponse, error) {
	return nil, nil
}

func (m *MockMVEService) DeleteMVE(ctx context.Context, req *megaport.DeleteMVERequest) (*megaport.DeleteMVEResponse, error) {
	return nil, nil
}

func (m *MockMVEService) ListMVEImages(ctx context.Context) ([]*megaport.MVEImage, error) {
	return nil, nil
}

func (m *MockMVEService) ListAvailableMVESizes(ctx context.Context) ([]*megaport.MVESize, error) {
	return nil, nil
}

func (m *MockMVEService) UpdateMVEResourceTags(ctx context.Context, mveID string, tags map[string]string) error {
	return nil
}

func TestReadMVEs_ListAll(t *testing.T) {
	mockMVEService := &MockMVEService{
		ListMVEsResult: []*megaport.MVE{
			{UID: "mve-1", Name: "MVE One"},
			{UID: "mve-2", Name: "MVE Two"},
		},
	}
	mockClient := &megaport.Client{MVEService: mockMVEService}
	ds := &mvesDataSource{client: mockClient}

	mves, err := ds.client.MVEService.ListMVEs(context.Background(), &megaport.ListMVEsRequest{IncludeInactive: false})
	assert.NoError(t, err)
	assert.Len(t, mves, 2)
	assert.Equal(t, "mve-1", mves[0].UID)
	assert.Equal(t, "mve-2", mves[1].UID)
}

func TestReadMVEs_GetByUID(t *testing.T) {
	mockMVEService := &MockMVEService{
		GetMVEResult: &megaport.MVE{UID: "mve-1", Name: "MVE One"},
	}
	mockClient := &megaport.Client{MVEService: mockMVEService}
	ds := &mvesDataSource{client: mockClient}

	mve, err := ds.client.MVEService.GetMVE(context.Background(), "mve-1")
	assert.NoError(t, err)
	assert.Equal(t, "mve-1", mve.UID)
	assert.Equal(t, "MVE One", mve.Name)
}

func TestReadMVEs_ListError(t *testing.T) {
	mockMVEService := &MockMVEService{
		ListMVEsErr: errors.New("API error"),
	}
	mockClient := &megaport.Client{MVEService: mockMVEService}
	ds := &mvesDataSource{client: mockClient}

	mves, err := ds.client.MVEService.ListMVEs(context.Background(), &megaport.ListMVEsRequest{IncludeInactive: false})
	assert.Error(t, err)
	assert.Nil(t, mves)
	assert.Contains(t, err.Error(), "API error")
}

func TestReadMVEs_GetByUIDError(t *testing.T) {
	mockMVEService := &MockMVEService{
		GetMVEErr: errors.New("MVE not found"),
	}
	mockClient := &megaport.Client{MVEService: mockMVEService}
	ds := &mvesDataSource{client: mockClient}

	mve, err := ds.client.MVEService.GetMVE(context.Background(), "mve-nonexistent")
	assert.Error(t, err)
	assert.Nil(t, mve)
	assert.Contains(t, err.Error(), "MVE not found")
}

func TestFromAPIMVEDetail(t *testing.T) {
	t.Run("Maps all fields correctly", func(t *testing.T) {
		mve := &megaport.MVE{
			UID:                   "mve-abc-123",
			Name:                  "My Test MVE",
			ProvisioningStatus:    "LIVE",
			CreatedBy:             "user@example.com",
			Market:                "Sydney",
			LocationID:            65,
			MarketplaceVisibility: true,
			VXCPermitted:          true,
			VXCAutoApproval:       false,
			SecondaryName:         "Secondary MVE",
			CompanyUID:            "company-abc",
			CompanyName:           "Acme Corp",
			CostCentre:            "CC-001",
			ContractTermMonths:    12,
			Locked:                false,
			AdminLocked:           true,
			Cancelable:            true,
			Vendor:                "cisco",
			Size:                  "MEDIUM",
			DiversityZone:         "zone-a",
			AttributeTags: map[string]string{
				"tag1": "val1",
				"tag2": "val2",
			},
		}

		tags := map[string]string{
			"env":   "production",
			"owner": "team-a",
		}

		detail, diags := fromAPIMVEDetail(mve, tags)
		assert.False(t, diags.HasError())

		assert.Equal(t, "mve-abc-123", detail.UID.ValueString())
		assert.Equal(t, "My Test MVE", detail.Name.ValueString())
		assert.Equal(t, "LIVE", detail.ProvisioningStatus.ValueString())
		assert.Equal(t, "user@example.com", detail.CreatedBy.ValueString())
		assert.Equal(t, "Sydney", detail.Market.ValueString())
		assert.Equal(t, int64(65), detail.LocationID.ValueInt64())
		assert.Equal(t, true, detail.MarketplaceVisibility.ValueBool())
		assert.Equal(t, true, detail.VXCPermitted.ValueBool())
		assert.Equal(t, false, detail.VXCAutoApproval.ValueBool())
		assert.Equal(t, "Secondary MVE", detail.SecondaryName.ValueString())
		assert.Equal(t, "company-abc", detail.CompanyUID.ValueString())
		assert.Equal(t, "Acme Corp", detail.CompanyName.ValueString())
		assert.Equal(t, "CC-001", detail.CostCentre.ValueString())
		assert.Equal(t, int64(12), detail.ContractTermMonths.ValueInt64())
		assert.Equal(t, false, detail.Locked.ValueBool())
		assert.Equal(t, true, detail.AdminLocked.ValueBool())
		assert.Equal(t, true, detail.Cancelable.ValueBool())
		assert.Equal(t, "cisco", detail.Vendor.ValueString())
		assert.Equal(t, "MEDIUM", detail.Size.ValueString())
		assert.Equal(t, "zone-a", detail.DiversityZone.ValueString())
		assert.False(t, detail.AttributeTags.IsNull())
		assert.False(t, detail.ResourceTags.IsNull())
	})

	t.Run("Nil time fields produce null values", func(t *testing.T) {
		mve := &megaport.MVE{
			UID:               "mve-nil-times",
			CreateDate:        nil,
			LiveDate:          nil,
			TerminateDate:     nil,
			ContractStartDate: nil,
			ContractEndDate:   nil,
		}

		detail, diags := fromAPIMVEDetail(mve, nil)
		assert.False(t, diags.HasError())

		assert.True(t, detail.CreateDate.IsNull())
		assert.True(t, detail.LiveDate.IsNull())
		assert.True(t, detail.TerminateDate.IsNull())
		assert.True(t, detail.ContractStartDate.IsNull())
		assert.True(t, detail.ContractEndDate.IsNull())
	})

	t.Run("Nil tags produce null maps", func(t *testing.T) {
		mve := &megaport.MVE{
			UID:           "mve-nil-tags",
			AttributeTags: nil,
		}

		detail, diags := fromAPIMVEDetail(mve, nil)
		assert.False(t, diags.HasError())

		assert.True(t, detail.AttributeTags.IsNull())
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Empty non-nil resource tags produce empty map (fetched but empty)", func(t *testing.T) {
		mve := &megaport.MVE{
			UID: "mve-empty-tags",
		}

		detail, diags := fromAPIMVEDetail(mve, map[string]string{})
		assert.False(t, diags.HasError())

		assert.False(t, detail.ResourceTags.IsNull())
		assert.Equal(t, 0, len(detail.ResourceTags.Elements()))
	})
}

func TestFromAPIMVEDetail_ResourceTagsOptIn(t *testing.T) {
	t.Run("Tags nil when not fetched", func(t *testing.T) {
		mve := &megaport.MVE{UID: "mve-1"}
		detail, diags := fromAPIMVEDetail(mve, nil)
		assert.False(t, diags.HasError())
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Tags populated when fetched", func(t *testing.T) {
		mve := &megaport.MVE{UID: "mve-1"}
		tags := map[string]string{"env": "prod"}
		detail, diags := fromAPIMVEDetail(mve, tags)
		assert.False(t, diags.HasError())
		assert.False(t, detail.ResourceTags.IsNull())
	})
}

func TestReadMVEs_TagsNotFetchedByDefault(t *testing.T) {
	tagsCalled := false
	mockMVEService := &MockMVEService{
		ListMVEsResult: []*megaport.MVE{
			{UID: "mve-1", Name: "MVE One"},
		},
		ListMVEResourceTagsFunc: func(_ context.Context, _ string) (map[string]string, error) {
			tagsCalled = true
			return map[string]string{"env": "test"}, nil
		},
	}
	mockClient := &megaport.Client{MVEService: mockMVEService}

	// Simulate Read path: tags should not be fetched when fetchTags is false
	mves, err := mockClient.MVEService.ListMVEs(context.Background(), &megaport.ListMVEsRequest{IncludeInactive: false})
	assert.NoError(t, err)
	assert.Len(t, mves, 1)

	// fetchTags = false path: do not call ListMVEResourceTags
	fetchTags := false
	for _, mve := range mves {
		var tags map[string]string
		if fetchTags {
			tags, _ = mockClient.MVEService.ListMVEResourceTags(context.Background(), mve.UID)
		}
		detail, diags := fromAPIMVEDetail(mve, tags)
		assert.False(t, diags.HasError())
		assert.True(t, detail.ResourceTags.IsNull())
	}
	assert.False(t, tagsCalled, "ListMVEResourceTags should not be called when include_resource_tags is false")

	// fetchTags = true path: call ListMVEResourceTags
	fetchTags = true
	for _, mve := range mves {
		var tags map[string]string
		if fetchTags {
			tags, _ = mockClient.MVEService.ListMVEResourceTags(context.Background(), mve.UID)
		}
		detail, diags := fromAPIMVEDetail(mve, tags)
		assert.False(t, diags.HasError())
		assert.False(t, detail.ResourceTags.IsNull())
	}
	assert.True(t, tagsCalled, "ListMVEResourceTags should be called when include_resource_tags is true")
}

func TestReadMVEs_GetByUIDReturnsNil(t *testing.T) {
	mockMVEService := &MockMVEService{
		GetMVEResult: nil,
		GetMVEErr:    nil,
	}
	mockClient := &megaport.Client{MVEService: mockMVEService}
	ds := &mvesDataSource{client: mockClient}

	mve, err := ds.client.MVEService.GetMVE(context.Background(), "mve-nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, mve)
}

// Ensure mvesModel compiles with the schema.
func TestMVEsModel_Structure(t *testing.T) {
	model := mvesModel{
		ProductUID: types.StringValue("mve-123"),
		MVEs:       types.ListNull(types.ObjectType{AttrTypes: mveDetailAttrs}),
	}
	assert.Equal(t, "mve-123", model.ProductUID.ValueString())
	assert.True(t, model.MVEs.IsNull())
}
