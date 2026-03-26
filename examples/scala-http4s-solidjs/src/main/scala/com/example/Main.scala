package com.example

import cats.effect.*
import cats.syntax.semigroupk.*
import com.comcast.ip4s.*
import fs2.io.file.Path
import org.http4s.*
import org.http4s.dsl.io.*
import org.http4s.ember.server.EmberServerBuilder
import org.http4s.headers.`Content-Type`
import org.http4s.implicits.*
import org.http4s.server.Router
import org.http4s.StaticFile

import com.example.routes.{HealthRoutes, TodoRoutes}

object Main extends IOApp.Simple:

  // Directory where the built SolidJS frontend lives at runtime.
  // In the Docker container this is /app/public; for local dev you
  // can override via the STATIC_DIR env var.
  private val staticDir: String =
    Option(System.getenv("STATIC_DIR")).getOrElse("/app/public")

  /** Serve static files from the frontend build directory.
    * Falls back to index.html for SPA client-side routing. */
  private val staticRoutes: HttpRoutes[IO] = HttpRoutes.of[IO] {
    case req @ GET -> path if !path.toString.startsWith("/api") =>
      val filePath = staticDir + path.toString
      StaticFile
        .fromPath(Path(filePath), Some(req))
        .getOrElseF {
          // SPA fallback: serve index.html for any non-file route
          StaticFile
            .fromPath(Path(s"$staticDir/index.html"), Some(req))
            .getOrElseF(NotFound("Not found"))
        }
  }

  def run: IO[Unit] =
    val apiRoutes = Router(
      "/api/health" -> HealthRoutes.routes,
      "/api/todos"  -> TodoRoutes.routes,
    )

    // API routes take priority, then static file serving
    val httpApp = (apiRoutes <+> staticRoutes).orNotFound

    EmberServerBuilder
      .default[IO]
      .withHost(host"0.0.0.0")
      .withPort(port"8080")
      .withHttpApp(httpApp)
      .build
      .useForever
