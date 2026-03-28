package com.example.service.db

import org.flywaydb.core.Flyway

/** Runs Flyway database migrations. Called during post-deploy. */
object DatabaseMigrator {

  def migrate(): Unit = {
    val dbHost     = sys.env.getOrElse("DB_HOST", "localhost")
    val dbName     = sys.env.getOrElse("DB_NAME", "service")
    val dbUser     = sys.env.getOrElse("DB_USER", "service")
    val dbPassword = sys.env.getOrElse("DB_PASSWORD", "secret")
    val jdbcUrl    = s"jdbc:postgresql://$dbHost:5432/$dbName"

    val flyway = Flyway.configure()
      .dataSource(jdbcUrl, dbUser, dbPassword)
      .locations("classpath:db/migration")
      .load()

    val result = flyway.migrate()
    println(s"Applied ${result.migrationsExecuted} migration(s). " +
            s"Current version: ${result.targetSchemaVersion}")
  }
}
