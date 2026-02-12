sendflix/
├── cmd/
│ ├── api/ # HTTP API server
│ │ └── main.go
│ ├── grpc/ # gRPC server
│ │ └── main.go
│ └── cli/ # CLI application
│ └── main.go
├── internal/
│ ├── domain/ # Domain layer
│ │ ├── email.go # Email entity
│ │ ├── template.go # Template entity
│ │ ├── provider.go # Provider interface
│ │ └── repository.go # Repository interfaces
│ ├── usecase/ # Use case layer
│ │ ├── email/
│ │ │ ├── send.go
│ │ │ ├── send_bulk.go
│ │ │ ├── schedule.go
│ │ │ └── get.go
│ │ └── template/
│ │ ├── create.go
│ │ ├── update.go
│ │ ├── delete.go
│ │ ├── get.go
│ │ ├── preview.go
│ │ └── activate.go
│ ├── infrastructure/ # Infrastructure layer
│ │ ├── database/
│ │ │ └── postgres/
│ │ │ ├── connection.go
│ │ │ ├── email_repository.go
│ │ │ └── template_repository.go
│ │ ├── provider/
│ │ │ ├── smtp/
│ │ │ │ ├── provider.go
│ │ │ │ ├── pool.go
│ │ │ │ └── composer.go
│ │ │ ├── ses/
│ │ │ │ └── provider.go
│ │ │ └── sendgrid/
│ │ │ └── provider.go
│ │ └── cache/
│ │ └── redis/
│ │ └── cache.go
│ ├── delivery/ # Delivery layer
│ │ ├── http/
│ │ │ ├── server.go
│ │ │ ├── handler/
│ │ │ │ ├── email.go
│ │ │ │ └── template.go
│ │ │ └── middleware/
│ │ │ ├── logger.go
│ │ │ └── cors.go
│ │ ├── grpc/
│ │ │ └── server.go
│ │ └── cli/
│ │ └── commands.go
│ └── worker/ # Background workers
│ ├── scheduler.go
│ ├── retry.go
│ └── cleanup.go
├── pkg/ # Public packages
│ ├── config/ # Configuration
│ │ └── config.go
│ ├── logger/ # Logging
│ │ └── logger.go
│ ├── metrics/ # Metrics
│ │ └── metrics.go
│ ├── errors/ # Error types
│ │ └── errors.go
│ └── utils/ # Utilities
│ └── utils.go
├── api/
│ └── proto/ # Protobuf definitions
│ └── sendflix.proto
├── migrations/ # Database migrations
│ ├── 001_create_emails.up.sql
│ ├── 001_create_emails.down.sql
│ ├── 002_create_templates.up.sql
│ └── 002_create_templates.down.sql
├── config/ # Configuration files
│ ├── config.yaml
│ └── config.prod.yaml
├── templates/ # Email templates
│ ├── layouts/
│ │ └── base.html
│ └── welcome.html
├── docker/
│ ├── Dockerfile
│ └── docker-compose.yaml
├── scripts/ # Build scripts
│ ├── build.sh
│ └── migrate.sh
├── go.mod
├── go.sum
├── Makefile
└── README.md
