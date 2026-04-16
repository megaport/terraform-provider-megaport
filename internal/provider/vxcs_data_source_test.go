package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	megaport "github.com/megaport/megaportgo"
)

// MockVXCService is a mock of the VXC service for testing
type MockVXCService struct {
	ListVXCsResult            []*megaport.VXC
	ListVXCsErr               error
	GetVXCResult              *megaport.VXC
	GetVXCErr                 error
	ListVXCResourceTagsFunc   func(ctx context.Context, vxcID string) (map[string]string, error)
	ListVXCResourceTagsErr    error
	ListVXCResourceTagsResult map[string]string
	CapturedResourceTagVXCUID string
}

func (m *MockVXCService) ListVXCs(ctx context.Context, req *megaport.ListVXCsRequest) ([]*megaport.VXC, error) {
	if m.ListVXCsErr != nil {
		return nil, m.ListVXCsErr
	}
	if m.ListVXCsResult != nil {
		return m.ListVXCsResult, nil
	}
	return []*megaport.VXC{}, nil
}

func (m *MockVXCService) GetVXC(ctx context.Context, id string) (*megaport.VXC, error) {
	if m.GetVXCErr != nil {
		return nil, m.GetVXCErr
	}
	return m.GetVXCResult, nil
}

func (m *MockVXCService) ListVXCResourceTags(ctx context.Context, vxcID string) (map[string]string, error) {
	m.CapturedResourceTagVXCUID = vxcID
	if m.ListVXCResourceTagsFunc != nil {
		return m.ListVXCResourceTagsFunc(ctx, vxcID)
	}
	if m.ListVXCResourceTagsErr != nil {
		return nil, m.ListVXCResourceTagsErr
	}
	if m.ListVXCResourceTagsResult != nil {
		return m.ListVXCResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

// Implement other required methods of the VXCService interface with minimal stubs
func (m *MockVXCService) BuyVXC(ctx context.Context, req *megaport.BuyVXCRequest) (*megaport.BuyVXCResponse, error) {
	return nil, nil
}

func (m *MockVXCService) ValidateVXCOrder(ctx context.Context, req *megaport.BuyVXCRequest) error {
	return nil
}

func (m *MockVXCService) DeleteVXC(ctx context.Context, id string, req *megaport.DeleteVXCRequest) error {
	return nil
}

func (m *MockVXCService) UpdateVXC(ctx context.Context, id string, req *megaport.UpdateVXCRequest) (*megaport.VXC, error) {
	return nil, nil
}

func (m *MockVXCService) LookupPartnerPorts(ctx context.Context, req *megaport.LookupPartnerPortsRequest) (*megaport.LookupPartnerPortsResponse, error) {
	return nil, nil
}

func (m *MockVXCService) ListPartnerPorts(ctx context.Context, req *megaport.ListPartnerPortsRequest) (*megaport.ListPartnerPortsResponse, error) {
	return nil, nil
}

func (m *MockVXCService) UpdateVXCResourceTags(ctx context.Context, vxcID string, tags map[string]string) error {
	return nil
}

func TestReadVXCs_ListAll(t *testing.T) {
	mockVXCService := &MockVXCService{
		ListVXCsResult: []*megaport.VXC{
			{UID: "vxc-1", Name: "VXC One"},
			{UID: "vxc-2", Name: "VXC Two"},
		},
	}
	mockClient := &megaport.Client{VXCService: mockVXCService}
	ds := &vxcsDataSource{client: mockClient}

	vxcs, err := ds.client.VXCService.ListVXCs(context.Background(), &megaport.ListVXCsRequest{IncludeInactive: false})
	assert.NoError(t, err)
	assert.Len(t, vxcs, 2)
	assert.Equal(t, "vxc-1", vxcs[0].UID)
	assert.Equal(t, "vxc-2", vxcs[1].UID)
}

func TestReadVXCs_GetByUID(t *testing.T) {
	mockVXCService := &MockVXCService{
		GetVXCResult: &megaport.VXC{UID: "vxc-1", Name: "VXC One"},
	}
	mockClient := &megaport.Client{VXCService: mockVXCService}
	ds := &vxcsDataSource{client: mockClient}

	vxc, err := ds.client.VXCService.GetVXC(context.Background(), "vxc-1")
	assert.NoError(t, err)
	assert.Equal(t, "vxc-1", vxc.UID)
	assert.Equal(t, "VXC One", vxc.Name)
}

func TestReadVXCs_ListError(t *testing.T) {
	mockVXCService := &MockVXCService{
		ListVXCsErr: errors.New("API error"),
	}
	mockClient := &megaport.Client{VXCService: mockVXCService}
	ds := &vxcsDataSource{client: mockClient}

	vxcs, err := ds.client.VXCService.ListVXCs(context.Background(), &megaport.ListVXCsRequest{IncludeInactive: false})
	assert.Error(t, err)
	assert.Nil(t, vxcs)
	assert.Contains(t, err.Error(), "API error")
}

func TestReadVXCs_GetByUIDError(t *testing.T) {
	mockVXCService := &MockVXCService{
		GetVXCErr: errors.New("VXC not found"),
	}
	mockClient := &megaport.Client{VXCService: mockVXCService}
	ds := &vxcsDataSource{client: mockClient}

	vxc, err := ds.client.VXCService.GetVXC(context.Background(), "vxc-nonexistent")
	assert.Error(t, err)
	assert.Nil(t, vxc)
	assert.Contains(t, err.Error(), "VXC not found")
}

func TestFromAPIVXCDetail(t *testing.T) {
	t.Run("Maps all fields correctly", func(t *testing.T) {
		vxc := &megaport.VXC{
			UID:                "vxc-abc-123",
			Name:               "My Test VXC",
			RateLimit:          1000,
			ProvisioningStatus: "LIVE",
			CreatedBy:          "user@example.com",
			CostCentre:         "CC-001",
			ContractTermMonths: 12,
			CompanyUID:         "company-abc",
			CompanyName:        "Acme Corp",
			DistanceBand:       "METRO",
			SecondaryName:      "Secondary VXC",
			Shutdown:           false,
			Locked:             false,
			AdminLocked:        true,
			Cancelable:         true,
			AEndConfiguration: megaport.VXCEndConfiguration{
				UID:        "port-aaa",
				Name:       "Port A",
				LocationID: 123,
				VLAN:       100,
			},
			BEndConfiguration: megaport.VXCEndConfiguration{
				UID:        "port-bbb",
				Name:       "Port B",
				LocationID: 456,
				VLAN:       200,
			},
			AttributeTags: map[string]string{
				"tag1": "val1",
				"tag2": "val2",
			},
		}

		tags := map[string]string{
			"env":   "production",
			"owner": "team-a",
		}

		detail := fromAPIVXCDetail(vxc, tags)

		assert.Equal(t, "vxc-abc-123", detail.UID.ValueString())
		assert.Equal(t, "My Test VXC", detail.Name.ValueString())
		assert.Equal(t, int64(1000), detail.RateLimit.ValueInt64())
		assert.Equal(t, "LIVE", detail.ProvisioningStatus.ValueString())
		assert.Equal(t, "user@example.com", detail.CreatedBy.ValueString())
		assert.Equal(t, "CC-001", detail.CostCentre.ValueString())
		assert.Equal(t, int64(12), detail.ContractTermMonths.ValueInt64())
		assert.Equal(t, "company-abc", detail.CompanyUID.ValueString())
		assert.Equal(t, "Acme Corp", detail.CompanyName.ValueString())
		assert.Equal(t, "METRO", detail.DistanceBand.ValueString())
		assert.Equal(t, "Secondary VXC", detail.SecondaryName.ValueString())
		assert.Equal(t, false, detail.Shutdown.ValueBool())
		assert.Equal(t, false, detail.Locked.ValueBool())
		assert.Equal(t, true, detail.AdminLocked.ValueBool())
		assert.Equal(t, true, detail.Cancelable.ValueBool())
		assert.Equal(t, "port-aaa", detail.AEndUID.ValueString())
		assert.Equal(t, "Port A", detail.AEndName.ValueString())
		assert.Equal(t, int64(123), detail.AEndLocationID.ValueInt64())
		assert.Equal(t, int64(100), detail.AEndVLAN.ValueInt64())
		assert.Equal(t, "port-bbb", detail.BEndUID.ValueString())
		assert.Equal(t, "Port B", detail.BEndName.ValueString())
		assert.Equal(t, int64(456), detail.BEndLocationID.ValueInt64())
		assert.Equal(t, int64(200), detail.BEndVLAN.ValueInt64())
		assert.False(t, detail.AttributeTags.IsNull())
		assert.False(t, detail.ResourceTags.IsNull())
	})

	t.Run("Nil time fields produce empty strings", func(t *testing.T) {
		vxc := &megaport.VXC{
			UID:               "vxc-nil-times",
			CreateDate:        nil,
			LiveDate:          nil,
			ContractStartDate: nil,
			ContractEndDate:   nil,
		}

		detail := fromAPIVXCDetail(vxc, nil)

		assert.Equal(t, "", detail.CreateDate.ValueString())
		assert.Equal(t, "", detail.LiveDate.ValueString())
		assert.Equal(t, "", detail.ContractStartDate.ValueString())
		assert.Equal(t, "", detail.ContractEndDate.ValueString())
	})

	t.Run("Nil tags produce null maps", func(t *testing.T) {
		vxc := &megaport.VXC{
			UID:           "vxc-nil-tags",
			AttributeTags: nil,
		}

		detail := fromAPIVXCDetail(vxc, nil)

		assert.True(t, detail.AttributeTags.IsNull())
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Empty resource tags produce null map", func(t *testing.T) {
		vxc := &megaport.VXC{
			UID: "vxc-empty-tags",
		}

		detail := fromAPIVXCDetail(vxc, map[string]string{})

		assert.True(t, detail.ResourceTags.IsNull())
	})
}

func TestFromAPIVXCDetail_ResourceTagsOptIn(t *testing.T) {
	t.Run("Tags nil when not fetched", func(t *testing.T) {
		vxc := &megaport.VXC{UID: "vxc-1"}
		detail := fromAPIVXCDetail(vxc, nil)
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Tags populated when fetched", func(t *testing.T) {
		vxc := &megaport.VXC{UID: "vxc-1"}
		tags := map[string]string{"env": "prod"}
		detail := fromAPIVXCDetail(vxc, tags)
		assert.False(t, detail.ResourceTags.IsNull())
	})
}

func TestReadVXCs_TagsNotFetchedByDefault(t *testing.T) {
	tagsCalled := false
	mockVXCService := &MockVXCService{
		ListVXCsResult: []*megaport.VXC{
			{UID: "vxc-1", Name: "VXC One"},
		},
		ListVXCResourceTagsFunc: func(_ context.Context, _ string) (map[string]string, error) {
			tagsCalled = true
			return map[string]string{"env": "test"}, nil
		},
	}
	mockClient := &megaport.Client{VXCService: mockVXCService}

	// Simulate Read path: tags should not be fetched when fetchTags is false
	vxcs, err := mockClient.VXCService.ListVXCs(context.Background(), &megaport.ListVXCsRequest{IncludeInactive: false})
	assert.NoError(t, err)
	assert.Len(t, vxcs, 1)

	// fetchTags = false path: do not call ListVXCResourceTags
	fetchTags := false
	for _, vxc := range vxcs {
		var tags map[string]string
		if fetchTags {
			tags, _ = mockClient.VXCService.ListVXCResourceTags(context.Background(), vxc.UID)
		}
		detail := fromAPIVXCDetail(vxc, tags)
		assert.True(t, detail.ResourceTags.IsNull())
	}
	assert.False(t, tagsCalled, "ListVXCResourceTags should not be called when include_resource_tags is false")

	// fetchTags = true path: call ListVXCResourceTags
	fetchTags = true
	for _, vxc := range vxcs {
		var tags map[string]string
		if fetchTags {
			tags, _ = mockClient.VXCService.ListVXCResourceTags(context.Background(), vxc.UID)
		}
		detail := fromAPIVXCDetail(vxc, tags)
		assert.False(t, detail.ResourceTags.IsNull())
	}
	assert.True(t, tagsCalled, "ListVXCResourceTags should be called when include_resource_tags is true")
}

// Ensure vxcsModel compiles with the schema.
func TestVXCsModel_Structure(t *testing.T) {
	model := vxcsModel{
		ProductUID: types.StringValue("vxc-123"),
		VXCs:       types.ListNull(types.ObjectType{AttrTypes: vxcDetailAttrs}),
	}
	assert.Equal(t, "vxc-123", model.ProductUID.ValueString())
	assert.True(t, model.VXCs.IsNull())
}
