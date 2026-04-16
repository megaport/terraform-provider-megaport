package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	megaport "github.com/megaport/megaportgo"
)

// MockIXService is a mock of the IX service for testing
type MockIXService struct {
	ListIXsResult []*megaport.IX
	ListIXsErr    error
	GetIXResult   *megaport.IX
	GetIXErr      error
}

func (m *MockIXService) ListIXs(ctx context.Context, req *megaport.ListIXsRequest) ([]*megaport.IX, error) {
	if m.ListIXsErr != nil {
		return nil, m.ListIXsErr
	}
	if m.ListIXsResult != nil {
		return m.ListIXsResult, nil
	}
	return []*megaport.IX{}, nil
}

func (m *MockIXService) GetIX(ctx context.Context, id string) (*megaport.IX, error) {
	if m.GetIXErr != nil {
		return nil, m.GetIXErr
	}
	return m.GetIXResult, nil
}

// Implement other required methods of the IXService interface with minimal stubs
func (m *MockIXService) BuyIX(ctx context.Context, req *megaport.BuyIXRequest) (*megaport.BuyIXResponse, error) {
	return nil, nil
}

func (m *MockIXService) ValidateIXOrder(ctx context.Context, req *megaport.BuyIXRequest) error {
	return nil
}

func (m *MockIXService) UpdateIX(ctx context.Context, id string, req *megaport.UpdateIXRequest) (*megaport.IX, error) {
	return nil, nil
}

func (m *MockIXService) DeleteIX(ctx context.Context, id string, req *megaport.DeleteIXRequest) error {
	return nil
}

func TestReadIXs_ListAll(t *testing.T) {
	mockIXService := &MockIXService{
		ListIXsResult: []*megaport.IX{
			{ProductUID: "ix-1", ProductName: "IX One"},
			{ProductUID: "ix-2", ProductName: "IX Two"},
		},
	}
	mockClient := &megaport.Client{IXService: mockIXService}
	ds := &ixsDataSource{client: mockClient}

	ixs, err := ds.client.IXService.ListIXs(context.Background(), &megaport.ListIXsRequest{})
	assert.NoError(t, err)
	assert.Len(t, ixs, 2)
	assert.Equal(t, "ix-1", ixs[0].ProductUID)
	assert.Equal(t, "ix-2", ixs[1].ProductUID)
}

func TestReadIXs_GetByUID(t *testing.T) {
	mockIXService := &MockIXService{
		GetIXResult: &megaport.IX{ProductUID: "ix-1", ProductName: "IX One"},
	}
	mockClient := &megaport.Client{IXService: mockIXService}
	ds := &ixsDataSource{client: mockClient}

	ix, err := ds.client.IXService.GetIX(context.Background(), "ix-1")
	assert.NoError(t, err)
	assert.Equal(t, "ix-1", ix.ProductUID)
	assert.Equal(t, "IX One", ix.ProductName)
}

func TestReadIXs_ListError(t *testing.T) {
	mockIXService := &MockIXService{
		ListIXsErr: errors.New("API error"),
	}
	mockClient := &megaport.Client{IXService: mockIXService}
	ds := &ixsDataSource{client: mockClient}

	ixs, err := ds.client.IXService.ListIXs(context.Background(), &megaport.ListIXsRequest{})
	assert.Error(t, err)
	assert.Nil(t, ixs)
	assert.Contains(t, err.Error(), "API error")
}

func TestReadIXs_GetByUIDError(t *testing.T) {
	mockIXService := &MockIXService{
		GetIXErr: errors.New("IX not found"),
	}
	mockClient := &megaport.Client{IXService: mockIXService}
	ds := &ixsDataSource{client: mockClient}

	ix, err := ds.client.IXService.GetIX(context.Background(), "ix-nonexistent")
	assert.Error(t, err)
	assert.Nil(t, ix)
	assert.Contains(t, err.Error(), "IX not found")
}

func TestFromAPIIXDetail(t *testing.T) {
	t.Run("Maps all fields correctly", func(t *testing.T) {
		ix := &megaport.IX{
			ProductUID:         "ix-abc-123",
			ProductName:        "My Test IX",
			ProvisioningStatus: "LIVE",
			LocationID:         65,
			RateLimit:          1000,
			Term:               12,
			SecondaryName:      "Secondary IX",
			VLAN:               100,
			MACAddress:         "00:11:22:33:44:55",
			ASN:                64512,
			NetworkServiceType: "Los Angeles IX",
			AttributeTags: map[string]string{
				"tag1": "val1",
				"tag2": "val2",
			},
		}

		detail, diags := fromAPIIXDetail(ix)
		assert.False(t, diags.HasError())

		assert.Equal(t, "ix-abc-123", detail.UID.ValueString())
		assert.Equal(t, "My Test IX", detail.Name.ValueString())
		assert.Equal(t, "LIVE", detail.ProvisioningStatus.ValueString())
		assert.Equal(t, int64(65), detail.LocationID.ValueInt64())
		assert.Equal(t, int64(1000), detail.RateLimit.ValueInt64())
		assert.Equal(t, int64(12), detail.Term.ValueInt64())
		assert.Equal(t, "Secondary IX", detail.SecondaryName.ValueString())
		assert.Equal(t, int64(100), detail.VLAN.ValueInt64())
		assert.Equal(t, "00:11:22:33:44:55", detail.MACAddress.ValueString())
		assert.Equal(t, int64(64512), detail.ASN.ValueInt64())
		assert.Equal(t, "Los Angeles IX", detail.NetworkServiceType.ValueString())
		assert.False(t, detail.AttributeTags.IsNull())
	})

	t.Run("Nil time fields produce null values", func(t *testing.T) {
		ix := &megaport.IX{
			ProductUID: "ix-nil-times",
			CreateDate: nil,
			DeployDate: nil,
		}

		detail, diags := fromAPIIXDetail(ix)
		assert.False(t, diags.HasError())

		assert.True(t, detail.CreateDate.IsNull())
		assert.True(t, detail.DeployDate.IsNull())
	})

	t.Run("Nil tags produce null maps", func(t *testing.T) {
		ix := &megaport.IX{
			ProductUID:    "ix-nil-tags",
			AttributeTags: nil,
		}

		detail, diags := fromAPIIXDetail(ix)
		assert.False(t, diags.HasError())

		assert.True(t, detail.AttributeTags.IsNull())
	})
}

// Ensure ixsModel compiles with the schema.
func TestIXsModel_Structure(t *testing.T) {
	model := ixsModel{
		ProductUID: types.StringValue("ix-123"),
		IXs:        types.ListNull(types.ObjectType{AttrTypes: ixDetailAttrs}),
	}
	assert.Equal(t, "ix-123", model.ProductUID.ValueString())
	assert.True(t, model.IXs.IsNull())
}
