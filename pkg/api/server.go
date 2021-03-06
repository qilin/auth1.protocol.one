package api

import (
	"context"
	"html/template"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ProtocolONE/auth1.protocol.one/internal/domain/repository"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/api/apierror"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/appcore"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/captcha"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/config"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/database"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/models"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/service"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/webhooks"

	geoproto "github.com/ProtocolONE/geoip-service/pkg/proto"
	"github.com/ProtocolONE/mfa-service/pkg/proto"
	"github.com/boj/redistore"
	"github.com/go-redis/redis"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ory/hydra-client-go/client/admin"
	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

// ServerConfig contains common configuration parameters for start application server
type ServerConfig struct {
	// ApiConfig is common http setting for the application like a port, timeouts & etc.
	ApiConfig *config.Server

	// HydraConfig contains settings for the public and admin url of the Hydra application.
	HydraConfig *config.Hydra

	// HydraAdminApi is client of the Hydra for administration requests.
	HydraAdminApi admin.ClientService

	// SessionConfig contains settings for the session.
	SessionConfig *config.Session

	GeoService geoproto.GeoIpService

	// MfaService describes the interface for working with MFA micro-service.
	MfaService proto.MfaService

	// MgoSession describes the interface for working with Mongo session.
	MgoSession database.MgoSession

	// SessionStore is client for session storage.
	SessionStore *redistore.RediStore

	// RedisClient is Redis client.
	RedisClient *redis.Client

	// Mailer contains settings for the postman service
	Mailer *config.Mailer

	// Recaptcha contains settings for recaptcha integration
	Recaptcha *config.Recaptcha

	// MailTemplates contains settings for email templates
	MailTemplates *config.MailTemplates

	// Centrifugo contains centrifugo settings
	Centrifugo *config.Centrifugo
}

// Server is the instance of the application
type Server struct {
	// Echo is instance of the Echo framework
	Echo *echo.Echo

	// ApiConfig is common http setting for the application like a port, timeouts & etc.
	ServerConfig *config.Server

	// RedisClient is Redis client.
	RedisHandler *redis.Client

	// HydraConfig contains settings for the public and admin url of the Hydra application.
	HydraConfig *config.Hydra

	// SessionConfig contains settings for the session.
	SessionConfig *config.Session

	// Registry is the Registry service
	Registry service.InternalRegistry

	// Recaptcha is recaptcha integration
	Recaptcha *captcha.Recaptcha

	// WebHooks is the web-hooks service
	WebHooks *webhooks.WebHooks

	// MailTemplates
	MailTemplates *config.MailTemplates

	// Centrifugo
	Centrifugo *config.Centrifugo
}

// Template is used to display HTML pages.
type Template struct {
	templates *template.Template
}

// NewServer creates new instance of the application.
func NewServer(
	c *ServerConfig,
	spaces repository.SpaceRepository,
) (*Server, error) {
	registryConfig := &service.RegistryConfig{
		MgoSession:        c.MgoSession,
		HydraAdminApi:     c.HydraAdminApi,
		MfaService:        c.MfaService,
		RedisClient:       c.RedisClient,
		Mailer:            service.NewMailer(c.Mailer),
		GeoIpService:      c.GeoService,
		CentrifugoService: service.NewCentrifugoService(c.Centrifugo),
		Spaces:            spaces,
	}
	server := &Server{
		Echo:          echo.New(),
		RedisHandler:  c.RedisClient,
		ServerConfig:  c.ApiConfig,
		SessionConfig: c.SessionConfig,
		HydraConfig:   c.HydraConfig,
		Registry:      service.NewRegistryBase(registryConfig),
		Recaptcha:     captcha.NewRecaptcha(c.Recaptcha.Key, c.Recaptcha.Secret, c.Recaptcha.Hostname),
		WebHooks:      webhooks.NewWebhooks(),
		MailTemplates: c.MailTemplates,
		Centrifugo:    c.Centrifugo,
	}

	t := &Template{
		templates: template.Must(template.ParseGlob("public/templates/*.html")),
	}
	s := server.Echo
	s.HideBanner = true
	s.Renderer = t

	// postprocessing middleware
	s.Use(RequestLogger(skip("/health")))
	s.Use(apierror.Middleware())

	// preprocessing middleware
	s.Use(middleware.RequestID())
	s.Use(service.DeviceID())
	s.Use(contextMiddleware())

	// TODO: Validate origins for each application by settings
	s.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowHeaders:     []string{"authorization", "content-type"},
		AllowOrigins:     c.ApiConfig.AllowOrigins,
		AllowCredentials: c.ApiConfig.AllowCredentials,
		AllowMethods:     []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete, http.MethodOptions},
	}))
	s.Use(CSRFWithConfig(CSRFConfig{
		TokenLookup: "header:X-XSRF-TOKEN",
		CookieName:  "_csrf",
		Skipper:     csrfSkipper,
		CookiePath:  "/",
	}))
	s.Use(session.Middleware(c.SessionStore))
	s.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			db := c.MgoSession.Copy()
			defer db.Close()

			ctx.Set("database", db)

			return next(ctx)
		}
	})

	registerCustomValidator(s)

	if err := server.setupRoutes(); err != nil {
		zap.L().Fatal("Setup routes failed", zap.Error(err))
	}

	return server, nil
}

func registerCustomValidator(e *echo.Echo) {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		if name == "-" {
			return ""
		}

		return name
	})
	e.Validator = &models.CustomValidator{
		Validator: v,
	}
}

func (s *Server) Start(shutdown chan os.Signal) error {
	go func() {
		err := s.Echo.Start(":" + strconv.Itoa(s.ServerConfig.Port))
		if err != nil {
			zap.L().Fatal("Failed to start server", zap.Error(err))
		}
	}()

	select {
	// wait on kill signal
	case <-shutdown:
		zap.L().Fatal("Server is shutting down")
	}

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.Echo.Shutdown(ctx)
}

func (s *Server) setupRoutes() error {
	routes := []func(c *Server) error{
		InitHealth,
		InitManage,
		InitOauth2,
		InitCaptcha,
		InitPasswordReset,
		InitSocial,
		InitCentrifugo,
		InitLogin,
		InitPasswordLess,
		InitMFA,
	}

	for _, r := range routes {
		if err := r(s); err != nil {
			return err
		}
	}

	return nil
}

func contextMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			rid := c.Response().Header().Get(echo.HeaderXRequestID)
			did := service.GetDeviceID(c)
			c.SetRequest(c.Request().WithContext(appcore.WithRequest(ctx, rid, did)))
			return next(c)
		}
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func skip(urls ...string) middleware.Skipper {
	return func(c echo.Context) bool {
		for _, u := range urls {
			if strings.HasPrefix(c.Path(), u) {
				return true
			}
		}
		return false
	}
}

func csrfSkipper(ctx echo.Context) bool {
	if ctx.Request().Method == http.MethodGet {
		return false
	}
	// TODO allow for all POST apis
	return ctx.Path() != "/api/login" && ctx.Path() != "/oauth2/login" && ctx.Path() != "/oauth2/signup"
}
