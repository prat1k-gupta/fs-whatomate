# Development Environment with Hot Reloading

This directory contains configurations for running Whatomate in development mode with hot reloading.

## Quick Start

### Start Development Environment

```bash
# From the docker directory
docker compose -f docker-compose.dev.yml up

# Or run in background
docker compose -f docker-compose.dev.yml up -d
```

### Stop Development Environment

```bash
docker compose -f docker-compose.dev.yml down
```

### View Logs

```bash
# All services
docker compose -f docker-compose.dev.yml logs -f

# Specific service
docker compose -f docker-compose.dev.yml logs -f app
docker compose -f docker-compose.dev.yml logs -f frontend
```

## What's Different in Dev Mode?

### Backend (Go)
- ✅ **Hot Reloading**: Uses [Air](https://github.com/air-verse/air) to automatically rebuild when Go files change
- ✅ **Source Code Mounted**: Your local code is mounted into the container
- ✅ **Instant Feedback**: Changes reflect in 1-2 seconds

### Frontend (Vue)
- ✅ **Vite Dev Server**: Runs with Vite's built-in hot module replacement (HMR)
- ✅ **Fast Refresh**: Changes reflect instantly in the browser
- ✅ **Source Maps**: Full debugging support
- 🌐 **Available at**: http://localhost:5173

### Database & Redis
- Same as production setup
- Separate volumes (`postgres-data-dev`, `redis-data-dev`) to avoid conflicts

## Configuration

### Air Configuration (`.air.toml`)
Located in the project root. Modify this to:
- Change which files trigger rebuilds
- Adjust build delays
- Customize build commands

### Environment Variables
Create a `config.toml` in the project root with your settings. The example is at `config.example.toml`.

## Accessing Services

| Service  | URL/Port                |
|----------|-------------------------|
| Backend  | http://localhost:8080   |
| Frontend | http://localhost:5173   |
| Database | localhost:5433          |
| Redis    | localhost:6379          |

## Troubleshooting

### Changes not reflecting?

**Backend:**
```bash
# Check Air logs
docker compose -f docker-compose.dev.yml logs -f app

# Rebuild container
docker compose -f docker-compose.dev.yml up --build app
```

**Frontend:**
```bash
# Check Vite logs
docker compose -f docker-compose.dev.yml logs -f frontend

# Clear node_modules and reinstall
docker compose -f docker-compose.dev.yml down
docker volume rm docker_frontend_node_modules 2>/dev/null || true
docker compose -f docker-compose.dev.yml up --build frontend
```

### Port already in use?

```bash
# Check what's using the port
lsof -ti:8080
lsof -ti:5173

# Stop production containers
docker compose -f docker-compose.yml down
```

### Database connection issues?

```bash
# Check database health
docker compose -f docker-compose.dev.yml ps

# Reset database
docker compose -f docker-compose.dev.yml down -v
docker compose -f docker-compose.dev.yml up -d db
```

## Production Build

When you're ready to deploy, use the production compose file:

```bash
# Build production image
docker compose -f docker-compose.yml build

# Start production
docker compose -f docker-compose.yml up -d
```

## Tips

1. **First Run**: The first startup may take a few minutes as Go downloads dependencies and npm installs packages
2. **Fast Subsequent Starts**: After the first run, containers start much faster
3. **Clean Rebuild**: Use `docker compose -f docker-compose.dev.yml up --build` to force rebuild
4. **Clear Everything**: Use `docker compose -f docker-compose.dev.yml down -v` to remove volumes too
