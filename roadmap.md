# GoPay Processing Engine - Project Roadmap & Progress

Este arquivo documenta o progresso de desenvolvimento dos 12 estágios da aplicação.

## Progresso Atual

* **[x] Stage 1: Basic Project Setup, Configuration & Logger**
  - [x] Inicialização do módulo Go (`github.com/aless/gopay-processing-engine`)
  - [x] Estrutura inicial de pastas (`cmd`, `internal`, `pkg`)
  - [x] Carregamento de configurações de variáveis de ambiente (`internal/config`)
  - [x] Integração de logging estruturado com Uber Zap (`pkg/logger`)

* **[x] Stage 2: Domain Layer & DB Migrations (PostgreSQL)**
  - [x] Modelos de domínio core: `User`, `Payment`, `OutboxEvent` (`internal/domain`)
  - [x] Scripts de migração SQL up/down (`migrations/000001_init_schema`)
  - [x] Implementação de repositórios Postgres usando `database/sql` com suporte a transação (`internal/infrastructure/postgres`)
  - [x] Configuração e verificação do pool de conexão com PostgreSQL

* **[x] Stage 3: Redis Integration (Cache & Idempotency)**
  - [x] Integração do cliente go-redis (`internal/infrastructure/redis/redis.go`)
  - [x] Implementação de cache de consultas de pagamentos (`payment_cache.go`)
  - [x] Implementação de gerenciador de idempotência atômica via Redis `SetNX` (`idempotency_manager.go`)

* **[x] Stage 4: Authentication & Security (JWT)**
  - [x] Funções de hash e verificação de senha com Bcrypt (`pkg/security/security.go`)
  - [x] Serviço de geração e validação de tokens JWT (`pkg/security/token.go`)
  - [x] Middleware de autenticação JWT para rotas do Gin (`internal/middleware/auth.go`)

* **[x] Stage 5: Messaging Layer (Kafka Producer & Consumer)**
  - [x] Instalação do pacote `github.com/segmentio/kafka-go`
  - [x] Criação da interface de domínio `EventPublisher` (`internal/domain/event.go`)
  - [x] Implementação de Kafka Producer com configuração de acks e timeouts (`internal/infrastructure/kafka/producer.go`)
  - [x] Implementação de Kafka Consumer reutilizável com commits manuais e tratamento de erros (`internal/infrastructure/kafka/consumer.go`)

* **[x] Stage 6: Application Layer & Gateway (Outbox, Circuit Breaker, FakeGateway)**
  - [x] Adicionado Circuit Breaker usando o pacote `github.com/sony/gobreaker/v2`
  - [x] Definido contrato de interface `PaymentGateway` em `internal/domain/gateway.go`
  - [x] Implementado Fake Gateway com simulação de latência e taxa de erro em `internal/infrastructure/gateway/fake_gateway.go`
  - [x] Criado wrapper de resiliência em `internal/infrastructure/gateway/circuit_breaker.go`
  - [x] Criado o `PaymentService` coordenando transações e idempotência (`internal/application/payment_service.go`)
  - [x] Criado o processador em background do outbox transactional (`internal/application/outbox_processor.go`)

* **[x] Stage 7: Worker Service (Async Processor, Retry & DLQ)**
  - [x] Criado o componente PaymentWorker gerenciando o ciclo de processamento assíncrono (`internal/application/payment_worker.go`)
  - [x] Implementada política de retentativas com backoff exponencial para falhas temporárias
  - [x] Implementado roteamento automático para Fila de Erros (Dead Letter Queue - DLQ)
  - [x] Desenvolvido o entrypoint executável do daemon do worker em `cmd/worker/main.go`

* **[x] Stage 8: API Layer (Gin Handlers & Endpoints)**
  - [x] Criado o componente AuthService para validação e login (`internal/application/auth_service.go`)
  - [x] Criados os Handlers HTTP para autenticação e rotas protegidas de pagamentos (`internal/interfaces/http`)
  - [x] Configurado o Router Gin com tratamento de CORS e middleware JWT (`internal/interfaces/http/router.go`)
  - [x] Atualizado o entrypoint executável da API em `cmd/api/main.go` com inicialização de conexões e encerramento gracioso
  - [x] Integrado e executado o OutboxProcessor em background no start da API

* **[x] Stage 9: Observability Integration (OTel, Prometheus, Grafana)**
  - [x] Configurado exportador OTel Tracer enviando spans para o Jaeger (`pkg/telemetry/telemetry.go`)
  - [x] Implementados coletores customizados de métricas do Prometheus para requisições HTTP e worker assíncrono
  - [x] Criado servidor HTTP em background para scraping de métricas (porta `2112` na API, `2113` no Worker)
  - [x] Criado middleware do Gin para registro automatizado de quantidade e latência das rotas HTTP (`pkg/telemetry/middleware.go`)
  - [x] Instrumentado o PaymentWorker para observar latência de pagamentos e volumes de erros/DLQ

* **[x] Stage 10: Dockerization & Build Controls (Makefile & Compose)**
  - [x] Desenvolvido o `Dockerfile` multi-stage compilando e expondo as imagens da API e do Worker em imagens Alpine
  - [x] Criado o `docker-compose.yml` orquestrando Postgres, Redis, Kafka, Zookeeper, Jaeger, Prometheus, API e Worker
  - [x] Configurado scraping do Prometheus em `docker/prometheus/prometheus.yml` monitorando as portas das duas aplicações
  - [x] Implementadas anotações do Swaggo nos handlers de autenticação e pagamentos
  - [x] Gerado pacote de documentação no diretório `/docs` via `swag init`
  - [x] Mapeada rota `/swagger/*any` no router HTTP para disponibilizar a interface Swagger UI
  - [x] Criado `Makefile` com atalhos de compilação, execução, testes, build docker e gerenciamento do compose

---

## Próximos Passos (Pendentes)

* **[x] Stage 11: Tests (Unit & Integration)**
  - [x] Configurado `testify` para asserções e mocks de repositório (`internal/domain/mocks`)
  - [x] Desenvolvidos testes unitários para o domínio de pagamentos e validação de estado
  - [x] Implementados testes do pacote de segurança (Bcrypt, JWT expirado e assinaturas)
  - [x] Escritos testes para `AuthService` e `PaymentService` com simulação de transações via `go-sqlmock`
  - [x] Desenvolvidos testes para Workers e processador do Outbox assegurando ciclos de vida, retry e DLQ
  - [x] Implementados testes de controladores HTTP via Gin TestRecorder
  - [x] Desenvolvidos testes de integração de repositório PostgreSQL e Redis Cache com `miniredis`
  - [x] Criado script portátil `scripts/check_coverage.go` e regra `test-coverage` no Makefile
  - [x] Configurada Pipeline CI/CD com GitHub Actions executando builds, testes e exigindo cobertura de **80%**

* **[x] Stage 12: Documentation & README**
  - [x] Criado o arquivo `README.md` completo e formatado com badges de tecnologias
  - [x] Documentados os fluxos de criação de pagamento e processamento assíncrono via diagrama sequencial Mermaid
  - [x] Documentada a árvore estrutural de diretórios e padrões de Clean Architecture do projeto
  - [x] Incluído guia de início rápido e execução via Docker Compose
  - [x] Mapeados todos os comandos de atalho de desenvolvimento via `Makefile`
  - [x] Adicionado guia para acesso e execução da interface interativa do Swagger UI
  - [x] Mapeados os endpoints de rastreamento distribuído (Jaeger) e coleta de métricas (Prometheus)

