package com.example.service.impl

import com.example.api.interfaces.OrderService
import com.example.api.models.{ApiRequest, ApiResponse}

/** Concrete implementation of OrderService from the API Interface. */
class OrderServiceImpl extends OrderService {

  override def getOrder(orderId: String): ApiResponse =
    ApiResponse(
      success = true,
      data = Some(Map("id" -> orderId, "status" -> "active")),
    )

  override def createOrder(request: ApiRequest): ApiResponse =
    ApiResponse(
      success = true,
      data = Some(Map("action" -> "created")),
      requestId = request.requestId,
    )

  override def listOrders(userId: String, page: Int, limit: Int): ApiResponse =
    ApiResponse(
      success = true,
      data = Some(Map("user_id" -> userId, "page" -> page.toString, "limit" -> limit.toString)),
    )
}
