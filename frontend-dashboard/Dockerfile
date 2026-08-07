# Stage 1: File staging
FROM alpine:3.19 AS builder

WORKDIR /app

COPY index.html styles.css app.js ./

# Stage 2: Web server runtime
FROM nginx:alpine

# Configure nginx permissions for non-root execution
RUN touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid /var/cache/nginx /var/log/nginx /usr/share/nginx/html

# Copy assets from builder stage
COPY --from=builder /app/ /usr/share/nginx/html/

# Switch to standard unprivileged nginx user
USER nginx

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]