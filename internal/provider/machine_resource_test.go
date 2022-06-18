package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"testing"
)

func TestAccFlyMachine(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testFlyMachineResourceConfig(rName),
				Check:  resource.TestCheckResourceAttr("fly_machine.testMachine", "name", rName),
			},
			{
				Config: testFlyMachineResourceUpdateConfig(rName),
				Check:  resource.TestCheckResourceAttr("fly_machine.testMachine", "env.updatedkey", "updatedValue"),
			},
		},
	})
}

func testFlyMachineResourceConfig(name string) string {
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
resource "fly_machine" "testMachine" {
	app = "%s"
	region = "ewr"
	name = "%s"
    image = "nginx"
	env = {
		updatedkey = "value"
    }
    services = [
      {
        ports = [
          {
            port     = 443
            handlers = ["tls", "http"]
          },
          {
            port     = 80
            handlers = ["http"]
          }
        ]
        "protocol" : "tcp",
        "internal_port" : 80
      }
    ]
}
`, app, name)
}

func testFlyMachineResourceUpdateConfig(name string) string {
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
resource "fly_machine" "testMachine" {
	app = "%s"
	region = "ewr"
	name = "%s"
    image = "nginx"
	env = {
		updatedkey = "updatedValue"
    }
    services = [
      {
        ports = [
          {
            port     = 443
            handlers = ["tls", "http"]
          },
          {
            port     = 80
            handlers = ["http"]
          }
        ]
        "protocol" : "tcp",
        "internal_port" : 80
      }
    ]
}
`, app, name)
}
