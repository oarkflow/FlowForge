val scala3Version = "3.4.2"

val http4sVersion = "0.23.30"
val circeVersion  = "0.14.10"
val doobieVersion = "1.0.0-RC6"

lazy val root = project
  .in(file("."))
  .settings(
    name         := "scala-http4s-app",
    version      := "0.1.0-SNAPSHOT",
    scalaVersion := scala3Version,
    libraryDependencies ++= Seq(
      "org.http4s"    %% "http4s-ember-server" % http4sVersion,
      "org.http4s"    %% "http4s-ember-client" % http4sVersion,
      "org.http4s"    %% "http4s-circe"        % http4sVersion,
      "org.http4s"    %% "http4s-dsl"          % http4sVersion,
      "io.circe"      %% "circe-generic"       % circeVersion,
      "io.circe"      %% "circe-parser"        % circeVersion,
      "org.tpolecat"  %% "doobie-core"         % doobieVersion,
      "org.tpolecat"  %% "doobie-hikari"       % doobieVersion,
      "ch.qos.logback" % "logback-classic"     % "1.5.16",
      "org.scalatest" %% "scalatest"           % "3.2.19" % Test,
    ),
    assembly / assemblyMergeStrategy := {
      case PathList("META-INF", xs @ _*) =>
        xs match {
          case "MANIFEST.MF" :: Nil => MergeStrategy.discard
          case "services" :: _      => MergeStrategy.concat
          case _                    => MergeStrategy.discard
        }
      case "reference.conf" => MergeStrategy.concat
      case x if x.endsWith(".class") => MergeStrategy.first
      case x =>
        val oldStrategy = (assembly / assemblyMergeStrategy).value
        oldStrategy(x)
    },
  )
