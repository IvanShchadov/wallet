# Wallet

A multi-currency wallet backend in Go with a double-entry ledger — a production-style pet project.

## Why

My pet project to show how a fintech backend is built end to end: money as integer minor units,
a double-entry ledger, transfers in a single database transaction, idempotent requests,
events via Kafka with the outbox pattern, and observability. It grows in small working stages —
the roadmap shows where it is now.

## Getting started

## Roadmap

<!-- Строки = этапы плана, по порядку -->
- [x] Repository setup and healthcheck
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
