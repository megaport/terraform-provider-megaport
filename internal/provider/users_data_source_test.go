package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

// TestUsersDataSource_FilterUsers tests the filtering logic of the users data source
func TestUsersDataSource_FilterUsers(t *testing.T) {
	// Create test data source
	ds := &usersDataSource{}

	// Create mock users for testing
	users := []*megaport.User{
		{
			EmploymentId: 1001,
			FirstName:    "John",
			LastName:     "Doe",
			Email:        "john.doe@example.com",
			Phone:        "+1234567890",
			Position:     "Technical Contact",
			PersonUid:    "uid-001",
			Name:         "John Doe",
		},
		{
			EmploymentId: 1002,
			FirstName:    "Jane",
			LastName:     "Smith",
			Email:        "jane.smith@example.com",
			Phone:        "+0987654321",
			Position:     "Company Admin",
			PersonUid:    "uid-002",
			Name:         "Jane Smith",
		},
		{
			EmploymentId: 1003,
			FirstName:    "Bob",
			LastName:     "Johnson",
			Email:        "bob.johnson@example.com",
			Phone:        "+1122334455",
			Position:     "Technical Contact",
			PersonUid:    "uid-003",
			Name:         "Bob Johnson",
		},
	}

	tests := []struct {
		name           string
		filters        usersModel
		expectedCount  int
		expectedUserID int64
	}{
		{
			name:          "No filters - returns all users",
			filters:       usersModel{},
			expectedCount: 3,
		},
		{
			name: "Filter by employee_id",
			filters: usersModel{
				EmployeeID: types.Int64Value(1001),
			},
			expectedCount:  1,
			expectedUserID: 1001,
		},
		{
			name: "Filter by first_name",
			filters: usersModel{
				FirstName: types.StringValue("Jane"),
			},
			expectedCount:  1,
			expectedUserID: 1002,
		},
		{
			name: "Filter by last_name",
			filters: usersModel{
				LastName: types.StringValue("Johnson"),
			},
			expectedCount:  1,
			expectedUserID: 1003,
		},
		{
			name: "Filter by email (case insensitive)",
			filters: usersModel{
				Email: types.StringValue("JOHN.DOE@EXAMPLE.COM"),
			},
			expectedCount:  1,
			expectedUserID: 1001,
		},
		{
			name: "Filter by phone",
			filters: usersModel{
				Phone: types.StringValue("+0987654321"),
			},
			expectedCount:  1,
			expectedUserID: 1002,
		},
		{
			name: "Filter by position",
			filters: usersModel{
				Position: types.StringValue("Company Admin"),
			},
			expectedCount:  1,
			expectedUserID: 1002,
		},
		{
			name: "Filter by uid",
			filters: usersModel{
				UID: types.StringValue("uid-003"),
			},
			expectedCount:  1,
			expectedUserID: 1003,
		},
		{
			name: "Filter by name",
			filters: usersModel{
				Name: types.StringValue("Jane Smith"),
			},
			expectedCount:  1,
			expectedUserID: 1002,
		},
		{
			name: "Multiple filters - matching user",
			filters: usersModel{
				FirstName: types.StringValue("John"),
				Position:  types.StringValue("Technical Contact"),
			},
			expectedCount:  1,
			expectedUserID: 1001,
		},
		{
			name: "Multiple filters - no matching user",
			filters: usersModel{
				FirstName: types.StringValue("John"),
				Position:  types.StringValue("Company Admin"),
			},
			expectedCount: 0,
		},
		{
			name: "Non-existent employee_id",
			filters: usersModel{
				EmployeeID: types.Int64Value(9999),
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ds.filterUsers(users, tt.filters)
			assert.Equal(t, tt.expectedCount, len(result), "Expected %d users, got %d", tt.expectedCount, len(result))

			if tt.expectedCount == 1 && tt.expectedUserID > 0 {
				assert.Equal(t, tt.expectedUserID, int64(result[0].EmploymentId), "Expected user with employment ID %d", tt.expectedUserID)
			}
		})
	}
}
