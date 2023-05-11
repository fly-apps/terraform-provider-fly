terraform {
  required_providers {
    fly = {
      source = "fly-apps/fly"
    }
  }
}

resource "fly_app" "acctestapp" {
  name = var.fly_ci_app
  org  = var.fly_ci_org
}

variable "fly_ci_org" {
  type    = string
  default = "fly-terraform-ci"
}

variable "fly_ci_app" {
  type    = string
  default = "acctestapp"
}
