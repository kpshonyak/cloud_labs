# Лабораторна 1 — Track Catalog

REST API для каталогу треків і плейлистів (Go + PostgreSQL), розгорнутий в AWS за допомогою Terraform
з автодеплоєм через GitHub Actions.

**Публічна адреса:** _http://&lt;dns-ім'я-балансувальника&gt;_ (вказується після `terraform apply`:
`terraform output -raw app_url`; після кожного `destroy`/`apply` адреса змінюється)

Документація API (Swagger UI): `<публічна адреса>/docs`

## Структура репозиторію

```
.github/workflows/deploy.yml   # CI/CD: збірка → ECR → оновлення ECS
lab1/
├── app/                       # код застосунку (Go), Dockerfile, OpenAPI-специфікація
├── terraform/                 # уся інфраструктура як код
├── scripts/                   # допоміжні скрипти
├── compose.yaml               # локальний запуск: застосунок + PostgreSQL
└── .env.example               # шаблон змінних середовища
```

## Ендпоінти

| Метод | Шлях | Опис |
|---|---|---|
| GET | `/health` | перевірка зв'язку з базою |
| GET, POST | `/playlists` | список і створення плейлистів |
| GET | `/playlists/{id}/tracks` | треки плейлиста |
| GET, POST | `/tracks` | список і створення треків |
| GET, PUT, DELETE | `/tracks/{id}` | читання, оновлення, видалення треку |
| GET | `/docs` | Swagger UI |
| GET | `/openapi.yaml` | специфікація OpenAPI |

## Локальний запуск

```bash
cd lab1
cp .env.example .env        
docker compose up --build
curl localhost:8080/health
```


## Розгортання в AWS

Потрібні: AWS CLI з налаштованим IAM-користувачем, Terraform, Docker.

```bash
cd lab1/terraform
terraform init

# 1) спочатку реєстр, щоб було куди покласти образ
terraform apply -target=aws_ecr_repository.app

# 2) перший образ (тег v1 використовується в task definition за замовчуванням)
cd ..
ECR=$(terraform -chdir=terraform output -raw ecr_repository_url)
aws ecr get-login-password --region eu-north-1 | docker login --username AWS --password-stdin ${ECR%%/*}
docker build -t $ECR:v1 ./app
docker push $ECR:v1

# 3) решта інфраструктури
cd terraform
terraform apply
terraform output -raw app_url
```

Видалення: `terraform destroy`.

## CI/CD

Workflow `.github/workflows/deploy.yml` запускається при push у `main` (зміни в `lab1/app/**`):

1. Автентифікація в AWS через OIDC: GitHub видає короткочасний токен, AWS перевіряє його за trust policy
   ролі (дозволено лише цей репозиторій і гілку `main`). Ключів у GitHub Secrets немає.
2. Збірка Docker-образу, тег — хеш коміту, публікація в ECR.
3. Нова ревізія task definition з новим образом і оновлення ECS-сервісу з очікуванням стабільності.

У GitHub потрібна одна змінна (Settings → Secrets and variables → Actions → Variables):
`AWS_ROLE_ARN` — виведення `terraform output -raw github_deploy_role_arn`.

Відкат: `aws ecs update-service --cluster track-catalog --service track-catalog --task-definition track-catalog:<ревізія>`.

## Безпека

- Пароль бази не зберігається в коді чи репозиторії (`random_password` → Secrets Manager).
- База без публічної адреси, доступ лише від security group застосунку.
- Застосунок приймає трафік лише від security group балансувальника.
- `terraform.tfstate`, `*.tfvars`, `.env` вказані в `.gitignore`.
