# Stage 1: File staging
FROM alpine:3.19 AS builder

WORKDIR /app

COPY index.html .

# Stage 2: Web server runtime
FROM nginx:alpine

# Configure nginx permissions for non-root execution
RUN touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid /var/cache/nginx /var/log/nginx /usr/share/nginx/html

# Copy index.html from builder stage
COPY --from=builder /app/index.html /usr/share/nginx/html/index.html

# Switch to standard unprivileged nginx user
USER nginx

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]