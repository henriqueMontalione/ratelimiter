# CLAUDE.md

## Papel e contexto

Você está implementando um rate limiter HTTP em Go. Siga estas instruções à risca em toda interação com este projeto.

---

## Regras de código — NUNCA faça

- Nunca use `panic` em código de aplicação
- Nunca ignore erros — trate todos explicitamente
- Nunca hardcode valores de configuração
- Nunca adicione comentários que apenas repetem o que o código já diz
- Nunca adicione dependências externas sem necessidade clara
- Nunca escreva logs desnecessários — apenas o que agrega valor em produção
- Nunca use `IStore`, `StoreInterface` ou sufixos redundantes em interfaces
- Nunca implemente lógica HTTP dentro de `internal/limiter`
- Nunca implemente lógica de negócio dentro de `internal/middleware`

---

## Convenções obrigatórias

### Nomenclatura
- Interfaces: nome descritivo simples — `Store`, não `IStore`
- Implementações: nome concreto + tipo — `RedisStore`
- Construtores: sempre `New{Type}` — `NewRateLimiter`, `NewRedisStore`
- Métodos: verbos claros — `Allow`, `IsBlocked`, `Increment`, `Block`

### Go idiomático
- Use `context.Context` em todas as operações de I/O
- Prefira composição
- Interfaces pequenas — no máximo 3 métodos
- Nomes que dispensam comentários
- Siga Effective Go, Go Code Review Comments e Google Go Style Guide

### Arquitetura Hexagonal — Ports & Adapters

```
internal/limiter/            → CORE: lógica de negócio pura, zero dependência externa
internal/ports/              → OUTPUT PORT: interfaces que o core exige
internal/adapters/redis/     → SECONDARY ADAPTER: implementa ports usando Redis
internal/adapters/http/      → PRIMARY ADAPTER: expõe o core via HTTP (Gin)
internal/config/             → configuração via variáveis de ambiente
cmd/server/                  → entry point, wiring de todas as dependências
tests/                       → testes de integração com Redis real
```

Respeite os limites de cada camada:
- `limiter/` não importa nada de `adapters/` nem pacotes HTTP
- `ports/` não importa nada de `adapters/`
- `adapters/http/` não contém regra de negócio — apenas tradução HTTP ↔ core

### Regra crítica de negócio
Token presente no header `API_KEY` → ignora IP completamente. A precedência é absoluta.

### Resposta 429 — texto exato, imutável
```
you have reached the maximum number of requests or actions allowed within a certain time frame
```

---

## Checklist antes de cada commit

- [ ] Todos os erros estão sendo tratados
- [ ] Nenhum valor hardcoded — tudo vem de `Config`
- [ ] `context.Context` passado em todas as operações de I/O
- [ ] Limites de pacote respeitados (limiter sem HTTP, middleware sem regra de negócio)
- [ ] Testes adicionados ou atualizados para a feature
- [ ] `go mod tidy` rodado
- [ ] `make test` passou sem erros

---

## Git workflow

### Branches
- Crie uma branch por feature a partir da `main`
- Nomenclatura: `feat/`, `fix/`, `test/`, `chore/`, `docs/`
- Após aprovação do usuário, faça merge na `main`
- A próxima branch sempre parte da `main` atualizada

### Fluxo
1. `git checkout main`
2. `git checkout -b feat/nome-da-feature`
3. Implemente em commits atômicos
4. Adicione ou atualize os testes da feature antes de commitar
5. Garanta que `make test` passa
6. Apresente os arquivos ao usuário para aprovação
7. `git add <arquivos específicos>` — nunca `git add .`
8. Commit após aprovação explícita do usuário
9. Merge na `main`

### Commits
- Mensagens em inglês, Conventional Commits
- `feat:` `fix:` `test:` `chore:` `docs:`
- Um commit = uma mudança lógica
- Nunca mencionar Claude ou IA na mensagem

---

## Notas críticas de implementação

### Atomicidade no Redis
Increment e verificação de limite DEVEM ser atômicos. Use Lua script via `redis.NewScript()`.
Nunca faça INCR + verificação em chamadas separadas — race condition garantida.

---

## Dependências do projeto

```
github.com/gin-gonic/gin          # HTTP framework
github.com/redis/go-redis/v9      # Redis client
github.com/joho/godotenv          # carrega .env
github.com/caarlos0/env/v11       # parse de env vars em struct tipada
github.com/stretchr/testify       # assertions nos testes
```

Adicione dependências com `go get`, finalize com `go mod tidy`.
