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

output "app_url" {
  description = "Public address of the application (load balancer)"
  value       = "http://${aws_lb.main.dns_name}"
}

output "target_group_arn" {
  description = "Target group ARN (used to check target health)"
  value       = aws_lb_target_group.app.arn
}

output "github_deploy_role_arn" {
  description = "IAM role assumed by GitHub Actions (put it in the AWS_ROLE_ARN repository variable)"
  value       = aws_iam_role.github_deploy.arn
}
