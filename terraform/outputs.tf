output "firebase_web_config" {
  description = "Firebase Web SDK config for local index.html."
  sensitive   = true

  value = {
    apiKey            = data.google_firebase_web_app_config.manga.api_key
    authDomain        = data.google_firebase_web_app_config.manga.auth_domain
    projectId         = local.project.id
    appId             = google_firebase_web_app.manga.app_id
    messagingSenderId = data.google_firebase_web_app_config.manga.messaging_sender_id
    storageBucket     = data.google_firebase_web_app_config.manga.storage_bucket
  }
}
