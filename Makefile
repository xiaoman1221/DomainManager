APP_NAME := domain-manager

.PHONY: all build-frontend build-backend clean

all: build-frontend build-backend

build-frontend:
	cd web && npm install && npm run build

build-backend:
	go build -o $(APP_NAME) .

clean:
	rm -f $(APP_NAME)
	rm -rf web/dist
