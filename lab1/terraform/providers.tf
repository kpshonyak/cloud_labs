provider "aws" {
  region = var.aws_region

  # these tags are added to every resource that supports tags
  default_tags {
    tags = {
      Project   = var.project_name
      Lab       = "lab1"
      ManagedBy = "terraform"
    }
  }
}
