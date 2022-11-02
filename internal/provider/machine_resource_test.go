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
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
%s

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "%s"
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
`, providerConfig(), app, getTestRegion(), name)
}

func testFlyMachineResourceUpdateConfig(name string) string {
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
%s

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "%s"
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
`, providerConfig(), app, getTestRegion(), name)
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
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
%s

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "%s"
	name = "%s"
    image = "nginx"
	env = {
		updatedkey = "value"
    }
}
`, providerConfig(), app, getTestRegion(), name)
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
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
%s

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "%s"
	name = "%s"
    image = "nginx"
	env = {
		updatedkey = "value"
    }
    services = []
}
`, providerConfig(), app, getTestRegion(), name)
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
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
%s

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "%s"
	name = "%s"
    image = "nginx"
	env = {
		updatedkey = "value"
    }
    cmd = ["cmdText"]
    entrypoint = ["entrypointText"]
	exec = ["execText"]
}
`, providerConfig(), app, getTestRegion(), name)
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
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
%s

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "%s"
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
`, providerConfig(), app, getTestRegion(), name)
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
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
%s

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "%s"
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
`, providerConfig(), getTestRegion(), app)
}

func testFlyMachineResourceEmptyNameUpdateConfig() string {
	app := os.Getenv("FLY_TF_TEST_APP")

	return fmt.Sprintf(`
%s

resource "fly_machine" "testMachine" {
	app = "%s"
	region = "%s"
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
`, providerConfig(), getTestRegion(), app)
}
