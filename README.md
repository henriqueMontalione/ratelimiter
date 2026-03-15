# Rate Limiter

Rate limiter HTTP implementado em Go como middleware para controle de tráfego por IP e por token de acesso, com persistência em Redis.

## Funcionalidades

- Limitação de requisições por IP
- Limitação de requisições por token (`API_KEY` header)
- Token tem precedência absoluta sobre IP
- Bloqueio temporário configurável após exceder o limite
- Limites individuais por token via variável de ambiente
- Arquitetura baseada no padrão Strategy — troca de backend sem alterar lógica de negócio

## Pré-requisitos

- [Docker](https://docs.docker.com/get-docker/) e Docker Compose

## Início rápido

```bash
git clone https://github.com/henriquemontalione/ratelimiter.git
cd ratelimiter

cp .env.example .env

make up
```

A aplicação sobe na porta `8080` com Redis incluso.

## Desenvolvimento local

```bash
# Sobe apenas o Redis
docker compose up -d redis

# Roda a aplicação localmente
make run
```

## Comandos

| Comando | Descrição |
|---|---|
| `make up` | Sobe app + Redis via Docker |
| `make down` | Derruba os containers |
| `make run` | Roda a aplicação localmente |
| `make build` | Compila o binário |
| `make test` | Roda todos os testes |

## Variáveis de ambiente

Copie `.env.example` para `.env` e ajuste conforme necessário.

| Variável | Padrão | Descrição |
|---|---|---|
| `PORT` | `8080` | Porta da aplicação |
| `IP_RATE_LIMIT` | `10` | Máximo de requisições por segundo por IP |
| `TOKEN_RATE_LIMIT` | `20` | Máximo de requisições por segundo por token (padrão) |
| `TOKEN_LIMITS` | — | Limites individuais por token: `token1:100,token2:50` |
| `BLOCK_DURATION_SECONDS` | `300` | Tempo de bloqueio em segundos após exceder o limite |
| `REDIS_ADDR` | — | Endereço do Redis (obrigatório) ex: `redis:6379` |
| `REDIS_PASSWORD` | — | Senha do Redis (opcional) |
| `REDIS_DB` | `0` | Banco do Redis |

## Exemplos de uso

### Requisição por IP (sem token)

```bash
curl http://localhost:8080/
```

### Requisição com token

```bash
curl -H "API_KEY: meu-token" http://localhost:8080/
```

### Resposta de sucesso

```
HTTP/1.1 200 OK

Rate Limiter OK
```

### Resposta ao exceder o limite

```
HTTP/1.1 429 Too Many Requests

you have reached the maximum number of requests or actions allowed within a certain time frame
```

### Testando o bloqueio por IP

```bash
# Dispara 15 requisições em sequência (limite padrão: 10 req/s)
for i in $(seq 1 15); do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/
done
```

### Testando precedência de token sobre IP

```bash
# Mesmo com IP no limite, token com limite maior continua passando
curl -H "API_KEY: vip-key" http://localhost:8080/
```

## Arquitetura

O projeto segue **Hexagonal Architecture (Ports & Adapters)**:

| Camada | Pacote | Responsabilidade |
|---|---|---|
| Core | `internal/limiter/` | Lógica de negócio pura |
| Output Port | `internal/ports/` | Contratos que o core exige |
| Secondary Adapter | `internal/adapters/redis/` | Implementa ports com Redis |
| Primary Adapter | `internal/adapters/http/` | Expõe o core via HTTP (Gin) |

## Trocando o backend de persistência

O output port `Store` desacopla o core do mecanismo de armazenamento.

Para adicionar um novo backend (ex: in-memory):

1. Crie `internal/adapters/memory/store.go`
2. Implemente a interface definida em `internal/ports/store.go`:

```go
type Store interface {
    IsBlocked(ctx context.Context, key string) (bool, error)
    Increment(ctx context.Context, key string, windowSecs int) (int64, error)
    Block(ctx context.Context, key string, duration time.Duration) error
}
```

3. Injete o novo adapter em `cmd/server/main.go`

## Testes

```bash
# Todos os testes (requer Redis rodando)
make test

# Apenas testes unitários
go test ./internal/limiter/... -v

# Apenas testes de integração
go test ./tests/... -v
```

## Troubleshooting

**Redis não conecta**
- Verifique se `REDIS_ADDR` está configurado corretamente
- Confirme que o container está rodando: `docker compose ps`

**Testes falham com erro de conexão**
- Certifique-se que o Redis está acessível antes de rodar os testes
- Em caso de estado sujo: `redis-cli FLUSHDB`
