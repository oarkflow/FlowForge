val scala213Version = "2.13.14"

lazy val root = project
  .in(file("."))
  .settings(
    name         := "api-interface",
    version      := "1.0.0",
    organization := "com.example",
    scalaVersion := scala213Version,
    libraryDependencies ++= Seq(
      "io.circe"      %% "circe-core"    % "0.14.10",
      "io.circe"      %% "circe-generic" % "0.14.10",
      "io.circe"      %% "circe-parser"  % "0.14.10",
      "org.scalatest" %% "scalatest"     % "3.2.19" % Test,
    ),
    // Publish settings for local use
    publishMavenStyle := true,
    publishTo := Some(Resolver.file("local-repo", file("target/local-repo"))),
  )
