output "ecr_repository_url" {
  description = "URL of the ECR repository (used for docker push)"
  value       = aws_ecr_repository.app.repository_url
}

output "ecs_cluster_name" {
  description = "ECS cluster name"
  value       = aws_ecs_cluster.main.name
}

output "rds_endpoint" {
  description = "Database address (reachable only from inside the VPC)"
  value       = aws_db_instance.main.address
}
