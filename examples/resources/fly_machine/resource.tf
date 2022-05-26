resource "fly_machine" "exampleMachine" {
  app    = "hellofromterraform"
  region = "iad"
  name   = "extremelyuniquenamelikesoveryunique8"
  image  = "nginx"
  env = {
    key = "value"
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
}
