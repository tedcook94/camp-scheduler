FROM golang:1.26

RUN go install github.com/air-verse/air@latest

ADD https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz /tmp/migrate.tar.gz
RUN tar -xzf /tmp/migrate.tar.gz -C /usr/local/bin && rm /tmp/migrate.tar.gz

RUN git config --global --add safe.directory /app

WORKDIR /app

CMD ["sh", "-c", "migrate -database \"$DATABASE_URL\" -path database/migrations up && migrate -database \"$DATABASE_URL_TEST\" -path database/migrations up && air"]
