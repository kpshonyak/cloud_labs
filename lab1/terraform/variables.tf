variable "aws_region" {
  description = "AWS region for all resources"
  type        = string
  default     = "eu-north-1"
}

variable "project_name" {
  description = "Prefix used in resource names"
  type        = string
  default     = "track-catalog"
}

variable "image_tag" {
  description = "Tag of the image in ECR that the ECS service runs (later: commit hash)"
  type        = string
  default     = "v1"
}

variable "github_repo" {
  description = "GitHub repository allowed to deploy, in the owner@id/name@id form used in the OIDC subject claim"
  type        = string
  default     = "kpshonyak@102899188/cloud_labs@1454282159"
}