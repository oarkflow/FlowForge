package com.example

import org.scalatest.funsuite.AnyFunSuite
import com.example.models.{Todo, CreateTodo}

class TodoRoutesSpec extends AnyFunSuite:
  test("Todo model should have correct fields"):
    val todo = Todo(1L, "Buy milk", false)
    assert(todo.id == 1L)
    assert(todo.title == "Buy milk")
    assert(!todo.completed)

  test("CreateTodo should accept a title"):
    val ct = CreateTodo("Write tests")
    assert(ct.title == "Write tests")
