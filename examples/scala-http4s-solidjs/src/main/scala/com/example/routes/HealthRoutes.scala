package com.example.routes

import cats.effect.IO
import org.http4s.HttpRoutes
import org.http4s.dsl.io.*

object HealthRoutes:
  val routes: HttpRoutes[IO] = HttpRoutes.of[IO]:
    case GET -> Root =>
      Ok("""{"status":"ok"}""")
