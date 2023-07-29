.PHONY: deps migration migrate

deps:
	curl -L https://packagecloud.io/golang-migrate/migrate/gpgkey | apt-key add -
	echo "deb https://packagecloud.io/golang-migrate/migrate/ubuntu/ xenial main" > /etc/apt/sources.list.d/migrate.list
	apt-get update
	apt-get install -y migrate

migration:
	migrate create -dir migrations -ext sql -seq $(name)

migrate:
	direnv allow
	migrate -path migrations -database "${DATABASE_URL}" goto ${VERSION}
