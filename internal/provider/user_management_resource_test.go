package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

	firstNameNew := "Updated"
	lastNameNew := "User" + RandomTestName()
	emailNew := fmt.Sprintf("updated.user.%s@example.com", RandomTestName())
	positionNew := "Read Only"

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
				Config: providerConfig + fmt.Sprintf(`
					resource "megaport_user" "test_user" {
						first_name = "%s"
						last_name  = "%s"
						email      = "%s"
						position   = "%s"
						active     = false
						notification_enabled = false
						newsletter = false
						promotions = false
					}`, firstNameNew, lastNameNew, emailNew, positionNew),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_user.test_user", "first_name", firstNameNew),
					resource.TestCheckResourceAttr("megaport_user.test_user", "last_name", lastNameNew),
					resource.TestCheckResourceAttr("megaport_user.test_user", "email", emailNew),
					resource.TestCheckResourceAttr("megaport_user.test_user", "position", positionNew),
					resource.TestCheckResourceAttr("megaport_user.test_user", "active", "false"),
					resource.TestCheckResourceAttr("megaport_user.test_user", "notification_enabled", "false"),
					resource.TestCheckResourceAttr("megaport_user.test_user", "newsletter", "false"),
					resource.TestCheckResourceAttr("megaport_user.test_user", "promotions", "false"),
				),
			},
		},
	})
}

// func (suite *UserManagementProviderTestSuite) TestAccMegaportUser_AdminUser() {
// 	firstName := "Admin"
// 	lastName := "User" + RandomTestName()
// 	email := fmt.Sprintf("admin.user.%s@example.com", RandomTestName())
// 	position := "Company Admin"

// 	resource.Test(suite.T(), resource.TestCase{
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: providerConfig + fmt.Sprintf(`
// 					resource "megaport_user" "admin_user" {
// 						first_name = "%s"
// 						last_name  = "%s"
// 						email      = "%s"
// 						position   = "%s"
// 						active     = true
// 						require_totp = true
// 						notification_enabled = true
// 						newsletter = true
// 						promotions = false
// 					}`, firstName, lastName, email, position),
// 				Check: resource.ComposeTestCheckFunc(
// 					resource.TestCheckResourceAttr("megaport_user.admin_user", "first_name", firstName),
// 					resource.TestCheckResourceAttr("megaport_user.admin_user", "last_name", lastName),
// 					resource.TestCheckResourceAttr("megaport_user.admin_user", "email", email),
// 					resource.TestCheckResourceAttr("megaport_user.admin_user", "position", position),
// 					resource.TestCheckResourceAttr("megaport_user.admin_user", "active", "true"),
// 					resource.TestCheckResourceAttr("megaport_user.admin_user", "require_totp", "true"),
// 					resource.TestCheckResourceAttr("megaport_user.admin_user", "notification_enabled", "true"),
// 					resource.TestCheckResourceAttr("megaport_user.admin_user", "newsletter", "true"),
// 					resource.TestCheckResourceAttr("megaport_user.admin_user", "promotions", "false"),
// 					testAccCheckMegaportUserExists("megaport_user.admin_user"),
// 				),
// 			},
// 		},
// 	})
// }
