package api

import (
	"fmt"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/helper"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/manager"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/models"
	"github.com/globalsign/mgo"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/http"
)

func InitManage(cfg *Server) error {
	g := cfg.Echo.Group("/api", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			db := c.Get("database").(*mgo.Session)
			logger := c.Get("logger").(*zap.Logger)
			c.Set("manage_manager", manager.NewManageManager(db, logger, cfg.Registry))

			return next(c)
		}
	})

	g.POST("/space", createSpace)
	g.PUT("/space/:id", updateSpace)
	g.GET("/space/:id", getSpace)
	g.POST("/app", createApplication)
	g.PUT("/app/:id", updateApplication)
	g.GET("/app/:id", getApplication)
	g.POST("/api/app/:id/password", setPasswordSettings)
	g.GET("/api/app/:id/password", getPasswordSettings)
	g.POST("/api/app/:id/identity", addIdentityProvider)
	g.PUT("/api/app/:app_id/identity/:id", updateIdentityProvider)
	g.GET("/api/app/:app_id/identity/:id", getIdentityProvider)
	g.GET("/api/app/:id/identity", getIdentityProviders)
	g.GET("/api/identity/templates", getIdentityProviderTemplates)
	g.POST("/mfa", addMFA)

	return nil
}

func createSpace(ctx echo.Context) error {
	form := &models.SpaceForm{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
			Error:   errors.Wrap(err, "CreateSpace bind form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(form); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
			Error:   errors.Wrap(err, "CreateSpace validate form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	s, err := m.CreateSpace(ctx, form)
	if err != nil {
		return ctx.HTML(http.StatusBadRequest, "Unable to create the space")
	}

	return ctx.JSON(http.StatusOK, s)
}

func getSpace(ctx echo.Context) error {
	id := ctx.Param("id")
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	space, err := m.GetSpace(ctx, id)
	if err != nil {
		return ctx.HTML(http.StatusBadRequest, "Space not exists")
	}

	return ctx.JSON(http.StatusOK, space)
}

func updateSpace(ctx echo.Context) error {
	id := ctx.Param("id")
	form := &models.SpaceForm{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
			Error:   errors.Wrap(err, "UpdateSpace bind form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(form); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
			Error:   errors.Wrap(err, "UpdateSpace validate form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	space, err := m.UpdateSpace(ctx, id, form)
	if err != nil {
		return ctx.HTML(http.StatusBadRequest, "Unable to update the space")
	}

	return ctx.JSON(http.StatusOK, space)
}

func createApplication(ctx echo.Context) error {
	applicationForm := &models.ApplicationForm{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(applicationForm); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
			Error:   errors.Wrap(err, "CreateApplication bind form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(applicationForm); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
			Error:   errors.Wrap(err, "CreateApplication validate form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	app, err := m.CreateApplication(ctx, applicationForm)
	if err != nil {
		return ctx.HTML(http.StatusBadRequest, "Unable to create the application")
	}

	return ctx.JSON(http.StatusOK, app)
}

func getApplication(ctx echo.Context) error {
	id := ctx.Param("id")

	m := ctx.Get("manage_manager").(*manager.ManageManager)

	a, err := m.GetApplication(ctx, id)
	if err != nil {
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
			Error:   errors.Wrap(err, "UpdateApplication bind form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(applicationForm); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
			Error:   errors.Wrap(err, "UpdateApplication validate form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	app, err := m.UpdateApplication(ctx, id, applicationForm)
	if err != nil {
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
			Error:   errors.Wrap(err, "AddMFA bind form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(mfaApplicationForm); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
			Error:   errors.Wrap(err, "AddMFA validate form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	app, err := m.AddMFA(ctx, mfaApplicationForm)
	if err != nil {
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
			Error:   errors.Wrap(err, "PasswordSettings bind form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(form); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
			Error:   errors.Wrap(err, "PasswordSettings validate form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := m.SetPasswordSettings(ctx, id, form); err != nil {
		return ctx.HTML(http.StatusBadRequest, "Unable to set password settings for the application")
	}

	return ctx.HTML(http.StatusOK, "")
}

func getPasswordSettings(ctx echo.Context) error {
	id := ctx.Param("id")

	m := ctx.Get("manage_manager").(*manager.ManageManager)
	ps, err := m.GetPasswordSettings(id)
	if err != nil {
		return ctx.HTML(http.StatusBadRequest, "Application not exists")
	}

	return ctx.JSON(http.StatusOK, ps)
}

func addIdentityProvider(ctx echo.Context) error {
	form := &models.AppIdentityProvider{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
			Error:   errors.Wrap(err, "AppIdentityProvider bind form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(form); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
			Error:   errors.Wrap(err, "Add AppIdentityProvider validate form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := m.AddAppIdentityProvider(ctx, form); err != nil {
		return ctx.HTML(http.StatusBadRequest, "Unable to add the identity provider to the application")
	}

	return ctx.JSON(http.StatusOK, form)
}

func getIdentityProvider(ctx echo.Context) error {
	appID := ctx.Param("app_id")
	id := ctx.Param("id")

	m := ctx.Get("manage_manager").(*manager.ManageManager)
	ip, err := m.GetIdentityProvider(ctx, appID, id)
	if err != nil {
		return ctx.HTML(http.StatusBadRequest, "Identity provider not exists")
	}

	return ctx.JSON(http.StatusOK, ip)
}

func getIdentityProviders(ctx echo.Context) error {
	appID := ctx.Param("id")

	m := ctx.Get("manage_manager").(*manager.ManageManager)
	list, err := m.GetIdentityProviders(ctx, appID)
	if err != nil {
		return ctx.HTML(http.StatusBadRequest, "Unable to give identity providers")
	}

	return ctx.JSON(http.StatusOK, list)
}

func getIdentityProviderTemplates(ctx echo.Context) error {
	m := ctx.Get("manage_manager").(*manager.ManageManager)
	return ctx.JSON(http.StatusOK, m.GetIdentityProviderTemplates())
}

func updateIdentityProvider(ctx echo.Context) error {
	id := ctx.Param("id")
	form := &models.AppIdentityProvider{}
	m := ctx.Get("manage_manager").(*manager.ManageManager)

	if err := ctx.Bind(form); err != nil {
		e := &models.GeneralError{
			Code:    BadRequiredCodeCommon,
			Message: models.ErrorInvalidRequestParameters,
			Error:   errors.Wrap(err, "Update AppIdentityProvider bind form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := ctx.Validate(form); err != nil {
		e := &models.GeneralError{
			Code:    fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			Message: models.ErrorRequiredField,
			Error:   errors.Wrap(err, "Update AppIdentityProvider validate form failed"),
		}
		helper.SaveErrorLog(ctx, m.Logger, e)
		return helper.JsonError(ctx, e)
	}

	if err := m.UpdateAppIdentityProvider(ctx, id, form); err != nil {
		return ctx.HTML(http.StatusBadRequest, "Unable to update the identity provider to the application")
	}

	return ctx.JSON(http.StatusOK, form)
}
