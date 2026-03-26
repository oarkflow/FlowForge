# ==============================================================================
# Scala + SolidJS — Multi-stage Docker Build
# Builds both the Scala backend and SolidJS frontend. The Scala http4s server
# serves the frontend static files directly — no nginx/supervisor needed.
# ==============================================================================

# ---------------------------------------------------------------------------
# Stage 1: Build Frontend
# ---------------------------------------------------------------------------
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --prefer-offline --no-audit
COPY frontend/ .
RUN npm run build

# ---------------------------------------------------------------------------
# Stage 2: Build Backend
# ---------------------------------------------------------------------------
FROM sbtscala/scala-sbt:eclipse-temurin-21.0.8_9_1.12.7_3.3.7 AS backend-builder

WORKDIR /app
COPY build.sbt .
COPY project/ project/
RUN sbt update

COPY src/ src/
RUN sbt assembly

# Capture the assembled JAR path (avoids hardcoding the Scala version)
RUN cp target/scala-*/scala-http4s-app-assembly-*.jar /app/app.jar

# ---------------------------------------------------------------------------
# Stage 3: Runtime — JRE only (Scala serves both API + frontend)
# ---------------------------------------------------------------------------
FROM eclipse-temurin:21-jre-jammy

WORKDIR /app

# Copy backend fat JAR
COPY --from=backend-builder /app/app.jar app.jar

# Copy frontend build output to /app/public — http4s serves it as static files
COPY --from=frontend-builder /app/frontend/dist /app/public

EXPOSE 8080

CMD ["java", "-jar", "app.jar"]
