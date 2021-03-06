package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ProtocolONE/auth1.protocol.one/pkg/api/apierror"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/captcha"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/config"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/database"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/manager"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/models"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/service"
	"github.com/globalsign/mgo"
	"github.com/labstack/echo/v4"
)

func InitSocial(cfg *Server) error {
	s := NewSocial(cfg)

	cfg.Echo.GET("/api/providers", s.List)
	cfg.Echo.GET("/api/providers/:name/profile", s.Profile)
	cfg.Echo.GET("/api/providers/:name/check", s.Check)
	cfg.Echo.GET("/api/providers/:name/confirm", s.Confirm)
	cfg.Echo.GET("/api/providers/:name/cancel", s.Cancel)
	cfg.Echo.POST("/api/providers/:name/link", s.Link)
	cfg.Echo.POST("/api/providers/:name/signup", s.Signup)
	// redirect based apis
	cfg.Echo.GET("/api/providers/:name/forward", s.Forward, apierror.Redirect("/error"))
	cfg.Echo.GET("/api/providers/:name/callback", s.Callback, apierror.Redirect("/error"))
	cfg.Echo.GET("/api/providers/:name/complete-auth", s.CompleteAuth, apierror.Redirect("/error"))

	return nil
}

type Social struct {
	registry service.InternalRegistry

	HydraConfig   *config.Hydra
	SessionConfig *config.Session
	ServerConfig  *config.Server
	Recaptcha     *captcha.Recaptcha
}

func NewSocial(cfg *Server) *Social {
	return &Social{
		registry:      cfg.Registry,
		HydraConfig:   cfg.HydraConfig,
		SessionConfig: cfg.SessionConfig,
		ServerConfig:  cfg.ServerConfig,
	}
}

type ProviderInfo struct {
	Name string `json:"name"`
	// Url  string `json:"url"`
}

func (s *Social) Signup(ctx echo.Context) error {
	form := new(models.Oauth2SignUpForm)
	var (
		db = ctx.Get("database").(database.MgoSession)
		m  = manager.NewOauthManager(db, s.registry, s.SessionConfig, s.HydraConfig, s.ServerConfig, s.Recaptcha)
	)

	if err := ctx.Bind(form); err != nil {
		return apierror.InvalidRequest(err)
	}

	url, err := m.SignUp(ctx, form)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"url": url})
}

func (s *Social) Link(ctx echo.Context) error {
	var (
		db = ctx.Get("database").(database.MgoSession)
		m  = manager.NewOauthManager(db, s.registry, s.SessionConfig, s.HydraConfig, s.ServerConfig, s.Recaptcha)
	)

	var form = new(models.Oauth2LoginSubmitForm)
	if err := ctx.Bind(form); err != nil {
		return apierror.InvalidRequest(err)
	}
	if err := ctx.Validate(form); err != nil {
		return apierror.InvalidParameters(err)
	}

	url, err := m.Auth(ctx, form)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"url": url})

}

func (s *Social) List(ctx echo.Context) error {
	var challenge = ctx.QueryParam("login_challenge")

	db := ctx.Get("database").(database.MgoSession)
	m := manager.NewLoginManager(db, s.registry)

	ips, err := m.Providers(challenge)
	if err != nil {
		return err
	}

	var res []ProviderInfo
	for i := range ips {
		res = append(res, ProviderInfo{
			Name: ips[i].Name,
			// Url:  "",
		})
	}

	return ctx.JSON(http.StatusOK, res)
}

