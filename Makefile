export GO111MODULE=on
export GOPROXY=https://proxy.golang.org,direct

SHELL := /bin/bash

BACKEND_DIR = backend

API_NAME = mapctf-api
MAP_NAME = mapctf-map

.PHONY: build static clean api run_api map run_map

# Build code according to caller OS and architecture
build:
	make api map

# Build API
api:
	$(MAKE) -C $(BACKEND_DIR) api

# Build map
map:
	$(MAKE) -C $(BACKEND_DIR) api

# Run API server locally
run_api:
	$(MAKE) -C $(BACKEND_DIR) run

# Clean API
clean-api:
	$(MAKE) -C $(BACKEND_DIR) clean

# Run map server locally
run_map:
	$(MAKE) -C $(BACKEND_DIR) run

# Clean map
clean-map:
	$(MAKE) -C $(BACKEND_DIR) clean

# Delete all compiled binaries
clean:
	make clean-api
	make clean-map

# Display systemd logs for API server
logs_api:
	sudo journalctl -f -t $(API_NAME)

# Display systemd logs for map server
logs_map:
	sudo journalctl -f -t $(MAP_NAME)

# Install API
# optional DEST=destination_path
install:
	make clean
	make build
	make install_api
	make install_map

# Install API server and restart service
# optional DEST=destination_path
install_api:
	sudo systemctl stop $(API_NAME)
	sudo cp $(OUTPUT)/$(API_NAME) $(DEST)
	sudo systemctl start $(API_NAME)

# Install map server and restart service
# optional DEST=destination_path
install_map:
	sudo systemctl stop $(MAP_NAME)
	sudo cp $(OUTPUT)/$(MAP_NAME) $(DEST)
	sudo systemctl start $(MAP_NAME)

# Display docker logs for API server
docker_dev_logs_api:
	docker logs -f $(API_NAME)-dev

# Display docker logs for map server
docker_dev_logs_map:
	docker logs -f $(MAP_NAME)-dev

# Display docker logs for postgresql server
docker_dev_logs_postgresql:
	docker logs -f mapctf-postgres-dev

# Display docker logs for redis server
docker_dev_logs_redis:
	docker logs -f mapctf-redis-dev

# Docker shell into API server
docker_dev_shell_api:
	docker exec -it $(API_NAME)-dev /bin/bash

# Docker shell into map server
docker_dev_shell_map:
	docker exec -it $(MAP_NAME)-dev /bin/bash

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
	docker images --format table | grep mapctf | awk '{print $$3}' | xargs -rI {} docker rmi -f {}

# Rebuild only the API server
docker_dev_rebuild_api:
	docker-compose -f docker-compose-dev.yml up --force-recreate --no-deps -d --build $(API_NAME)

# Rebuild only the map server
docker_dev_rebuild_map:
	docker-compose -f docker-compose-dev.yml up --force-recreate --no-deps -d --build $(MAP_NAME)

# Run linter
lint:
	golangci-lint run

# Test with coverage
test:
	$(MAKE) -C $(BACKEND_DIR) test
