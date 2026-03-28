package com.example.api.interfaces

import com.example.api.models.{ApiRequest, ApiResponse}

/** Service interface trait — implemented by the Scala Service module. */
trait UserService {
  def getUser(userId: String): ApiResponse
  def createUser(request: ApiRequest): ApiResponse
  def updateUser(userId: String, request: ApiRequest): ApiResponse
  def deleteUser(userId: String): ApiResponse
}

/** Order service interface. */
trait OrderService {
  def getOrder(orderId: String): ApiResponse
  def createOrder(request: ApiRequest): ApiResponse
  def listOrders(userId: String, page: Int, limit: Int): ApiResponse
}
