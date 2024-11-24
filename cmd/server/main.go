package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-batteries/shortner/app/config"
	"github.com/go-batteries/shortner/app/db"
	"github.com/go-batteries/shortner/app/models"
	"github.com/go-batteries/shortner/app/seed"
	"github.com/go-batteries/shortner/cmd/server/controller"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
)

type EchoServer struct{}

func (app *EchoServer) StartHTTPServer(ctx context.Context, cfg *config.AppConfig) {
	seeder := seed.RegisterUrlSeeder()
	keyRanges := seeder.Shards(5)

	database := db.NewSqliteCoordinator(keyRanges)

	if err := database.ConnectShards(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to databases")
	}

	if _, err := database.ConnectCoordinatorDB(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to coordinator db")
	}

	shards, ok := database.GetShards()
	if !ok {
		log.Fatal().Msg("should not have failed to create shards")
	}

	database.SetPolicy(&db.RoundRobinPolicy[string]{Shards: shards})
	// statusRepo := models.NewShardStatusRepo(statusDB, config.MustParseSeedSize(cfg.SeedSize, "1M"))

	urlRepo := models.NewURLRepo(database)
	ctrl := controller.NewURLShortnerCtrl(urlRepo)

	port := cfg.AppPort

	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{fmt.Sprintf("http://localhost:%s", port)},
		AllowCredentials: true,
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderContentDisposition,
			echo.HeaderConnection,
			echo.HeaderCacheControl,
			// Access Token Headers,
		},
		ExposeHeaders: []string{
			echo.HeaderContentLength,
			echo.HeaderContentDisposition,
			echo.HeaderContentEncoding,
			echo.HeaderContentType,
			echo.HeaderCacheControl,
			echo.HeaderConnection,
			// Access Token Headers,
		},
	}))
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: e,
	}

	// hc := &HellowController{}
	// e.GET("/hellow", hc.Get)
	// e.DELETE("/hellow", hc.Delete)

	e.GET("/:shortKey", ctrl.Get)

	go func() {
		log.Info().Str("port", port).Msg("server started at")

		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	appCtx := context.Background()
	ctx, stop := signal.NotifyContext(appCtx, os.Interrupt)
	defer stop()

	<-ctx.Done()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("error during server shutdown")
	}
}

func main() {
	srvr := &EchoServer{}
	ctx := context.Background()

	srvr.StartHTTPServer(ctx, &config.AppConfig{
		AppPort:  "9091",
		SeedSize: "1M",
	})
}
