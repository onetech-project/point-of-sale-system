# Build stage
FROM node:22.20-alpine AS builder

WORKDIR /app

# Define build arguments
ARG NEXT_PUBLIC_API_URL
ARG API_GATEWAY_URL
ARG NEXT_PUBLIC_S3_ENDPOINT
ARG NEXT_PUBLIC_APP_NAME="POS System"
ARG NEXT_PUBLIC_APP_VERSION="1.0.0"
ARG NEXT_PUBLIC_ENVIRONMENT="development"
ARG NEXT_PUBLIC_ENABLE_ANALYTICS="true"
ARG NEXT_PUBLIC_ENABLE_DEBUG="false"
ARG NEXT_PUBLIC_DEFAULT_LOCALE="en"
ARG NEXT_PUBLIC_SUPPORTED_LOCALES="en,id"

# Set environment variables
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV API_GATEWAY_URL=$API_GATEWAY_URL
ENV NEXT_PUBLIC_S3_ENDPOINT=$NEXT_PUBLIC_S3_ENDPOINT
ENV NEXT_PUBLIC_APP_NAME=$NEXT_PUBLIC_APP_NAME
ENV NEXT_PUBLIC_APP_VERSION=$NEXT_PUBLIC_APP_VERSION
ENV NEXT_PUBLIC_ENVIRONMENT=$NEXT_PUBLIC_ENVIRONMENT
ENV NEXT_PUBLIC_ENABLE_ANALYTICS=$NEXT_PUBLIC_ENABLE_ANALYTICS
ENV NEXT_PUBLIC_ENABLE_DEBUG=$NEXT_PUBLIC_ENABLE_DEBUG
ENV NEXT_PUBLIC_DEFAULT_LOCALE=$NEXT_PUBLIC_DEFAULT_LOCALE
ENV NEXT_PUBLIC_SUPPORTED_LOCALES=$NEXT_PUBLIC_SUPPORTED_LOCALES

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Runtime stage
FROM node:22.20-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install production dependencies only
RUN npm ci --only=production

# Copy built application from builder
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/next.config.js ./

# Expose port
EXPOSE 3000

# Start the application
CMD ["npm", "start"]
