package repository

import (
	"github.com/ProtocolONE/auth1.protocol.one/internal/repository"
	"github.com/ProtocolONE/auth1.protocol.one/internal/repository/application"
	"github.com/ProtocolONE/auth1.protocol.one/internal/repository/profile"
	"github.com/ProtocolONE/auth1.protocol.one/internal/repository/user"
	"github.com/ProtocolONE/auth1.protocol.one/internal/repository/user_identity"
	"go.uber.org/fx"
)

func New() fx.Option {
	return fx.Provide(
		profile.New,
		user.New,
		application.New,
		user_identity.New,
		repository.MakeSpaceRepo,
	)
}
