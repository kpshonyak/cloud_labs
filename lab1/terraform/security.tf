# --- application ---
resource "aws_security_group" "app" {
  name        = "${var.project_name}-app"
  description = "Application containers"
  vpc_id      = aws_vpc.main.id
}

# Temporary: open to the internet on the app port (task 2). In task 4 this becomes "only from the load balancer".
resource "aws_vpc_security_group_ingress_rule" "app_http" {
  security_group_id = aws_security_group.app.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "tcp"
  from_port         = 8080
  to_port           = 8080
}

# Outbound: pull the image from ECR, talk to the database, send logs
resource "aws_vpc_security_group_egress_rule" "app_all" {
  security_group_id = aws_security_group.app.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
}

# --- database ---
resource "aws_security_group" "db" {
  name        = "${var.project_name}-db"
  description = "PostgreSQL database"
  vpc_id      = aws_vpc.main.id
}

# The database accepts connections ONLY from the application's security group (not from IP ranges)
resource "aws_vpc_security_group_ingress_rule" "db_from_app" {
  security_group_id            = aws_security_group.db.id
  referenced_security_group_id = aws_security_group.app.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
}
