package com.example.routes

import cats.effect.IO
import cats.effect.unsafe.implicits.global
import io.circe.syntax.*
import org.http4s.HttpRoutes
import org.http4s.circe.*
import org.http4s.dsl.io.*

import com.example.models.{CreateTodo, Todo}

import java.util.concurrent.atomic.{AtomicLong, AtomicReference}
import scala.collection.immutable.Vector

object TodoRoutes:
  private val idGen = AtomicLong(0)
  private val store = AtomicReference(Vector.empty[Todo])

  val routes: HttpRoutes[IO] = HttpRoutes.of[IO]:
    case GET -> Root =>
      Ok(store.get().asJson)

    case req @ POST -> Root =>
      for
        body   <- req.as[CreateTodo](IO.asyncForIO, jsonOf[IO, CreateTodo])
        todo    = Todo(idGen.incrementAndGet(), body.title, completed = false)
        _       = store.updateAndGet(_ :+ todo)
        resp   <- Created(todo.asJson)
      yield resp

    case PUT -> Root / LongVar(id) / "toggle" =>
      store.updateAndGet(_.map: t =>
        if t.id == id then t.copy(completed = !t.completed) else t
      )
      Ok(store.get().find(_.id == id).map(_.asJson).getOrElse("null".asJson))

    case DELETE -> Root / LongVar(id) =>
      store.updateAndGet(_.filterNot(_.id == id))
      Ok("""{"deleted":true}""")
