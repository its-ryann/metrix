# Stage 1: Build workspace
FROM python:3.11-slim AS builder

WORKDIR /app

COPY collector.py .

# Stage 2: Runtime image
FROM python:3.11-slim

RUN groupadd -r appgroup && useradd -r -g appgroup appuser

WORKDIR /app

COPY --from=builder /app /app

RUN chown -R appuser:appgroup /app

USER appuser

CMD ["python", "-u", "collector.py"]