package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"

	"github.com/caitunai/dotconfig/api/base"
	"github.com/caitunai/dotconfig/api/route"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/logger"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Port string
	Mode string
}

func NewServer(port, mode string) *Server {
	return &Server{Port: port, Mode: mode}
}

func (s *Server) Start(ctx context.Context) {
	if s.Mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(requestid.New())
	r.Use(logger.SetLogger(logger.WithLogger(func(c *gin.Context, _ zerolog.Logger) zerolog.Logger {
		tml := log.Logger.With()
		traceID := c.GetHeader("x-trace-id")
		if traceID != "" {
			tml = tml.Str("traceID", traceID)
		}
		spanID := c.GetHeader("x-span-id")
		if spanID != "" {
			tml = tml.Str("spanID", spanID)
		}
		logID := c.GetHeader("x-log-id")
		if logID != "" {
			tml = tml.Str("logID", logID)
		}
		tml = tml.Str("requestID", requestid.Get(c))
		tmp := tml.Logger()
		c.Request = c.Request.WithContext(tmp.WithContext(c.Request.Context()))
		return tmp.With().Str("namespace", "ginRequest").Logger()
	})))
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"https://www.example.com",
			"https://example.com",
			viper.GetString("url"),
		},
		AllowMethods:     []string{"POST", "PUT", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "x-requested-with"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			return strings.Contains(u.Host, "localhost") || strings.Contains(u.Host, "127.0.0.1")
		},
		MaxAge: 12 * time.Hour,
	}))
	route.InitRoute(base.NewRouter(r))
	r.HTMLRender = base.NewRender()
	srv := &http.Server{
		Addr:    ":" + s.Port,
		Handler: r,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("listen error")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown:")
		return
	}

	log.Info().Msg("Server exiting")
}
