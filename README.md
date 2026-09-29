# Wallet

<!-- Одна фраза: что это и на чём написано. По ней EM решает, читать ли дальше -->
TODO

## Why

<!-- 2–3 предложения своими словами: зачем проект и что он показывает. Прямо назови его pet project -->
TODO

## Roadmap

<!-- Строки = этапы плана, по порядку -->
- [ ] Repository setup and healthcheck
- [ ] Domain: money, accounts, double-entry ledger
- [ ] REST API with in-memory storage
- [ ] PostgreSQL: transfers in a single transaction, migrations
- [ ] Idempotent transfers (`Idempotency-Key`)
- [ ] Exchange rates and currency exchange via quotes
- [ ] Kafka events with outbox, notifier service
- [ ] React UI: accounts, transfers, history, live rates
- [ ] Observability: logs, metrics, dashboards, alerts
- [ ] Rates as a separate gRPC service, CI, AWS deploy

## Out of scope

<!-- Чего нет намеренно. Так видно, что граница — решение, а не недоделка -->
- Real payment providers and cards
- KYC
