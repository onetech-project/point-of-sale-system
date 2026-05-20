FROM migrate/migrate:v4.18.3

LABEL org.opencontainers.image.title="POS database migrations"
LABEL org.opencontainers.image.description="Schema migration image for the Point of Sale PostgreSQL database"

ENV MIGRATIONS_PATH=/migrations

WORKDIR /migrations

COPY *.sql /migrations/

RUN set -eux; \
    { \
        echo '#!/bin/sh'; \
        echo 'set -eu'; \
        echo ''; \
        echo 'MIGRATIONS_PATH="${MIGRATIONS_PATH:-/migrations}"'; \
        echo ''; \
        echo 'if [ "$#" -eq 0 ]; then'; \
        echo '    set -- up'; \
        echo 'fi'; \
        echo ''; \
        echo 'case "$1" in'; \
        echo '    -*)'; \
        echo '        exec migrate -path="$MIGRATIONS_PATH" "$@"'; \
        echo '        ;;'; \
        echo 'esac'; \
        echo ''; \
        echo 'if [ -z "${DATABASE_URL:-}" ]; then'; \
        echo '    echo "DATABASE_URL is required when running migrate commands without -database." >&2'; \
        echo '    echo "Example: docker run --rm -e DATABASE_URL=postgresql://user:pass@host:5432/db?sslmode=disable IMAGE" >&2'; \
        echo '    echo "Example: docker run --rm IMAGE -database postgresql://user:pass@host:5432/db?sslmode=disable up" >&2'; \
        echo '    exit 1'; \
        echo 'fi'; \
        echo ''; \
        echo 'exec migrate -path="$MIGRATIONS_PATH" -database="$DATABASE_URL" "$@"'; \
    } > /usr/local/bin/run-migrations; \
    chmod +x /usr/local/bin/run-migrations

ENTRYPOINT ["/usr/local/bin/run-migrations"]
