resource "fly_ip" "exampleIp" {
  app  = "hellofromterraform"
  type = "v4"
}

resource "fly_ip" "exampleIpv6" {
  app  = "hellofromterraform"
  type = "v6"
}