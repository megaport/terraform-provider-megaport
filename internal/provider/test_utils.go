package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

const TestNamePrefix = "tf-acc-test-"

func RandomTestName(additionalNames ...string) string {
	var prefix strings.Builder
	prefix.WriteString(TestNamePrefix)
	for _, n := range additionalNames {
		prefix.WriteString("-" + strings.ReplaceAll(n, " ", "_"))
	}
	return randomName(prefix.String(), 10)
}

func randomName(prefix string, length int) string {
	return fmt.Sprintf("%s%s", prefix, acctest.RandString(length))
}

// RandomMACAddress returns a locally administered, unicast MAC address with
// random octets. Tests must not share a MAC: the API rejects a port whose MAC
// is already in use, which collides when tests run in parallel. The first
// octet 0x02 marks the address locally administered and unicast, so it never
// clashes with a real vendor OUI.
func RandomMACAddress() string {
	return fmt.Sprintf("02:%02X:%02X:%02X:%02X:%02X",
		acctest.RandIntRange(0, 256),
		acctest.RandIntRange(0, 256),
		acctest.RandIntRange(0, 256),
		acctest.RandIntRange(0, 256),
		acctest.RandIntRange(0, 256),
	)
}
