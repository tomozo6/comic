resource "google_project_service" "main" {
  for_each = toset([
    "firebase.googleapis.com",
    "identitytoolkit.googleapis.com",
  ])

  project            = local.project.id
  service            = each.value
  disable_on_destroy = false
}
