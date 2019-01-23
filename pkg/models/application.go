package models

import (
	"auth-one-api/pkg/database"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/twitch"
	"golang.org/x/oauth2/vk"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

type (
	ApplicationService struct {
		db *mgo.Database
	}

	Application struct {
		ID          bson.ObjectId `bson:"_id" json:"id"`                        // unique application identifier
		SpaceId     bson.ObjectId `bson:"space_id" json:"space_id"`             // application space owner
		Name        string        `bson:"name" json:"name" validate:"required"` // application name
		Description string        `bson:"description" json:"description"`       // application description
		IsActive    bool          `bson:"is_active" json:"is_active"`           // is application active
		CreatedAt   time.Time     `bson:"created_at" json:"-"`                  // date of create application
		UpdatedAt   time.Time     `bson:"updated_at" json:"-"`                  // date of update application
	}

	ApplicationForm struct {
		SpaceId     bson.ObjectId       `json:"space_id"`                        // unique space identifier
		Application *ApplicationFormApp `json:"application" validate:"required"` // application data
	}

	ApplicationFormApp struct {
		Name        string `bson:"name" json:"name" validate:"required"` // application name
		Description string `bson:"description" json:"description"`       // application description
		IsActive    bool   `bson:"is_active" json:"is_active"`           // is application active
	}
)

func NewApplicationService(dbHandler *database.Handler) *ApplicationService {
	return &ApplicationService{dbHandler.Session.DB(dbHandler.Name)}
}

func (s ApplicationService) Create(app *Application) error {
	if err := s.db.C(database.TableApplication).Insert(app); err != nil {
		return err
	}

	return nil
}

func (s ApplicationService) Update(app *Application) error {
	if err := s.db.C(database.TableApplication).UpdateId(app.ID, app); err != nil {
		return err
	}

	return nil
}

func (s ApplicationService) Get(id bson.ObjectId) (*Application, error) {
	a := &Application{}
	if err := s.db.C(database.TableApplication).
		FindId(id).
		One(&a); err != nil {
		return nil, err
	}

	return a, nil
}

func (s ApplicationService) LoadPasswordSettings() (*PasswordSettings, error) {
	return &PasswordSettings{
		BcryptCost:        10,
		Min:               4,
		Max:               10,
		RequireNumber:     true,
		RequireSpecial:    true,
		RequireUpper:      true,
		ChangeTokenLength: 128,
		ChangeTokenTTL:    86400,
	}, nil
}

func (s ApplicationService) LoadAuthTokenSettings() (*AuthTokenSettings, error) {
	return &AuthTokenSettings{
		JwtKey:        []byte("k33)%(7cltD:q.N4AyuXfjAuK{zO,nzP"),
		JwtMethod:     jwt.SigningMethodHS256,
		JwtTTL:        3600,
		RefreshLength: 512,
		RefreshTTL:    86400,
	}, nil
}

func (s ApplicationService) LoadSessionSettings() (*CookieSettings, error) {
	return &CookieSettings{
		Name: "X-AUTH-ONE-TOKEN",
		TTL:  3600,
	}, nil
}

func (s ApplicationService) LoadSocialSettings() (*SocialSettings, error) {
	return &SocialSettings{
		LinkedTokenLength: 128,
		LinkedTTL:         3600,
	}, nil
}

func (s ApplicationService) GetUserIdentityConnection(app *Application, provider string, connection string) (*UserIdentityConnection, error) {
	switch provider {
	case UserIdentityProviderSocial:
		switch connection {
		case "facebook":
			return &UserIdentityConnection{
				ID:                  bson.NewObjectId(),
				AppID:               app.ID,
				Provider:            UserIdentityProviderSocial,
				IsSocial:            true,
				ClientID:            "524897438010722",
				ClientSecret:        "23fdec80c554f037045872339a1305f8",
				ClientScopes:        []string{"email", "user_birthday", "user_friends"},
				EndpointAuthURL:     facebook.Endpoint.AuthURL,
				EndpointTokenURL:    facebook.Endpoint.TokenURL,
				EndpointUserInfoURL: "https://graph.facebook.com/me?fields=id,name,first_name,last_name,email,birthday,picture&access_token=%s",
				Connection:          "facebook",
			}, nil
		case "twitch":
			return &UserIdentityConnection{
				ID:                  bson.NewObjectId(),
				AppID:               app.ID,
				Provider:            UserIdentityProviderSocial,
				IsSocial:            true,
				ClientID:            "r0elllpn5whuyf3et3pm6apqifn9yg",
				ClientSecret:        "2xblkou7im51y2unpidiagcvtiburz",
				ClientScopes:        []string{"user_read", "channel_subscriptions"},
				EndpointAuthURL:     twitch.Endpoint.AuthURL,
				EndpointTokenURL:    twitch.Endpoint.TokenURL,
				EndpointUserInfoURL: "https://api.twitch.tv/kraken/user?client_id=r0elllpn5whuyf3et3pm6apqifn9yg&oauth_token=%s",
				Connection:          "twitch",
			}, nil
		case "google":
			return &UserIdentityConnection{
				ID:                  bson.NewObjectId(),
				AppID:               app.ID,
				Provider:            UserIdentityProviderSocial,
				IsSocial:            true,
				ClientID:            "923530866593-d5leidhhrbv9uhskqmbi6b5b80tulb6f.apps.googleusercontent.com",
				ClientSecret:        "Yscmu_se7_up7M12G6JK4p57",
				ClientScopes:        []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
				EndpointAuthURL:     google.Endpoint.AuthURL,
				EndpointTokenURL:    google.Endpoint.TokenURL,
				EndpointUserInfoURL: "https://www.googleapis.com/oauth2/v1/userinfo?access_token=%s",
				Connection:          "google",
			}, nil
		case "vk":
			return &UserIdentityConnection{
				ID:                  bson.NewObjectId(),
				AppID:               app.ID,
				Provider:            UserIdentityProviderSocial,
				IsSocial:            true,
				ClientID:            "2755495",
				ClientSecret:        "XAD7d9qiUf0ZNEOuqTQj",
				ClientScopes:        []string{"email", "friends"},
				EndpointAuthURL:     vk.Endpoint.AuthURL,
				EndpointTokenURL:    vk.Endpoint.TokenURL,
				EndpointUserInfoURL: "https://api.vk.com/method/users.get?fields=bdate,photo_50&v=5.92&access_token=%s",
				Connection:          "vk",
			}, nil
		}
	}

	return nil, errors.New("not found")
}

func (s ApplicationService) LoadMfaConnection(connection string) ([]*MfaConnection, error) {
	conn := []*MfaConnection{
		{
			Name:    "Application",
			Type:    "otp",
			Channel: "auth1",
		},
	}
	return conn, nil
}