package mongo

import (
	"errors"
	"time"

	"github.com/ProtocolONE/auth1.protocol.one/internal/domain/entity"
	"github.com/globalsign/mgo/bson"
)

type model struct {
	// ID is the id of user.
	ID bson.ObjectId `bson:"_id" json:"id"`

	// AppID is the id of the application.
	AppID bson.ObjectId `bson:"app_id" json:"app_id"`

	// Email is the email address of the user.
	Email string `bson:"email" json:"email" validate:"required,email"`

	// EmailVerified is status of verification user address.
	EmailVerified bool `bson:"email_verified" json:"email_verified"`

	// PhoneNumber is the phone number of the user.
	PhoneNumber string `bson:"phone_number" json:"phone_number"`

	// PhoneVerified is status of verification user phone.
	PhoneVerified bool `bson:"phone_verified" json:"phone_verified"`

	// Username is the nickname of the user.
	Username string `bson:"username" json:"username"`

	// UniqueUsername is index flag that username must be unique within app.
	UniqueUsername bool `bson:"unique_username" json:"-"`

	// Name is the name of the user. Contains first anf last name.
	Name string `bson:"name" json:"name"`

	// Picture is the avatar of the user.
	Picture string `bson:"picture" json:"picture"`

	// LastIp returns the ip of the last login.
	LastIp string `bson:"last_ip" json:"last_ip"`

	// LastLogin returns the timestamp of the last login.
	LastLogin time.Time `bson:"last_login" json:"last_login"`

	// LoginsCount contains count authorization for the user.
	LoginsCount int `bson:"logins_count" json:"logins_count"`

	// Blocked is status of user blocked.
	Blocked bool `bson:"blocked" json:"blocked"`

	// DeviceID is unique user client identifier
	DeviceID []string `bson:"device_id" json:"device_id"`

	// CreatedAt returns the timestamp of the user creation.
	CreatedAt time.Time `bson:"created_at" json:"created_at"`

	// UpdatedAt returns the timestamp of the last update.
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

func (m model) Convert() *entity.User {
	return &entity.User{
		ID:             m.ID.Hex(),
		AppID:          m.AppID.Hex(),
		Email:          m.Email,
		EmailVerified:  m.EmailVerified,
		Phone:          m.PhoneNumber,
		PhoneVerified:  m.PhoneVerified,
		Username:       m.Username,
		UniqueUsername: m.UniqueUsername,
		Name:           m.Name,
		Picture:        m.Picture,
		LastIp:         m.LastIp,
		LastLogin:      m.LastLogin,
		LoginsCount:    m.LoginsCount,
		Blocked:        m.Blocked,
		DeviceID:       m.DeviceID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func newModel(i *entity.User) (*model, error) {
	if i.ID == "" {
		return nil, errors.New("User.ID is empty")
	}
	if i.AppID == "" {
		return nil, errors.New("User.AppID is empty")
	}
	return &model{
		ID:             bson.ObjectIdHex(i.ID),
		AppID:          bson.ObjectIdHex(i.AppID),
		Email:          i.Email,
		EmailVerified:  i.EmailVerified,
		PhoneNumber:    i.Phone,
		PhoneVerified:  i.PhoneVerified,
		Username:       i.Username,
		UniqueUsername: i.UniqueUsername,
		Name:           i.Name,
		Picture:        i.Picture,
		LastIp:         i.LastIp,
		LastLogin:      i.LastLogin,
		LoginsCount:    i.LoginsCount,
		Blocked:        i.Blocked,
		DeviceID:       i.DeviceID,
		CreatedAt:      i.CreatedAt,
		UpdatedAt:      i.UpdatedAt,
	}, nil
}
