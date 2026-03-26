package com.example.models

import io.circe.{Decoder, Encoder}
import io.circe.generic.semiauto.*

case class Todo(id: Long, title: String, completed: Boolean)

object Todo:
  given Encoder[Todo] = deriveEncoder[Todo]
  given Decoder[Todo] = deriveDecoder[Todo]

case class CreateTodo(title: String)

object CreateTodo:
  given Decoder[CreateTodo] = deriveDecoder[CreateTodo]
  given Encoder[CreateTodo] = deriveEncoder[CreateTodo]
