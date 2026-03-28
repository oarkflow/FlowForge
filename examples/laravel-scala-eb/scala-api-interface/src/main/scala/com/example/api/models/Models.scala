package com.example.api.models

import io.circe.{Decoder, Encoder}
import io.circe.generic.semiauto._

/** Shared request/response models used by both the API Interface and the Service. */

case class ApiRequest(
  action: String,
  payload: Map[String, String] = Map.empty,
  requestId: Option[String] = None,
)

object ApiRequest {
  implicit val decoder: Decoder[ApiRequest] = deriveDecoder
  implicit val encoder: Encoder[ApiRequest] = deriveEncoder
}

case class ApiResponse(
  success: Boolean,
  data: Option[Map[String, String]] = None,
  error: Option[String] = None,
  requestId: Option[String] = None,
)

object ApiResponse {
  implicit val decoder: Decoder[ApiResponse] = deriveDecoder
  implicit val encoder: Encoder[ApiResponse] = deriveEncoder
}
