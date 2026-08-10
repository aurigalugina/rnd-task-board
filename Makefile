DB_URL=postgres://rndops:rndops@localhost:5432/rndops?sslmode=disable
MIGRATIONS=$(PWD)/backend/db/migrations
SEED=$(PWD)/backend/db/seed

.PHONY: up down up-dev down-dev migrate-up migrate-down seed seed-dev

up:
	docker compose up --build

down:
	docker compose down

# Versi dev: hot-reload (air buat backend, vite dev server buat frontend).
# Port sama seperti versi build (5432/8080), jadi jangan jalanin dua-duanya
# bersamaan -- `make down` dulu kalau stack build masih nyala.
# Frontend dev server: http://localhost:5173 (bukan :8080, karena gak ada nginx di sini)
up-dev:
	docker compose -f docker-compose.dev.yml up --build

down-dev:
	docker compose -f docker-compose.dev.yml down

# Menjalankan migration via image resmi golang-migrate, tanpa perlu install CLI lokal.
# Di Mac/Windows Docker Desktop, --network host mungkin perlu diganti dengan
# host.docker.internal pada DB_URL.
migrate-up:
	docker run --rm -v $(MIGRATIONS):/migrations --network host migrate/migrate \
		-path=/migrations -database "$(DB_URL)" up

migrate-down:
	docker run --rm -v $(MIGRATIONS):/migrations --network host migrate/migrate \
		-path=/migrations -database "$(DB_URL)" down 1

seed:
	docker exec -i $$(docker compose ps -q postgres) psql -U rndops -d rndops < $(SEED)/0001_seed_dev_user.sql

seed-dev:
	docker exec -i $$(docker compose -f docker-compose.dev.yml ps -q postgres) psql -U rndops -d rndops < $(SEED)/0001_seed_dev_user.sql
