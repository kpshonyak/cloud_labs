resource "random_password" "jwt_secret" {
  length  = 48
  special = false
}

resource "random_password" "admin" {
  length  = 20
  special = false
}

resource "aws_secretsmanager_secret" "auth" {
  name                    = "${var.project_name}/auth"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "auth" {
  secret_id = aws_secretsmanager_secret.auth.id
  secret_string = jsonencode({
    jwt_secret     = random_password.jwt_secret.result
    admin_password = random_password.admin.result
  })
}
