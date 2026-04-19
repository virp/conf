# Conf
Simple configuration package for Go applications.
It is easy to use and supports only environment variables (for [12-Factor apps](https://12factor.net/#the_twelve_factors)).

## Install
```bash
go get github.com/virp/conf
```

## Usage Example
```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/virp/conf"
)

type RedisConfig struct {
	Addr      string `conf:"required"`
	Password  string
	DB        int           `conf:"default:0"`
	Endpoints []string      `conf:"default:localhost:6379;localhost:6380"`
	Timeout   time.Duration `conf:"default:5s"`
}

type AWS struct {
	Key    string `conf:"required,env:AWS_ACCESS_KEY_ID"`
	Secret string `conf:"required,env:AWS_SECRET_ACCESS_KEY"`
	Region string `conf:"default:eu-central-1,env:AWS_DEFAULT_REGION"`
}

type Config struct {
	Debug bool
	Redis RedisConfig
	AWS   AWS
	URLs  map[string]string `conf:"default:api:https://api.example.com;admin:https://admin.example.com"`
}

func main() {
	var cfg Config
	if err := conf.Parse("my_service", &cfg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Debug: %t\n", cfg.Debug) // MY_SERVICE_DEBUG
	fmt.Printf(
		"Redis:\n\tAddr: %s\n\tPassword: %s\n\tDB: %d\n",
		cfg.Redis.Addr,     // MY_SERVICE_REDIS_ADDR
		cfg.Redis.Password, // MY_SERVICE_REDIS_PASSWORD
		cfg.Redis.DB,       // MY_SERVICE_REDIS_DB
	)
	fmt.Printf(
		"AWS\n\tKey: %s\n\tSecret: %s\n\tRegion: %s\n",
		cfg.AWS.Key,    // AWS_ACCESS_KEY_ID or MY_SERVICE_AWS_ACCESS_KEY_ID
		cfg.AWS.Secret, // AWS_SECRET_ACCESS_KEY or MY_SERVICE_AWS_SECRET_ACCESS_KEY
		cfg.AWS.Region, // AWS_DEFAULT_REGION or MY_SERVICE_AWS_DEFAULT_REGION
	)
}

```

## Environment Names

Field names are converted from CamelCase to upper snake case and prefixed:

```go
type Config struct {
	Debug    bool   // MY_SERVICE_DEBUG
	HTTPPort int   // MY_SERVICE_HTTP_PORT
	RedisURL string // MY_SERVICE_REDIS_URL
}

_ = conf.Parse("my_service", &cfg)
```

Nested structs append their field name to the prefix:

```go
type Config struct {
	Redis RedisConfig
}

type RedisConfig struct {
	Addr string // MY_SERVICE_REDIS_ADDR
}
```

An empty prefix uses only the field name, for example `DEBUG`.

## Tags

Supported `conf` tag options:

- `required`: fail if no env variable is set.
- `default:value`: set a default before reading env variables.
- `env:NAME`: read `NAME` first, then fall back to the generated env name.
- `-`: skip the field.

Unknown tag options are treated as errors.

```go
type Config struct {
	Addr   string `conf:"required"`
	Port   int    `conf:"default:8080"`
	Region string `conf:"default:eu-central-1,env:AWS_DEFAULT_REGION"`
	Secret string `conf:"-"`
}
```

`required` and `default` cannot be used together.

## Custom Lookup

`Parse` reads values from `os.LookupEnv`. Use `ParseWithLookup` when tests or tools need a custom env-like source:

```go
env := map[string]string{
	"MY_SERVICE_DEBUG": "true",
	"MY_SERVICE_REDIS_ADDR": "localhost:6379",
}

lookup := func(key string) (string, bool) {
	value, ok := env[key]
	return value, ok
}

var cfg Config
if err := conf.ParseWithLookup("my_service", &cfg, lookup); err != nil {
	log.Fatal(err)
}
```

## Supported Types

Built-in parsing supports:

- `string`
- signed and unsigned integers
- `bool`
- `float32`, `float64`
- `time.Duration`
- slices, separated by `;`
- maps, formatted as `key:value;key:value`
- pointers to supported types and nested structs

Map values may contain `:`:

```go
type Config struct {
	Routes map[string]string `conf:"default:api:https://api.example.com:443;admin:http://localhost:8080"`
}
```

## Custom Types

Custom types can implement `conf.Setter`, `encoding.TextUnmarshaler`, or `encoding.BinaryUnmarshaler`.

```go
type Token string

func (t *Token) Set(value string) error {
	*t = Token("Bearer " + value)
	return nil
}

type Config struct {
	APIToken Token `conf:"required,env:API_TOKEN"`
}
```
