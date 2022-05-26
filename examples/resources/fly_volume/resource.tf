resource "fly_volume" "exampleApp" {
  name       = "exampleVolume"
  app        = "hellofromterraform"
  size       = 10
  region     = "ewr"
  depends_on = [fly_app.exampleApp]
}
