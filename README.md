# HuJSON - "Human JSON" ([JWCC](https://nigeltao.github.io/blog/2021/json-with-commas-comments.html))

The `github.com/tailscale/hujson` package implements
the [JWCC](https://nigeltao.github.io/blog/2021/json-with-commas-comments.html) extension
of [standard JSON](https://datatracker.ietf.org/doc/html/rfc8259).
This package is a fork of the Go standard library's `encoding/json`.

The `JWCC` format permits two things over standard JSON:

1. C-style line comments and block comments intermixed with whitespace,
2. allows trailing commas after the last member/element in an object/array.

All JSON is valid JWCC.

For details, see the JWCC docs at:

https://nigeltao.github.io/blog/2021/json-with-commas-comments.html



