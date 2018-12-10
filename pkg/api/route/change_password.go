package route

import (
	"auth-one-api/pkg/api/manager"
	"auth-one-api/pkg/api/models"
	"auth-one-api/pkg/helper"
	"fmt"
	"github.com/labstack/echo"
	"net/http"
)

type ChangePassword struct {
	Manager manager.ChangePasswordManager
}

func ChangePasswordInit(cfg Config) error {
	route := &ChangePassword{
		Manager: manager.InitChangePasswordManager(cfg.Logger),
	}

	cfg.Http.POST("/dbconnections/change_password", route.ChangePasswordStart)
	cfg.Http.POST("/dbconnections/change_password/verify", route.ChangePasswordVerify)

	return nil
}

func (l *ChangePassword) ChangePasswordStart(ctx echo.Context) error {
	form := new(models.ChangePasswordStartForm)

	if err := ctx.Bind(form); err != nil {
		return helper.NewErrorResponse(
			ctx,
			BadRequiredHttpCode,
			BadRequiredCodeCommon,
			`Invalid request parameters`,
		)
	}

	if err := ctx.Validate(form); err != nil {
		return helper.NewErrorResponse(
			ctx,
			BadRequiredHttpCode,
			fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			`This is required field`,
		)
	}

	token, e := l.Manager.ChangePasswordStart(form)
	if e != nil {
		return helper.NewErrorResponse(ctx, BadRequiredHttpCode, e.GetCode(), e.GetMessage())
	}

	return ctx.JSON(http.StatusOK, token)
}

func (l *ChangePassword) ChangePasswordVerify(ctx echo.Context) error {
	form := new(models.ChangePasswordVerifyForm)

	if err := ctx.Bind(form); err != nil {
		return helper.NewErrorResponse(
			ctx,
			BadRequiredHttpCode,
			BadRequiredCodeCommon,
			`Invalid request parameters`,
		)
	}

	if err := ctx.Validate(form); err != nil {
		return helper.NewErrorResponse(
			ctx,
			BadRequiredHttpCode,
			fmt.Sprintf(BadRequiredCodeField, helper.GetSingleError(err).Field()),
			`This is required field`,
		)
	}

	token, e := l.Manager.ChangePasswordVerify(form)
	if e != nil {
		return helper.NewErrorResponse(ctx, BadRequiredHttpCode, e.GetCode(), e.GetMessage())
	}

	return ctx.JSON(http.StatusOK, token)
}
