package route

import (
	"github.com/ProtocolONE/auth1.protocol.one/pkg/config"
	"github.com/ProtocolONE/mfa-service/pkg/proto"
	"github.com/globalsign/mgo"
	"github.com/go-redis/redis"
	"github.com/labstack/echo/v4"
	"github.com/ory/hydra/sdk/go/hydra"
)

type Config struct {
	Echo          *echo.Echo
	MgoSession    *mgo.Session
	Redis         *redis.Client
	MfaService    proto.MfaService
	Hydra         *hydra.CodeGenSDK
	SessionConfig *config.Session
}
