package com.example.service

import org.scalatra._
import org.json4s.{DefaultFormats, Formats}
import org.scalatra.json._

// Import interfaces from the API Interface JAR (in lib/)
import com.example.api.interfaces.{UserService, OrderService}
import com.example.api.models.{ApiRequest, ApiResponse}
import com.example.service.impl.{UserServiceImpl, OrderServiceImpl}

class ServiceServlet extends ScalatraServlet with JacksonJsonSupport {
  protected implicit lazy val jsonFormats: Formats = DefaultFormats

  private val userService: UserService   = new UserServiceImpl()
  private val orderService: OrderService = new OrderServiceImpl()

  before() {
    contentType = formats("json")
  }

  // Health check
  get("/health") {
    Map("status" -> "healthy", "service" -> "scala-service", "version" -> "1.0.0")
  }

  // User endpoints
  get("/api/users/:id") {
    userService.getUser(params("id"))
  }

  post("/api/users") {
    val req = parsedBody.extract[ApiRequest]
    userService.createUser(req)
  }

  // Order endpoints
  get("/api/orders/:id") {
    orderService.getOrder(params("id"))
  }

  post("/api/orders") {
    val req = parsedBody.extract[ApiRequest]
    orderService.createOrder(req)
  }

  // Admin: trigger database migration
  post("/admin/migrate") {
    db.DatabaseMigrator.migrate()
    Map("status" -> "success", "message" -> "Database migrations complete")
  }
}
