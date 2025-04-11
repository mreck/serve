# Serve

A utility for serving files via http.

## Config

| cli  | env        | description                  | default      |
| ---- | ---------- | ---------------------------- | ------------ |
| addr | SERVE_ADDR | The server address           | 0.0.0.0:8000 |
| json | SERVE_JSON | Format log messages as JSON  | false        |
| dirs | SERVE_DIRS | The dirs that will be served | . [^1]       |
| ui   | SERVE_UI   | Run with web UI              | false        |
| api  | SERVE_API  | Run with JSON API            | true         |

[^1]: Format: `<NAME>=<DIR>;<NAME>=<DIR>;...` with the name being optional.
