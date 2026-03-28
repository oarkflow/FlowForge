val scala213Version = "2.13.14"

val http4sVersion = "0.23.30"
val doobieVersion = "1.0.0-RC6"

lazy val root = project
  .in(file("."))
  .enablePlugins(WarPlugin)
  .settings(
    name         := "scala-service",
    version      := "1.0.0",
    organization := "com.example",
    scalaVersion := scala213Version,
    libraryDependencies ++= Seq(
      // Web
      "javax.servlet"  % "javax.servlet-api" % "4.0.1"     % Provided,
      "org.scalatra"  %% "scalatra"          % "3.1.0",
      "org.scalatra"  %% "scalatra-json"     % "3.1.0",

      // JSON
      "io.circe"      %% "circe-core"        % "0.14.10",
      "io.circe"      %% "circe-generic"     % "0.14.10",
      "io.circe"      %% "circe-parser"      % "0.14.10",
      "org.json4s"    %% "json4s-jackson"    % "4.0.7",

      // Database
      "org.tpolecat"  %% "doobie-core"       % doobieVersion,
      "org.tpolecat"  %% "doobie-hikari"     % doobieVersion,
      "org.tpolecat"  %% "doobie-postgres"   % doobieVersion,
      "org.flywaydb"   % "flyway-core"       % "10.15.0",

      // Logging
      "ch.qos.logback" % "logback-classic"   % "1.5.16",

      // Testing
      "org.scalatest" %% "scalatest"         % "3.2.19"    % Test,
      "org.scalatra"  %% "scalatra-test"     % "3.1.0"     % Test,
    ),

    // WAR packaging
    // Unmanaged dependencies from lib/ directory include the API Interface JAR
  )
