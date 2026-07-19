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
