resource "fly_cert" "exampleCert" {
  app      = "hellofromterraform"
  hostname = "example.com"
}
