# OTELZAP Logger

Библиотека логирования на основе Zap с поддержкой OTLP и контекстного обогащения.

## Установка
```bash
go get github.com/major1ink/otelzap
```

## Пример

```go
package main

import (
    "context"
    "github.com/major1ink/otelzap"
	"go.uber.org/zap"
	"time"
)

func main() {
    ctx := context.Background()
    l, err := logger.NewLogger(
        ctx,
        logger.WithAsJSON(true),
        logger.WithEnableOTLP(true),
        logger.WithOTLPEndpoint("localhost:4317"),
        logger.WithLevel("info"),
        logger.WithServiceName("my-service"),
        logger.WithServiceEnvironment("prod"),
        logger.WithEnableStdout(true),
        logger.WithShutdownTimeout(5*time.Second),
    )
    if err != nil {
        panic(err)
    }
    defer l.Close()

    ctx = context.WithValue(ctx, logger.traceIDKey, "12345")
    l.Info(ctx, "Привет, мир!", zap.String("key", "value"))
}
```

