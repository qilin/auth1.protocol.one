package repository

import (
	"context"
	"time"

	"github.com/ProtocolONE/auth1.protocol.one/internal/domain/entity"
	"github.com/ProtocolONE/auth1.protocol.one/internal/domain/repository"
	"github.com/ProtocolONE/auth1.protocol.one/internal/env"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type spaceModel struct {
	ID                bson.ObjectId    `bson:"_id"`
	Name              string           `bson:"name"`
	Description       string           `bson:"description"`
	UniqueUsernames   bool             `bson:"unique_usernames"`
	RequiresCaptcha   bool             `bson:"requires_captcha"`
	PasswordSettings  passwordSettings `bson:"password_settings"`
	IdentityProviders []idProvider     `bson:"identity_providers"`
	Roles             []string         `bson:"roles" json:"roles"`
	DefaultRole       string           `bson:"default_role" json:"default_role"`
	CreatedAt         time.Time        `bson:"created_at"`
	UpdatedAt         time.Time        `bson:"updated_at"`
}

type passwordSettings struct {
	BcryptCost     int  `bson:"bcrypt_cost"`
	Min            int  `bson:"min"`
	Max            int  `bson:"max"`
	RequireNumber  bool `bson:"require_number"`
	RequireUpper   bool `bson:"require_upper"`
	RequireSpecial bool `bson:"require_special"`
	RequireLetter  bool `bson:"require_letter"`
	TokenLength    int  `bson:"token_length"`
	TokenTTL       int  `bson:"token_ttl"`
}

type idProvider struct {
	ID                  bson.ObjectId `bson:"_id"`
	DisplayName         string        `bson:"display_name"`
	Name                string        `bson:"name"`
	Type                string        `bson:"type"`
	ClientID            string        `bson:"client_id"`
	ClientSecret        string        `bson:"client_secret"`
	ClientScopes        []string      `bson:"client_scopes"`
	EndpointAuthURL     string        `bson:"endpoint_auth_url"`
	EndpointTokenURL    string        `bson:"endpoint_token_url"`
	EndpointUserInfoURL string        `bson:"endpoint_userinfo_url"`
}

func newSpaceModel(s *entity.Space) *spaceModel {
	providers := make([]idProvider, 0, len(s.IdentityProviders))
	for _, provider := range s.IdentityProviders {
		providers = append(providers, idProvider{
			ID:                  bson.ObjectIdHex(string(provider.ID)),
			DisplayName:         provider.DisplayName,
			Name:                provider.Name,
			Type:                string(provider.Type),
			ClientID:            provider.ClientID,
			ClientSecret:        provider.ClientSecret,
			ClientScopes:        provider.ClientScopes,
			EndpointAuthURL:     provider.EndpointAuthURL,
			EndpointTokenURL:    provider.EndpointTokenURL,
			EndpointUserInfoURL: provider.EndpointUserInfoURL,
		})
	}

	return &spaceModel{
		ID:                bson.ObjectIdHex(string(s.ID)),
		Name:              s.Name,
		Description:       s.Description,
		UniqueUsernames:   s.UniqueUsernames,
		RequiresCaptcha:   s.RequiresCaptcha,
		PasswordSettings:  passwordSettings(s.PasswordSettings),
		IdentityProviders: providers,
		Roles:             s.Roles,
		DefaultRole:       s.DefaultRole,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}

func (m *spaceModel) convert() *entity.Space {
	providers := make([]entity.IdentityProvider, 0, len(m.IdentityProviders))
	for _, provider := range m.IdentityProviders {
		providers = append(providers, entity.IdentityProvider{
			ID:                  entity.IdentityProviderID(provider.ID.Hex()),
			DisplayName:         provider.DisplayName,
			Name:                provider.Name,
			Type:                entity.IDProviderType(provider.Type),
			ClientID:            provider.ClientID,
			ClientSecret:        provider.ClientSecret,
			ClientScopes:        provider.ClientScopes,
			EndpointAuthURL:     provider.EndpointAuthURL,
			EndpointTokenURL:    provider.EndpointTokenURL,
			EndpointUserInfoURL: provider.EndpointUserInfoURL,
		})
	}

	return &entity.Space{
		ID:                entity.SpaceID(m.ID.Hex()),
		Name:              m.Name,
		Description:       m.Description,
		UniqueUsernames:   m.UniqueUsernames,
		RequiresCaptcha:   m.RequiresCaptcha,
		PasswordSettings:  entity.PasswordSettings(m.PasswordSettings),
		IdentityProviders: providers,
		Roles:             m.Roles,
		DefaultRole:       m.DefaultRole,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

type SpaceRepository struct {
	col *mgo.Collection
}

func MakeSpaceRepo(env *env.Env) repository.SpaceRepository {
	return NewSpaceRepository(env.Store.Mongo)
}

func NewSpaceRepository(env *env.Mongo) *SpaceRepository {
	return &SpaceRepository{
		col: env.DB.C("space"),
	}
}

func (r *SpaceRepository) Find(ctx context.Context) ([]*entity.Space, error) {
	var m []spaceModel
	if err := r.col.Find(nil).All(&m); err != nil {
		return nil, err
	}

	var result []*entity.Space
	for i := range m {
		result = append(result, m[i].convert())
	}

	return result, nil
}

func (r *SpaceRepository) FindByID(ctx context.Context, id entity.SpaceID) (*entity.Space, error) {
	var m spaceModel
	oid := bson.ObjectIdHex(string(id))
	if err := r.col.FindId(oid).One(&m); err != nil {
		return nil, err
	}
	return m.convert(), nil
}

func (r *SpaceRepository) FindForProvider(ctx context.Context, id entity.IdentityProviderID) (*entity.Space, error) {
	var m spaceModel
	oid := bson.ObjectIdHex(string(id))
	if err := r.col.Find(bson.M{"identity_providers._id": oid}).One(&m); err != nil {
		return nil, err
	}
	return m.convert(), nil
}

func (r *SpaceRepository) Create(ctx context.Context, space *entity.Space) error {
	if space.ID == "" {
		space.ID = entity.SpaceID(bson.NewObjectId().Hex())
	}
	for i := range space.IdentityProviders {
		if space.IdentityProviders[i].ID == "" {
			space.IdentityProviders[i].ID = entity.IdentityProviderID(bson.NewObjectId().Hex())
		}
	}

	m := newSpaceModel(space)
	if err := r.col.Insert(m); err != nil {
		return err
	}
	*space = *m.convert()
	return nil
}

func (r *SpaceRepository) Update(ctx context.Context, space *entity.Space) error {
	for i := range space.IdentityProviders {
		if space.IdentityProviders[i].ID == "" {
			space.IdentityProviders[i].ID = entity.IdentityProviderID(bson.NewObjectId().Hex())
		}
	}

	m := newSpaceModel(space)
	m.UpdatedAt = time.Now()
	if err := r.col.UpdateId(m.ID, m); err != nil {
		return err
	}
	*space = *m.convert()
	return nil
}
