package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

const TestNamePrefix = "tf-acc-test-"

func RandomTestName(additionalNames ...string) string {
	prefix := TestNamePrefix
	for _, n := range additionalNames {
		prefix += "-" + strings.ReplaceAll(n, " ", "_")
	}
	return randomName(prefix, 10)
}

func randomName(prefix string, length int) string {
	return fmt.Sprintf("%s%s", prefix, acctest.RandString(length))
}
