#!/usr/bin/env bash
# Prints the public IP of the running ECS task
set -euo pipefail

CLUSTER="${1:-track-catalog}"
REGION="${AWS_REGION:-eu-north-1}"

TASK=$(aws ecs list-tasks --cluster "$CLUSTER" --region "$REGION" --query 'taskArns[0]' --output text)
if [ "$TASK" = "None" ]; then
  echo "No running tasks in cluster $CLUSTER" >&2
  exit 1
fi

ENI=$(aws ecs describe-tasks --cluster "$CLUSTER" --tasks "$TASK" --region "$REGION" \
  --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value" --output text)

aws ec2 describe-network-interfaces --network-interface-ids "$ENI" --region "$REGION" \
  --query 'NetworkInterfaces[0].Association.PublicIp' --output text
