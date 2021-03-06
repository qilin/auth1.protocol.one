package api

import (
	"net/http"

	"github.com/ProtocolONE/auth1.protocol.one/pkg/api/apierror"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/database"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/helper"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/manager"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/models"
	"github.com/labstack/echo/v4"
)

func InitOauth2(cfg *Server) error {
	middleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			db := c.Get("database").(database.MgoSession)
			c.Set("oauth_manager", manager.NewOauthManager(db, cfg.Registry, cfg.SessionConfig, cfg.HydraConfig, cfg.ServerConfig, cfg.Recaptcha))

			return next(c)
		}
	}
	g := cfg.Echo.Group("/oauth2", middleware)

	g.GET("/consent", oauthConsent)
	g.POST("/consent", oauthConsentSubmit)
	g.POST("/introspect", oauthIntrospect)
	g.GET("/callback", oauthCallback)

	cfg.Echo.POST("/api/signup", oauthSignUp, middleware)
	cfg.Echo.POST("/api/checkUsername", oauthCheckUsername, middleware)

	return nil
}

func oauthConsent(ctx echo.Context) error {
	form := new(models.Oauth2ConsentForm)
	m := ctx.Get("oauth_manager").(*manager.OauthManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
		}
		ctx.Error(err)
		return ctx.HTML(http.StatusBadRequest, e.Message)
	}

	scopes, err := m.Consent(ctx, form)
	if err != nil {
		ctx.Error(err.Err)
		return ctx.HTML(http.StatusBadRequest, err.Message)
	}

	if len(scopes) == 0 || m.HasOnlyDefaultScopes(scopes) {
		url, err := m.ConsentSubmit(ctx, &models.Oauth2ConsentSubmitForm{
			Challenge: form.Challenge,
			Scope:     scopes,
		})

		if err != nil {
			ctx.Error(err.Err)
			return ctx.HTML(http.StatusBadRequest, err.Message)
		}

		return ctx.Redirect(http.StatusFound, url)
	}

	return ctx.Render(http.StatusOK, "oauth_consent.html", map[string]interface{}{
		"AuthWebFormSdkUrl": m.ApiCfg.AuthWebFormSdkUrl,
		"Challenge":         form.Challenge,
		"Scopes":            scopes,
	})
}

func oauthConsentSubmit(ctx echo.Context) error {
	form := new(models.Oauth2ConsentSubmitForm)
	m := ctx.Get("oauth_manager").(*manager.OauthManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
		}
		ctx.Error(err)
		return ctx.HTML(http.StatusBadRequest, e.Message)
	}

	url, err := m.ConsentSubmit(ctx, form)
	if err != nil {
		return ctx.Render(http.StatusOK, "oauth_consent.html", map[string]interface{}{
			"AuthWebFormSdkUrl": m.ApiCfg.AuthWebFormSdkUrl,
			"Challenge":         form.Challenge,
			"Scope":             m.GetScopes(form.Scope),
			"Error":             err.Error(),
		})
	}

	return ctx.Redirect(http.StatusPermanentRedirect, url)
}

func oauthIntrospect(ctx echo.Context) error {
	form := new(models.Oauth2IntrospectForm)
	m := ctx.Get("oauth_manager").(*manager.OauthManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	token, err := m.Introspect(ctx, form)
	if err != nil {
		ctx.Error(err.Err)
		return helper.JsonError(ctx, err)
	}

	return ctx.JSON(http.StatusOK, token)
}

func oauthSignUp(ctx echo.Context) error {
	form := new(models.Oauth2SignUpForm)
	m := ctx.Get("oauth_manager").(*manager.OauthManager)

	if err := ctx.Bind(form); err != nil {
		return apierror.InvalidRequest(err)
	}

	url, err := m.SignUp(ctx, form)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"url": url})
}

func oauthCheckUsername(ctx echo.Context) error {
	var r struct {
		Challenge string `json:"challenge"`
		Username  string `json:"username"`
	}

	if err := ctx.Bind(&r); err != nil {
		return apierror.InvalidRequest(err)
	}

	m := ctx.Get("oauth_manager").(*manager.OauthManager)

	ok, err := m.IsUsernameFree(ctx, r.Challenge, r.Username)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"available": ok,
	})
}

func oauthCallback(ctx echo.Context) error {
	form := new(models.Oauth2CallBackForm)
	m := ctx.Get("oauth_manager").(*manager.OauthManager)

	if err := ctx.Bind(form); err != nil {
		ctx.Error(err)
		return ctx.HTML(http.StatusBadRequest, models.ErrorInvalidRequestParameters)
	}

	code := http.StatusOK
	response, err := m.CallBack(ctx, form)
	if err != nil {
		ctx.Error(err.Err)
		code = http.StatusBadRequest
	}
	return ctx.Render(code, "oauth_callback.html", map[string]interface{}{
		"AuthWebFormSdkUrl": m.ApiCfg.AuthWebFormSdkUrl,
		"Success":           response.Success,
		"ErrorMessage":      response.ErrorMessage,
		"AccessToken":       response.AccessToken,
		"ExpiresIn":         response.ExpiresIn,
		"IdToken":           response.IdToken,
	})
}
