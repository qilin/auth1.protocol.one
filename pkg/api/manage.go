package api

import (
	"fmt"
	"net/http"

	"github.com/ProtocolONE/auth1.protocol.one/pkg/api/apierror"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/database"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/helper"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/manager"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/models"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
)

func InitManage(cfg *Server) error {
	g := cfg.Echo.Group("/api/manage", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			db := c.Get("database").(database.MgoSession)
			c.Set("manage_manager", manager.NewManageManager(db, cfg.Registry))
			return next(c)
		}
	}, middleware.BasicAuth(func(u, p string, ctx echo.Context) (bool, error) {
		return u == "admin" && p == cfg.ServerConfig.ManageSecret, nil
	}))

	g.GET("/space/:id/identity", getIdentityProviders)
	g.POST("/space/:id/identity", addIdentityProvider)
	g.PUT("/space/:space_id/identity/:id", updateIdentityProvider)
	g.GET("/space/:space_id/identity/:id", getIdentityProvider)
	g.POST("/app", createApplication)
	g.PUT("/app/:id", updateApplication)
	g.GET("/app/:id", getApplication)
	g.POST("/app/:id/password", setPasswordSettings)
	g.GET("/app/:id/password", getPasswordSettings)
	g.GET("/identity/templates", getIdentityProviderTemplates)
	g.POST("/app/:id/ott", setOneTimeTokenSettings)
	g.POST("/mfa", addMFA)
	g.GET("/authlog", authlog)

	return nil
}

// Manage
func authlog(ctx echo.Context) error {
	var req struct {
		UserID   string `query:"user_id"`
		DeviceID string `query:"device_id"`
		From     string `query:"from"`
		Count    int    `query:"count"`
	}
	req.Count = 100 // default

	if err := ctx.Bind(&req); err != nil {
		return apierror.InvalidRequest(err)
	}

	if err := ctx.Validate(req); err != nil {
		return apierror.InvalidParameters(err)
	}

	// limit max records
	if req.Count > 10000 {
		req.Count = 10000
	}

	db := ctx.Get("database").(database.MgoSession)
	s := service.NewAuthLogService(db, nil)
	var logs []*service.AuthorizeLog
	var err error
	if req.UserID != "" {
		logs, err = s.Get(req.UserID, req.Count, req.From)
	} else {
		logs, err = s.GetByDevice(req.DeviceID, req.Count, req.From)
	}
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, logs)
}

func createApplication(ctx echo.Context) error {
	applicationForm := &models.ApplicationForm{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(applicationForm); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(applicationForm); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	app, err := m.CreateApplication(ctx, applicationForm)
	if err != nil {
		ctx.Error(err.Err)
		return ctx.HTML(http.StatusBadRequest, "Unable to create the application")
	}

	return ctx.JSON(http.StatusOK, app)
}

func getApplication(ctx echo.Context) error {
	id := ctx.Param("id")

	m := ctx.Get("manage_manager").(*manager.ManageManager)

	a, err := m.GetApplication(ctx, id)
	if err != nil {
		ctx.Error(err.Err)
		return ctx.HTML(http.StatusBadRequest, "Application not exists")
	}

	return ctx.JSON(http.StatusOK, a)
}

func updateApplication(ctx echo.Context) error {
	id := ctx.Param("id")
	applicationForm := &models.ApplicationForm{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(applicationForm); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(applicationForm); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	app, err := m.UpdateApplication(ctx, id, applicationForm)
	if err != nil {
		ctx.Error(err.Err)
		return ctx.HTML(http.StatusBadRequest, "Unable to update the application")
	}

	return ctx.JSON(http.StatusOK, app)
}

func addMFA(ctx echo.Context) error {
	mfaApplicationForm := &models.MfaApplicationForm{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(mfaApplicationForm); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(mfaApplicationForm); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	app, err := m.AddMFA(ctx, mfaApplicationForm)
	if err != nil {
		ctx.Error(err.Err)
		return ctx.HTML(http.StatusBadRequest, "Unable to create the application")
	}

	return ctx.JSON(http.StatusOK, app)
}

func setPasswordSettings(ctx echo.Context) error {
	id := ctx.Param("id")
	form := &models.PasswordSettings{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(form); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := m.SetPasswordSettings(ctx, id, form); err != nil {
		ctx.Error(err.Err)
		return ctx.HTML(http.StatusBadRequest, "Unable to set password settings for the application")
	}

	return ctx.HTML(http.StatusOK, "")
}

func getPasswordSettings(ctx echo.Context) error {
	id := ctx.Param("id")

	m := ctx.Get("manage_manager").(*manager.ManageManager)
	ps, err := m.GetPasswordSettings(id)
	if err != nil {
		ctx.Error(err.Err)
		return ctx.HTML(http.StatusBadRequest, "Application not exists")
	}

	return ctx.JSON(http.StatusOK, ps)
}

func addIdentityProvider(ctx echo.Context) error {
	id := ctx.Param("id")
	form := &models.AppIdentityProvider{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(form); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := m.AddAppIdentityProvider(id, form); err != nil {
		return errors.Wrap(err, "unable to add the identity provider")
	}

	return ctx.JSON(http.StatusOK, form)
}

func getIdentityProvider(ctx echo.Context) error {
	spaceID := ctx.Param("space_id")
	id := ctx.Param("id")

	m := ctx.Get("manage_manager").(*manager.ManageManager)
	ip, err := m.GetIdentityProvider(ctx, spaceID, id)
	if err != nil {
		return errors.Wrap(err, "unable to get identity provider")
	}

	return ctx.JSON(http.StatusOK, ip)
}

func getIdentityProviders(ctx echo.Context) error {
	spaceID := ctx.Param("id")

	m := ctx.Get("manage_manager").(*manager.ManageManager)
	list, err := m.GetIdentityProviders(ctx, spaceID)
	if err != nil {
		return errors.Wrap(err, "unable to get identity providers")
	}

	return ctx.JSON(http.StatusOK, list)
}

func getIdentityProviderTemplates(ctx echo.Context) error {
	m := ctx.Get("manage_manager").(*manager.ManageManager)
	return ctx.JSON(http.StatusOK, m.GetIdentityProviderTemplates())
}

func updateIdentityProvider(ctx echo.Context) error {
	id := ctx.Param("id")
	spaceID := ctx.Param("space_id")
	form := &models.AppIdentityProvider{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(form); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := m.UpdateAppIdentityProvider(spaceID, id, form); err != nil {
		ctx.Error(err.Err)
		return ctx.HTML(http.StatusBadRequest, "Unable to update the identity provider to the application")
	}

	return ctx.JSON(http.StatusOK, form)
}

func setOneTimeTokenSettings(ctx echo.Context) error {
	id := ctx.Param("id")
	form := &models.OneTimeTokenSettings{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(form); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
		}
		ctx.Error(err)
		return helper.JsonError(ctx, e)
	}

	if err := m.SetOneTimeTokenSettings(ctx, id, form); err != nil {
		ctx.Error(err.Err)
		return ctx.HTML(http.StatusBadRequest, "Unable to set OneTimeToken settings for the application")
	}

	return ctx.HTML(http.StatusOK, "")
}
