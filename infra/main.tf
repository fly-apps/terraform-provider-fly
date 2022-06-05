terraform {
  required_providers {
    fly = {
      source = "fly-apps/fly"
    }
  }
}

resource "fly_app" "acctestapp" {
    name = "acctestapp"
    org = "fly-terraform-ci"
  }
