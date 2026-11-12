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
  description = "GitHub repository allowed to deploy (owner/name)"
  type        = string
  default     = "kpshonyak/cloud_labs"
}
