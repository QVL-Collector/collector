provider "aws" {
  region = var.region

  default_tags {
    tags = {
      projet = "collector"
      couche = "persistant"
      outil  = "terraform"
    }
  }
}
