# Conf
Simple configuration package for Go applications.
It is easy to use and this package support only environment variables (for [12-Factor apps](https://12factor.net/#the_twelve_factors)).

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

	"github.com/virp/conf"
)

type RedisConfig struct {
	Addr     string `conf:"required"`
	Password string
	DB       int `conf:"default:0"`
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