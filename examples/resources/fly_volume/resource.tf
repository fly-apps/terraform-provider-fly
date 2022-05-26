resource "fly_volume" "exampleApp" {
  name   = "exampleVolume"
  app    = "hellofromterraform"
  size   = 10
  region = "ewr"
}
