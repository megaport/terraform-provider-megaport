package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type UserManagementProviderTestSuite ProviderTestSuite

func TestUserManagementProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UserManagementProviderTestSuite))
}

func (suite *UserManagementProviderTestSuite) TestAccMegaportUser_Basic() {
	firstName := "Test"
	lastName := "User" + RandomTestName()
	email := fmt.Sprintf("test.user.%s@example.com", RandomTestName())
	position := "Technical Contact"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
					resource "megaport_user" "test_user" {
						first_name = "%s"
						last_name  = "%s"
						email      = "%s"
						position   = "%s"
						active     = true
					}`, firstName, lastName, email, position),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_user.test_user", "first_name", firstName),
					resource.TestCheckResourceAttr("megaport_user.test_user", "last_name", lastName),
					resource.TestCheckResourceAttr("megaport_user.test_user", "email", email),
					resource.TestCheckResourceAttr("megaport_user.test_user", "position", position),
					resource.TestCheckResourceAttr("megaport_user.test_user", "active", "true"),
					resource.TestCheckResourceAttrSet("megaport_user.test_user", "employee_id"),
					resource.TestCheckResourceAttrSet("megaport_user.test_user", "party_id"),
					resource.TestCheckResourceAttrSet("megaport_user.test_user", "uid"),
					resource.TestCheckResourceAttrSet("megaport_user.test_user", "username"),
				),
			},

			{
				ResourceName:                         "megaport_user.test_user",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "employee_id",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_user.test_user"
					var rawState map[string]string
					for _, m := range state.Modules {
						if len(m.Resources) > 0 {
							if v, ok := m.Resources[resourceName]; ok {
								rawState = v.Primary.Attributes
							}
						}
					}
					return rawState["employee_id"], nil
				},
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}

func (suite *UserManagementProviderTestSuite) TestAccMegaportUser_WithDataSourceIntegration() {
	firstName := "DataSource"
	lastName := "TestUser" + RandomTestName()
	email := fmt.Sprintf("datasource.test.%s@example.com", RandomTestName())
	position := "Technical Contact"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create user and verify it appears in data source
			{
				Config: providerConfig + fmt.Sprintf(`
					resource "megaport_user" "test_user" {
						first_name = "%s"
						last_name  = "%s"
						email      = "%s"
						position   = "%s"
						active     = true
					}

					# Data source to list all users
					data "megaport_users" "all_users" {
						depends_on = [megaport_user.test_user]
					}

					# Data source filtered by first name to find our specific user
					data "megaport_users" "filtered_by_first_name" {
						first_name = "%s"
						depends_on = [megaport_user.test_user]
					}

					# Data source filtered by email to find our specific user
					data "megaport_users" "filtered_by_email" {
						email = "%s"
						depends_on = [megaport_user.test_user]
					}
				`, firstName, lastName, email, position, firstName, email),
				Check: resource.ComposeTestCheckFunc(
					// Check the user resource was created
					resource.TestCheckResourceAttr("megaport_user.test_user", "first_name", firstName),
					resource.TestCheckResourceAttr("megaport_user.test_user", "email", email),

					// Check the user appears in the unfiltered data source
					resource.TestCheckResourceAttrSet("data.megaport_users.all_users", "users.#"),

					// Check the user appears in the first name filtered data source
					resource.TestCheckResourceAttr("data.megaport_users.filtered_by_first_name", "users.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_users.filtered_by_first_name", "users.0.first_name", firstName),
					resource.TestCheckResourceAttr("data.megaport_users.filtered_by_first_name", "users.0.last_name", lastName),
					resource.TestCheckResourceAttr("data.megaport_users.filtered_by_first_name", "users.0.email", email),
					resource.TestCheckResourceAttr("data.megaport_users.filtered_by_first_name", "users.0.position", position),

					// Check the user appears in the email filtered data source
					resource.TestCheckResourceAttr("data.megaport_users.filtered_by_email", "users.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_users.filtered_by_email", "users.0.first_name", firstName),
					resource.TestCheckResourceAttr("data.megaport_users.filtered_by_email", "users.0.email", email),
				),
			},
		},
	})
}

func (suite *UserManagementProviderTestSuite) TestAccMegaportUser_DataSourceFiltering() {
	firstName1 := "Filter"
	lastName1 := "TestUser1" + RandomTestName()
	email1 := fmt.Sprintf("filter.test1.%s@example.com", RandomTestName())

	firstName2 := "Filter"
	lastName2 := "TestUser2" + RandomTestName()
	email2 := fmt.Sprintf("filter.test2.%s@example.com", RandomTestName())

	position := "Technical Contact"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
					resource "megaport_user" "test_user1" {
						first_name = "%s"
						last_name  = "%s"
						email      = "%s"
						position   = "%s"
						active     = true
					}

					resource "megaport_user" "test_user2" {
						first_name = "%s"
						last_name  = "%s"
						email      = "%s"
						position   = "%s"
						active     = true
					}

					# Filter by first name (should return both users)
					data "megaport_users" "by_first_name" {
						first_name = "%s"
						depends_on = [megaport_user.test_user1, megaport_user.test_user2]
					}

					# Filter by last name (should return only one user)
					data "megaport_users" "by_last_name" {
						last_name = "%s"
						depends_on = [megaport_user.test_user1, megaport_user.test_user2]
					}

					# Filter by specific email (should return only one user)
					data "megaport_users" "by_email" {
						email = "%s"
						depends_on = [megaport_user.test_user1, megaport_user.test_user2]
					}

					# Also create an unfiltered data source to see all users for debugging
					data "megaport_users" "all_users_debug" {
						depends_on = [megaport_user.test_user1, megaport_user.test_user2]
					}
				`, firstName1, lastName1, email1, position,
					firstName2, lastName2, email2, position,
					firstName1, lastName1, email1),
				Check: resource.ComposeTestCheckFunc(
					// Check filtering by first name returns both users
					resource.TestCheckResourceAttr("data.megaport_users.by_first_name", "users.#", "2"),

					// Check filtering by last name returns only one user
					resource.TestCheckResourceAttr("data.megaport_users.by_last_name", "users.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_users.by_last_name", "users.0.last_name", lastName1),

					// Check filtering by email returns only one user
					resource.TestCheckResourceAttr("data.megaport_users.by_email", "users.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_users.by_email", "users.0.email", email1),
				),
			},
		},
	})
}
