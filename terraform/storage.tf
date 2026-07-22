resource "google_storage_bucket" "manga_images" {
  name                        = "tomozo-manga-images"
  project                     = local.project.id
  location                    = local.region
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  # Preserve a previous generation when an upload accidentally overwrites an
  # image. Deleting the bucket remains non-forced (the provider default).
  versioning {
    enabled = true
  }
}

resource "google_service_account" "manga_media_signer" {
  account_id   = "manga-media-signer"
  display_name = "Manga media URL signer"
  description  = "Signs temporary GET URLs for private manga image objects."

  depends_on = [google_project_service.main["iam.googleapis.com"]]
}

resource "google_storage_bucket_iam_member" "manga_media_signer_object_viewer" {
  bucket = google_storage_bucket.manga_images.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.manga_media_signer.email}"
}

resource "google_service_account_iam_member" "local_media_signer_token_creator" {
  service_account_id = google_service_account.manga_media_signer.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = var.local_media_signer_member
}
