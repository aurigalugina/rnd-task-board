DSN=mysql://myagenda:myagenda@tcp(localhost:3307)/myagenda
MIGRATIONS=$(PWD)/db/migrations

.PHONY: up down migrate-up migrate-down

up:
	docker compose up --build

down:
	docker compose down

# Migration via image resmi golang-migrate, tanpa perlu install CLI lokal.
# Di Mac/Windows Docker Desktop, --network host mungkin perlu diganti dengan
# host.docker.internal pada DSN.
migrate-up:
	docker run --rm -v $(MIGRATIONS):/migrations --network host migrate/migrate \
		-path=/migrations -database "$(DSN)" up

migrate-down:
	docker run --rm -v $(MIGRATIONS):/migrations --network host migrate/migrate \
		-path=/migrations -database "$(DSN)" down 1
