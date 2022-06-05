terraform {
  required_providers {
    fly = {
      source = "dov.dev/fly/fly-provider"
    }
  }
}

resource "fly_app" "exampleApp" {
  name = "hellofromterraform"
  org  = "fly-external"
}

resource "fly_volume" "exampleVol" {
  name       = "exampleVolume"
  app        = "hellofromterraform"
  size       = 10
  region     = "ewr"
  depends_on = [fly_app.exampleApp]
}

resource "fly_volume" "secondVolume" {
  name       = "secondVolume"
  app        = "hellofromterraform"
  size       = 10
  region     = "ewr"
  depends_on = [fly_app.exampleApp]
}

resource "fly_ip" "exampleIp" {
  app        = "hellofromterraform"
  type       = "v4"
  depends_on = [fly_app.exampleApp]
}

resource "fly_ip" "exampleIpv6" {
  app        = "hellofromterraform"
  type       = "v6"
  depends_on = [fly_ip.exampleIp]
}

resource "fly_cert" "exampleCert" {
  app        = "hellofromterraform"
  hostname   = "example.com"
  depends_on = [fly_app.exampleApp]
}

resource "fly_machine" "exampleMachine" {
  app    = "hellofromterraform"
  region = "ewr"
  name   = "extremelyuniquenamelikesoveryunique18"
  image  = "nginx"
  env = {
    key = "value"
    otherKey = "theothervalue"
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
    },
    {
      ports = [
        {
          port     = 8080
          handlers = ["tls", "http"]
        },
        {
          port     = 8081
          handlers = ["http"]
        }
      ]
      "protocol" : "tcp",
      "internal_port" : 8089
    }
  ]
  mounts = [
    {
      path   = "/volume_mount"
      volume = fly_volume.exampleVol.id
    },
    {
      path   = "/the_other_volume_mount"
      volume = fly_volume.secondVolume.id
    }
  ]
  depends_on = [fly_app.exampleApp, fly_volume.exampleVol, fly_volume.secondVolume]
}

output "machineID" {
  value = fly_machine.exampleMachine.id
}

output "testipv4" {
  value = fly_ip.exampleIp.address
}

output "testipv6" {
  value = fly_ip.exampleIpv6.address
}

output "certValidationTarget" {
  value = fly_cert.exampleCert.dnsvalidationtarget
}

output "certValidationhostname" {
  value = fly_cert.exampleCert.dnsvalidationhostname
}
