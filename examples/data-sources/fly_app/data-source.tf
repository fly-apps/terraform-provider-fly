data "fly_app" "example" {
  name = "hellofromterraform"
  depends_on = [
    fly_app.exampleApp
  ]
}
