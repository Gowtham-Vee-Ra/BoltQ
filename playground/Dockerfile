FROM node:20-alpine

WORKDIR /app

# Install dependencies
COPY package.json package-lock.json* ./
RUN npm ci

# Copy project files
COPY . .

# Set environment variables
ENV VITE_API_URL=http://localhost:8080
ENV VITE_GRAFANA_URL=http://localhost:3000
ENV VITE_PROMETHEUS_URL=http://localhost:9092

# Expose port
EXPOSE 5173

# Start development server with host set for Docker
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]