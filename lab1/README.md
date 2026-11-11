# Лабораторна 1 

REST API для каталогу треків і плейлистів (Go + PostgreSQL).

**Публічна адреса:** _буде після деплою_

## Структура
- `app/` — код застосунку і Dockerfile
- `terraform/` — інфраструктура як код
- `compose.yaml` — локальний запуск застосунку з базою
- `.env.example` — шаблон змінних середовища

## Локальний запуск
```bash
cd lab1
cp .env.example .env 
docker compose up --build
```

## Ендпоінти
- `GET /health`
- `GET, POST /playlists`, `GET /playlists/{id}/tracks`
- `GET, POST /tracks`, `GET, PUT, DELETE /tracks/{id}`
