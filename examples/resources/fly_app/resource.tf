resource "fly_app" "exampleApp" {
  name    = "hellofromterraform"
  regions = ["ewr", "ord", "lax"]
}