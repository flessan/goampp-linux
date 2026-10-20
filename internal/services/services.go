//go:build windows

// Package services links every supported service into the binary. Each
// subpackage registers its download recipe, latest-version resolver and
// post-install steps with internal/packages from its init function.
package services

import (
	_ "goampp/internal/services/adminer"
	_ "goampp/internal/services/apache"
	_ "goampp/internal/services/composer"
	_ "goampp/internal/services/crystal"
	_ "goampp/internal/services/dart"
	_ "goampp/internal/services/elixir"
	_ "goampp/internal/services/erlang"
	_ "goampp/internal/services/golang"
	_ "goampp/internal/services/haskell"
	_ "goampp/internal/services/java"
	_ "goampp/internal/services/julia"
	_ "goampp/internal/services/kotlin"
	_ "goampp/internal/services/lua"
	_ "goampp/internal/services/mailpit"
	_ "goampp/internal/services/minio"
	_ "goampp/internal/services/mysql"
	_ "goampp/internal/services/nginx"
	_ "goampp/internal/services/nodejs"
	_ "goampp/internal/services/pgweb"
	_ "goampp/internal/services/php"
	_ "goampp/internal/services/phpmyadmin"
	_ "goampp/internal/services/postgresql"
	_ "goampp/internal/services/python"
	_ "goampp/internal/services/rabbitmq"
	_ "goampp/internal/services/redis"
	_ "goampp/internal/services/ruby"
	_ "goampp/internal/services/rust"
	_ "goampp/internal/services/scala"
	_ "goampp/internal/services/zig"
)
