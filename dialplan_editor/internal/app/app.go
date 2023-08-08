package app

import (
	"context"
	"fmt"
	"myapp/config"
	"myapp/internal/handler"
	"myapp/internal/usecase"
	"myapp/pkg/jaegerotel"
	"myapp/pkg/ssh"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
)

func Run(cfg config.Config) {

	// Инициализация трейсера
	tp, err := jaegerotel.NewJaegerTracerProvider(
		viper.GetString("jaeger_host"),
		jaegerotel.WithConfig("diaplan_editor", viper.GetString("app_env")),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("не удалось инициализировать jaeger tracer provider")
	}

	otel.SetTracerProvider(tp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func(ctx context.Context) {
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("не удалось корректно остановить сервис jaeger tracer provider")
		}
	}(ctx)
	// END Инициализация трейсера

	// Старт трейсинга инийциализации сервисов
	tctx, span := jaegerotel.StartNewSpan("ServicesInitialization")
	// END Старт трейсинга инийциализации сервисов

	// SSH
	sshConn, err := ssh.New(tctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msgf("app - Run - SSH.New: %v", err)
	}
	defer sshConn.Close()

	span.End()

	// Use case
	asteriskUseCases := usecase.NewAsteriskUseCases(sshConn)

	// HTTP Server
	router := gin.Default()

	handler.NewRouter(router, *asteriskUseCases)

	router.Run(fmt.Sprintf("localhost:%d", cfg.HttpPort))

}
