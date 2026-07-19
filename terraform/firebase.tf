resource "google_firebase_project" "default" {
  provider = google-beta
  project  = local.project.id
}

resource "google_firebase_web_app" "manga" {
  provider     = google-beta
  project      = local.project.id
  display_name = "manga"
}

data "google_firebase_web_app_config" "manga" {
  provider   = google-beta
  web_app_id = google_firebase_web_app.manga.app_id
}

resource "google_identity_platform_config" "auth" {
  project = local.project.id

  authorized_domains = [
    "localhost",
    "${local.project.id}.firebaseapp.com",
    "${local.project.id}.web.app",
    "manga.tomozo6.com",
  ]

  multi_tenant {
    allow_tenants = false
  }

  sign_in {
    allow_duplicate_emails = false

    anonymous {
      enabled = false
    }

    email {
      enabled           = false
      password_required = false
    }

    phone_number {
      enabled = false
    }
  }
}

#resource "google_identity_platform_default_supported_idp_config" "google" {
#  provider      = google-beta
#  project       = local.project.id
#  idp_id        = "google.com"
#  enabled       = true
#  client_id     = local.google_oauth.client_id
#  client_secret = local.google_oauth.client_secret
#}
