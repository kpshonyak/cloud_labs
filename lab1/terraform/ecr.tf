# Private container registry for the application image
resource "aws_ecr_repository" "app" {
  name = var.project_name

  # tags cannot be overwritten: a tag = one commit hash = one image (needed for CD and rollback)
  image_tag_mutability = "IMMUTABLE"

  # terraform destroy removes the repository even if it still contains images
  force_delete = true

  image_scanning_configuration {
    scan_on_push = true
  }
}
