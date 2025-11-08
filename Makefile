export GO111MODULE=on
export GOPROXY=https://proxy.golang.org,direct

SHELL := /bin/bash

API_DIR = backend

.PHONY: build static clean api run_api

# Build code according to caller OS and architecture
build:
	make api

# Build API
api:
	$(MAKE) -C $(API_DIR) api

# Run API server locally
run_api:
	$(MAKE) -C $(API_DIR) run

# Clean API
clean-api:
	$(MAKE) -C $(API_DIR) clean

# Delete all compiled binaries
clean:
	make clean-api

# Display systemd logs for API server
logs_api:
	sudo journalctl -f -t $(API_NAME)

# Install API
# optional DEST=destination_path
install:
	make clean
	make build
	make install_api

# Install API server and restart service
# optional DEST=destination_path
install_api:
	sudo systemctl stop $(API_NAME)
	sudo cp $(OUTPUT)/$(API_NAME) $(DEST)
	sudo systemctl start $(API_NAME)

# Display docker logs for API server
docker_dev_logs_api:
	docker logs -f $(API_NAME)-dev

# Display docker logs for postgresql server
docker_dev_logs_postgresql:
	docker logs -f mapctf-postgres-dev

# Display docker logs for redis server
docker_dev_logs_redis:
	docker logs -f mapctf-redis-dev

# Docker shell into API server
docker_dev_shell_api:
	docker exec -it $(API_NAME)-dev /bin/bash

# Docker shell into postgresql server
docker_dev_shell_postgres:
	docker exec -it mapctf-postgres-dev /bin/bash

# Docker shell into redis server
docker_dev_shell_redis:
	docker exec -it mapctf-redis-dev /bin/sh

# Build dev docker containers and run them
docker_dev_build:
ifeq (,$(wildcard ./.env))
	$(error Missing .env file)
endif
	docker-compose -f docker-compose-dev.yml build

# Build and run dev docker containers
make docker_dev:
	make docker_dev_build
	make docker_dev_up

# Run docker containers
docker_dev_up:
	docker-compose -f docker-compose-dev.yml up

up-backend:
	docker-compose -f docker-compose-dev.yml up mapctf-postgres mapctf-redis

# Takes down docker containers
docker_dev_down:
	docker-compose -f docker-compose-dev.yml down

# Deletes all mapctf docker images
docker_dev_clean:
	docker images | grep mapctf | awk '{print $$3}' | xargs -rI {} docker rmi -f {}

# Rebuild only the API server
docker_dev_rebuild_api:
	docker-compose -f docker-compose-dev.yml up --force-recreate --no-deps -d --build $(API_NAME)

# Run linter
lint:
	golangci-lint run
