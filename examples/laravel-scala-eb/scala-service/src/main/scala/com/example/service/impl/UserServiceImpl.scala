package com.example.service.impl

import com.example.api.interfaces.UserService
import com.example.api.models.{ApiRequest, ApiResponse}

/** Concrete implementation of UserService from the API Interface. */
class UserServiceImpl extends UserService {

  override def getUser(userId: String): ApiResponse =
    ApiResponse(
      success = true,
      data = Some(Map("id" -> userId, "name" -> "Example User")),
    )

  override def createUser(request: ApiRequest): ApiResponse =
    ApiResponse(
      success = true,
      data = Some(Map("action" -> "created")),
      requestId = request.requestId,
    )

  override def updateUser(userId: String, request: ApiRequest): ApiResponse =
    ApiResponse(
      success = true,
      data = Some(Map("id" -> userId, "action" -> "updated")),
      requestId = request.requestId,
    )

  override def deleteUser(userId: String): ApiResponse =
    ApiResponse(success = true, data = Some(Map("id" -> userId, "action" -> "deleted")))
}
