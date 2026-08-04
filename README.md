# testtask

Сервис задач с командами. Go + MySQL + Redis.

```bash
docker compose up --build
```

```bash
curl -X POST localhost:8080/api/v1/register -d '{"email":"a@b.com","password":"pass1234","name":"Alice"}'
curl -X POST localhost:8080/api/v1/login -d '{"email":"a@b.com","password":"pass1234"}'

curl -X POST localhost:8080/api/v1/teams -H "Authorization: Bearer <token>" -d '{"name":"Dev Team"}'
curl -X POST localhost:8080/api/v1/tasks -H "Authorization: Bearer <token>" -d '{"team_id":1,"title":"First task"}'
curl "localhost:8080/api/v1/tasks?team_id=1&status=todo" -H "Authorization: Bearer <token>"
```

Тесты: `go test ./...` (интеграционные — `-tags integration`, нужен Docker).
