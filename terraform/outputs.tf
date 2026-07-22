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

output "manga_images_bucket_name" {
  description = "Private GCS bucket that stores production manga image objects."
  value       = google_storage_bucket.manga_images.name
}

output "manga_media_signer_service_account" {
  description = "Service account used to sign temporary GCS image URLs."
  value       = google_service_account.manga_media_signer.email
}