func (s *Social) Forward(ctx echo.Context) error {
	var (
		name      = ctx.Param("name")
		challenge = ctx.QueryParam("login_challenge")
		launcher  = ctx.QueryParam("launcher")
		domain    = fmt.Sprintf("%s://%s", ctx.Scheme(), ctx.Request().Host)
	)

	db := ctx.Get("database").(database.MgoSession)
	m := manager.NewLoginManager(db, s.registry)

	url, err := m.ForwardUrl(challenge, name, domain, launcher)
	if err != nil {
		return err
	}

	// if launcher == true, then store challenge and options
	if launcher == "true" {
		err := s.registry.LauncherTokenService().Set(challenge, models.LauncherToken{
			Challenge: challenge,
			Name:      name,
			Status:    "in_progress",
		}, &models.LauncherTokenSettings{
			TTL: 600,
		})
		if err != nil {
			return err
		}
		err = s.registry.CentrifugoService().InProgress(challenge)
		if err != nil {
			return err
		}
	}

	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (s *Social) Callback(ctx echo.Context) error {
	var (
		name = ctx.Param("name")
		req  struct {
			Code  string `query:"code"`
			State string `query:"state"`
			Error string `query:"error"`
		}
		domain = fmt.Sprintf("%s://%s", ctx.Scheme(), ctx.Request().Host)
	)

	db := ctx.Get("database").(database.MgoSession)
	m := manager.NewLoginManager(db, s.registry)

	if err := ctx.Bind(&req); err != nil {
		return apierror.InvalidRequest(err)
	}

	if req.Error != "" {
		s, err := manager.DecodeState(req.State)
		if err != nil {
			return err
		}
		return ctx.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/sign-in?login_challenge=%s", s.Challenge))
	}

	// if launcher token with login_challenge key exists, then return to launcher
	state, err := manager.DecodeState(req.State)
	if err != nil {
		return err
	}

	ui, uis, err := m.GetUserIdentities(state.Challenge, name, domain, req.Code)
	if err != nil && err != mgo.ErrNotFound {
		return err
	}

	// For Launcher
	if state.Launcher == "true" {
		t := &models.LauncherToken{}
		err := s.registry.LauncherTokenService().Get(state.Challenge, t)
		if err != nil {
			return err
		}

		t.Domain = domain
		t.UserIdentity = ui
		t.UserIdentitySocial = uis

		err = s.registry.LauncherTokenService().Set(state.Challenge, t, &models.LauncherTokenSettings{TTL: 600})
		if err != nil {
			return err
		}
		return ctx.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/social-sign-in-confirm?login_challenge=%s&name=%s", state.Challenge, name))
	}

	// For Web
	//if ui != nil && err != mgo.ErrNotFound {
	//	// accept login and redirect
	//	url, err := m.Accept(ctx, ui, name, state.Challenge)
	//	if err != nil {
	//		return err
	//	}
	//	return ctx.Redirect(http.StatusTemporaryRedirect, url)
	//}
	//// UserIdentity does not exist: link or sign up
	//url, err := m.SocialLogin(uis, domain, name, state.Challenge)
	//if err != nil {
	//	return err
	//}
	url, err := s.accept(ctx, m, ui, uis, name, domain, state.Challenge)
	if err != nil {
		return err
	}
	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (s *Social) Profile(ctx echo.Context) error {
	var token = ctx.QueryParam("token")

	db := ctx.Get("database").(database.MgoSession)
	m := manager.NewLoginManager(db, s.registry)

	profile, err := m.Profile(token)
	if err != nil {
		return err
	}

	profile.HideSensitive()

	return ctx.JSON(http.StatusOK, profile)

}

func (s *Social) Check(ctx echo.Context) error {
	type response struct {
		Status string `json:"status"`
		URL    string `json:"url,omitempty"`
	}

	var (
		name           = ctx.Param("name")
		loginChallenge = ctx.QueryParam("login_challenge")
		t              = &models.LauncherToken{}
	)

	err := s.registry.LauncherTokenService().Get(loginChallenge, t)
	if err != nil {
		if err != models.LauncherToken_NotFound {
			ctx.Logger().Error(err.Error())
		}
		return ctx.JSON(http.StatusOK, response{
			Status: "expired",
		})
	}

	if t.Name != name {
		return ctx.JSON(http.StatusOK, response{
			Status: "expired",
		})
	}

	return ctx.JSON(http.StatusOK, response{
		Status: t.Status,
		URL:    t.URL,
	})
}

func (s *Social) Confirm(ctx echo.Context) error {
	var (
		challenge = ctx.QueryParam("login_challenge")
	)

	t := &models.LauncherToken{}
	err := s.registry.LauncherTokenService().Get(challenge, t)
	if err != nil {
		if err == models.LauncherToken_NotFound {
			return ctx.JSON(http.StatusOK, map[string]string{
				"status": "canceled",
			})
		}
		return err
	}

	if t.Status == models.LauncherAuth_Canceled {
		return ctx.JSON(http.StatusOK, map[string]string{
			"status": "canceled",
		})
	}

	db := ctx.Get("database").(database.MgoSession)
	m := manager.NewLoginManager(db, s.registry)

	// if UserIdentity found, launcher must complete auth process via follow url
	var url = t.Domain + "/api/providers/" + t.Name + "/complete-auth?login_challenge=" + t.Challenge
	if t.UserIdentity == nil {
		url, err = s.accept(ctx, m, t.UserIdentity, t.UserIdentitySocial, t.Name, t.Domain, t.Challenge)
		if err != nil {
			return err
		}
	}

	err = s.registry.CentrifugoService().Success(challenge, url)
	if err != nil {
		return err
	}

	t.Status = "success"
	t.URL = url
	err = s.registry.LauncherTokenService().Set(challenge, t, &models.LauncherTokenSettings{TTL: 600})
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]string{
		"status": "success",
	})
}

func (s *Social) Cancel(ctx echo.Context) error {
	var (
		challenge = ctx.QueryParam("login_challenge")
		url       = ""
	)

	t := &models.LauncherToken{}
	err := s.registry.LauncherTokenService().Get(challenge, t)
	if err != nil {
		if err == models.LauncherToken_NotFound {
			return ctx.JSON(http.StatusOK, map[string]string{
				"status": "expired",
			})
		}
		return err
	}

	t.Status = models.LauncherAuth_Canceled
	t.URL = url
	err = s.registry.LauncherTokenService().Set(challenge, t, &models.LauncherTokenSettings{TTL: 600})
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]string{
		"status": "success",
	})
}

func (s *Social) CompleteAuth(ctx echo.Context) error {
	var (
		challenge = ctx.QueryParam("login_challenge")
	)

	var t models.LauncherToken
	err := s.registry.LauncherTokenService().Get(challenge, &t)
	if err != nil {
		return err
	}

	if t.Status != models.LauncherAuth_Success {
		return errors.New("invalid token state: not successful")
	}

	if t.UserIdentity == nil {
		return errors.New("invalid token state: no user identity")
	}

	db := ctx.Get("database").(database.MgoSession)
	m := manager.NewLoginManager(db, s.registry)

	url, err := m.Accept(ctx, t.UserIdentity, t.Name, t.Challenge)
	if err != nil {
		return err
	}

	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (s *Social) accept(ctx echo.Context, m manager.LoginManagerInterface, ui *models.UserIdentity, uis *models.UserIdentitySocial, name, domain, challenge string) (string, error) {
	if ui != nil {
		// accept login and redirect
		return m.Accept(ctx, ui, name, challenge)
	}
	// UserIdentity does not exist: link or sign up
	return m.SocialLogin(uis, domain, name, challenge)
}
