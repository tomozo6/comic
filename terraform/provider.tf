terraform {
  required_version = ">= 1.15.7"

  backend "gcs" {
    bucket = "tomozo6-tfstate-gcs"
    prefix = "manga"
  }

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.40.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 7.40.0"
    }
  }
}

provider "google" {
  project               = local.project.id
  region                = local.region
  user_project_override = true
  billing_project       = local.project.id
}

provider "google-beta" {
  project               = local.project.id
  region                = local.region
  user_project_override = true
  billing_project       = local.project.id
}
