package service

import (
	"net/http"
	"time"

	"github.com/ProtocolONE/auth1.protocol.one/pkg/appcore/log"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var deviceIdCookie = "qdid"

// DeviceID middleware for generate and retrive unique per client device_id
func DeviceID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			c, err := ctx.Cookie(deviceIdCookie)
			if err != nil {
				if err != http.ErrNoCookie {
					return err // some internal error
				}
				c = &http.Cookie{
					HttpOnly: true,
					Secure:   true,
					Name:     deviceIdCookie,
					Value:    uuid.New().String(),
					MaxAge:   5 * 365 * 24 * int(time.Hour.Seconds()),
					Path:     "/",
				}

				ctx.SetCookie(c)
			}

			ctx.Set(deviceIdCookie, c.Value)
			return next(ctx)
		}
	}
}

func GetDeviceID(ctx echo.Context) string {
	value, ok := ctx.Get(deviceIdCookie).(string)
	if !ok {
		log.Error(ctx.Request().Context(), "device_id not found in request context, maybe you forgot add DeviceID middleware?")
	}
	return value
}
