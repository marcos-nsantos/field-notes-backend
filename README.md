# Field Notes Backend

Backend API para aplicação mobile de notas de campo com sincronização offline-first.

## Tech Stack

- **Go 1.25+** com Gin framework
- **PostgreSQL** com PostGIS para dados geoespaciais
- **Redis** para rate limiting
- **MinIO/S3** para armazenamento de imagens
- **JWT** para autenticação

## Funcionalidades

- Autenticação com JWT e refresh tokens
- CRUD de notas com geolocalização
- Sincronização offline-first (last-write-wins)
- Upload de imagens com compressão
- Rate limiting distribuído
- Documentação Swagger

## Requisitos

- Go 1.25+
- Docker e Docker Compose
- Make (opcional)

## Começar

### 1. Clonar e configurar

```bash
git clone https://github.com/marcos-nsantos/field-notes-backend.git
cd field-notes-backend
cp .env.example .env
```

### 2. Iniciar serviços

```bash
docker-compose up -d postgres redis minio createbucket
```

### 3. Executar migrações

```bash
make migrate-up
```

### 4. Iniciar servidor

```bash
make run
```

O servidor estará disponível em `http://localhost:8080`.

Documentação Swagger: `http://localhost:8080/swagger/index.html`

## Endpoints da API

### Autenticação

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/api/v1/auth/register` | Registar novo utilizador |
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/refresh` | Renovar access token |
| POST | `/api/v1/auth/logout` | Logout (requer auth) |

### Notas

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/api/v1/notes` | Listar notas (paginado, filtro por bbox) |
| POST | `/api/v1/notes` | Criar nota |
| GET | `/api/v1/notes/:id` | Obter nota por ID |
| PUT | `/api/v1/notes/:id` | Atualizar nota |
| DELETE | `/api/v1/notes/:id` | Eliminar nota (soft delete) |

### Sincronização

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/api/v1/sync` | Sincronizar notas (batch) |

### Upload

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/api/v1/upload/:note_id` | Upload de imagem para nota |
| DELETE | `/api/v1/photos/:id` | Eliminar foto |

## Configuração

Variáveis de ambiente (ver `.env.example`):

| Variável | Descrição | Default |
|----------|-----------|---------|
| `SERVER_PORT` | Porta do servidor | 8080 |
| `DB_HOST` | Host PostgreSQL | localhost |
| `DB_PORT` | Porta PostgreSQL | 5432 |
| `DB_USER` | Utilizador PostgreSQL | - |
| `DB_PASSWORD` | Password PostgreSQL | - |
| `DB_NAME` | Nome da base de dados | - |
| `JWT_SECRET_KEY` | Chave secreta JWT | - |
| `JWT_ACCESS_TOKEN_TTL` | TTL do access token | 15m |
| `JWT_REFRESH_TOKEN_TTL` | TTL do refresh token | 720h |
| `REDIS_HOST` | Host Redis | localhost |
| `REDIS_PORT` | Porta Redis | 6379 |
| `RATE_LIMIT_ENABLED` | Ativar rate limiting | true |
| `RATE_LIMIT_REQUESTS_PER_MIN` | Requests por minuto | 100 |
| `S3_ENDPOINT` | Endpoint S3/MinIO | - |
| `S3_BUCKET` | Bucket S3 | - |
| `S3_ACCESS_KEY_ID` | Access key S3 | - |
| `S3_SECRET_ACCESS_KEY` | Secret key S3 | - |

## Desenvolvimento

### Comandos Make

```bash
make run          # Executar servidor
make build        # Compilar binário
make test         # Executar testes
make test-e2e     # Executar testes e2e
make lint         # Executar linter
make migrate-up   # Aplicar migrações
make migrate-down # Reverter migrações
make swagger      # Gerar documentação Swagger
make tools        # Instalar ferramentas de desenvolvimento
```

### Estrutura do Projeto

```
├── cmd/api/              # Entrypoint da aplicação
├── internal/
│   ├── adapter/
│   │   ├── handler/      # HTTP handlers e DTOs
│   │   └── repository/   # Implementações de repositório
│   ├── domain/           # Entidades e value objects
│   ├── infrastructure/   # Config, middleware, database, etc.
│   ├── pkg/              # Utilitários partilhados
│   └── usecase/          # Lógica de negócio
├── migrations/           # Migrações SQL
├── test/e2e/             # Testes end-to-end
└── docs/                 # Documentação Swagger gerada
```

### Testes

```bash
# Testes unitários
make test

# Testes e2e (requer Docker)
make test-e2e

# Cobertura
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Protocolo de Sincronização

O cliente envia notas modificadas desde o último sync:

```json
{
  "device_id": "uuid",
  "sync_cursor": "2024-01-01T00:00:00Z",
  "notes": [
    {
      "client_id": "uuid",
      "title": "Nota",
      "content": "Conteúdo",
      "latitude": 38.7223,
      "longitude": -9.1393,
      "updated_at": "2024-01-02T10:00:00Z",
      "is_deleted": false
    }
  ]
}
```

O servidor responde com notas do servidor e resolução de conflitos:

```json
{
  "server_notes": [...],
  "new_cursor": "2024-01-02T12:00:00Z",
  "conflicts": [
    {
      "client_id": "uuid",
      "resolution": "server_wins"
    }
  ]
}
```

Estratégia: **Last Write Wins** - a versão com `updated_at` mais recente prevalece.

## Licença

MIT
