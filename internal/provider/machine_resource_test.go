package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"testing"
)

func TestAccFlyMachineBase(t *testing.T) {
	t.Parallel()
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
	org := os.Getenv("FLY_TF_TEST_ORG")
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
provider "fly" {
  useinternaltunnel    = true
  internaltunnelorg    = "%s"
  internaltunnelregion = "ewr"
}

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
`, org, app, name)
}

func testFlyMachineResourceUpdateConfig(name string) string {
	org := os.Getenv("FLY_TF_TEST_ORG")
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
provider "fly" {
  useinternaltunnel    = true
  internaltunnelorg    = "%s"
  internaltunnelregion = "ewr"
}

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
`, org, app, name)
}

func TestAccFlyMachineNoServices(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testFlyMachineResourceNoServicesConfig(rName),
				Check:  resource.TestCheckResourceAttr("fly_machine.testMachine", "name", rName),
			},
		},
	})
}

func testFlyMachineResourceNoServicesConfig(name string) string {
	org := os.Getenv("FLY_TF_TEST_ORG")
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
provider "fly" {
  useinternaltunnel    = true
  internaltunnelorg    = "%s"
  internaltunnelregion = "ewr"
}

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "ewr"
	name = "%s"
    image = "nginx"
	env = {
		updatedkey = "value"
    }
}
`, org, app, name)
}

func TestAccFlyMachineEmptyServices(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testFlyMachineResourceEmptyServicesConfig(rName),
				Check:  resource.TestCheckResourceAttr("fly_machine.testMachine", "name", rName),
			},
		},
	})
}

func testFlyMachineResourceEmptyServicesConfig(name string) string {
	org := os.Getenv("FLY_TF_TEST_ORG")
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
provider "fly" {
  useinternaltunnel    = true
  internaltunnelorg    = "%s"
  internaltunnelregion = "ewr"
}

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "ewr"
	name = "%s"
    image = "nginx"
	env = {
		updatedkey = "value"
    }
    services = []
}
`, org, app, name)
}

func TestAccFlyMachineInitOptions(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testFlyMachineResourceInitOptionsConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fly_machine.testMachine", "name", rName),
					resource.TestCheckResourceAttr("fly_machine.testMachine", "cmd.0", "cmdText"),
					resource.TestCheckResourceAttr("fly_machine.testMachine", "entrypoint.0", "entrypointText"),
					resource.TestCheckResourceAttr("fly_machine.testMachine", "exec.0", "execText"),
				),
			},
		},
	})
}

func testFlyMachineResourceInitOptionsConfig(name string) string {
	org := os.Getenv("FLY_TF_TEST_ORG")
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
provider "fly" {
  useinternaltunnel    = true
  internaltunnelorg    = "%s"
  internaltunnelregion = "ewr"
}

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "ewr"
	name = "%s"
    image = "nginx"
	env = {
		updatedkey = "value"
    }
    cmd = ["cmdText"]
    entrypoint = ["entrypointText"]
	exec = ["execText"]
}
`, org, app, name)
}

func TestAccFlyMachineModifyImage(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testFlyMachineResourceConfig(rName),
			},
			{
				Config: testFlyMachineResourceChangeImageConfig(rName),
			},
		},
	})
}

func testFlyMachineResourceChangeImageConfig(name string) string {
	org := os.Getenv("FLY_TF_TEST_ORG")
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
provider "fly" {
  useinternaltunnel    = true
  internaltunnelorg    = "%s"
  internaltunnelregion = "ewr"
}

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "ewr"
	name = "%s"
    image = "nginx:latest"
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
`, org, app, name)
}

func TestAccFlyMachineEmptyName(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testFlyMachineResourceEmptyNameConfig(),
			},
			{
				Config: testFlyMachineResourceEmptyNameUpdateConfig(),
			},
		},
	})
}

func testFlyMachineResourceEmptyNameConfig() string {
	org := os.Getenv("FLY_TF_TEST_ORG")
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
provider "fly" {
  useinternaltunnel    = true
  internaltunnelorg    = "%s"
  internaltunnelregion = "ewr"
}

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "ewr"
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
`, org, app)
}

func testFlyMachineResourceEmptyNameUpdateConfig() string {
	org := os.Getenv("FLY_TF_TEST_ORG")
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
provider "fly" {
  useinternaltunnel    = true
  internaltunnelorg    = "%s"
  internaltunnelregion = "ewr"
}

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "ewr"
	image = "nginx:latest"
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
`, org, app)
}
