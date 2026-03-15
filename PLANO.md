# Plano de Arquitetura — Rate Limiter

## Decisões de arquitetura

### Padrão Strategy para persistência

A interface `Store` isola completamente a lógica de negócio do mecanismo de armazenamento. O `RateLimiter` depende da abstração, não da implementação concreta. Trocar Redis por qualquer outro backend exige apenas uma nova struct que implemente `Store` — sem tocar em `limiter.go`.

```go
type Store interface {
    IsBlocked(ctx context.Context, key string) (bool, error)
    Increment(ctx context.Context, key string, windowSecs int) (int64, error)
    Block(ctx context.Context, key string, duration time.Duration) error
}
```

### Separação de responsabilidades

- `internal/limiter` contém a regra de negócio pura — sem nenhuma dependência de HTTP
- `internal/middleware` é a cola entre HTTP e o limiter — sem regra de negócio
- Essa separação permite testar o limiter com um store mockado, sem levantar servidor

### Algoritmo: Fixed Window

Escolhido por simplicidade e suficiência para a especificação. O contador tem TTL de 1 segundo no Redis. Ao expirar, a janela reseta naturalmente sem nenhuma lógica adicional.

Trade-off conhecido: o Fixed Window pode permitir o dobro do limite na virada de janela (burst na junção de duas janelas). Para este projeto, é aceitável — o Sliding Window adicionaria complexidade (sorted sets) sem benefício justificado pela spec.

### Atomicidade via Lua script

O incremento e a definição de TTL precisam ser atômicos para evitar race conditions em ambientes concorrentes. Dois comandos Redis separados (INCR + EXPIRE) criariam uma janela de inconsistência. A solução é um Lua script executado como transação no Redis:

```lua
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
```

### Precedência Token > IP

Quando o header `API_KEY` está presente, o sistema usa exclusivamente o limite do token. O IP não é consultado. Isso simplifica o fluxo e garante a regra de ouro da spec sem ambiguidade.

---

## Estratégia de chaves Redis

| Chave | TTL | Propósito |
|---|---|---|
| `rl:counter:ip:{ip}` | 1s | contador de janela por IP |
| `rl:counter:token:{token}` | 1s | contador de janela por token |
| `rl:blocked:ip:{ip}` | `BLOCK_DURATION_SECONDS` | flag de bloqueio por IP |
| `rl:blocked:token:{token}` | `BLOCK_DURATION_SECONDS` | flag de bloqueio por token |

O prefixo `rl:` isola as chaves do limiter de outras aplicações que possam compartilhar o mesmo Redis.

---

## Fluxo de decisão — `limiter.Allow(ip, token)`

```
Token presente?
├── SIM → chave = "token:{token}"
│         limite = limite individual do token ou TOKEN_RATE_LIMIT
└── NÃO → chave = "ip:{ip}"
          limite = IP_RATE_LIMIT

Chave está bloqueada?
└── SIM → retorna false

INCR atômico (Lua script)
└── count > limite?
    └── SIM → Block(chave, BLOCK_DURATION) → retorna false

retorna true
```

---

## Configuração de limites individuais por token

Tokens individuais são configurados via `TOKEN_LIMITS` no formato `token:limite,token:limite`. O `config.go` faz o parse para um `map[string]int` na inicialização. Tokens não listados usam `TOKEN_RATE_LIMIT` como fallback.

---

## Arquitetura Hexagonal

O projeto segue o padrão Ports & Adapters (Hexagonal Architecture):

- **Core (domínio)**: `internal/limiter/` — lógica pura, sem dependências externas
- **Output Port**: `internal/ports/store.go` — contrato que o core exige do mundo externo
- **Secondary Adapter**: `internal/adapters/redis/` — implementa o port usando Redis
- **Primary Adapter**: `internal/adapters/http/` — expõe o core via HTTP (Gin middleware)

```
cmd/server/main.go                  → entry point, wiring de dependências
internal/config/config.go           → struct Config, parse via caarlos0/env
internal/ports/store.go             → output port (interface Store)
internal/adapters/redis/store.go    → secondary adapter (Redis)
internal/adapters/http/middleware.go → primary adapter (Gin)
internal/limiter/limiter.go         → core/domínio (RateLimiter, método Allow)
internal/limiter/limiter_test.go    → testes unitários com mock do port
tests/ratelimiter_test.go           → testes de integração
```
