variable "local_media_signer_member" {
  description = "IAM member that may use manga-media-signer to sign GCS image URLs, for example user:developer@example.com."
  type        = string

  validation {
    condition     = can(regex("^(user|group):.+@.+$", var.local_media_signer_member))
    error_message = "local_media_signer_member must be an IAM user or group member, for example user:developer@example.com."
  }
}
