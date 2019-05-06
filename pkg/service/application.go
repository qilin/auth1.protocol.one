package service

import (
	"github.com/ProtocolONE/auth1.protocol.one/pkg/database"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/models"
	"github.com/ProtocolONE/auth1.protocol.one/pkg/persist"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"sync"
)

const ApplicationWatcherChannel = "application"

type ApplicationServiceInterface interface {
	Create(*models.Application) error
	Update(*models.Application) error
	Get(bson.ObjectId) (*models.Application, error)
	LoadSocialSettings() (*models.SocialSettings, error)
	LoadMfaConnection(string) ([]*models.MfaConnection, error)
	AddIdentityProvider(*models.Application, *models.AppIdentityProvider) error
	UpdateIdentityProvider(*models.Application, *models.AppIdentityProvider) error
}

type ApplicationService struct {
	db *mgo.Database
	mx sync.Mutex

	pool    map[bson.ObjectId]*models.Application
	watcher persist.Watcher
}

func NewApplicationService(r InternalRegistry) *ApplicationService {
	a := &ApplicationService{
		db:      r.MgoSession().DB(""),
		pool:    make(map[bson.ObjectId]*models.Application),
		watcher: r.Watcher(),
	}

	a.watcher.SetUpdateCallback(ApplicationWatcherChannel, func(id string) {
		a.mx.Lock()
		defer a.mx.Unlock()

		_, _ = a.loadToCache(bson.ObjectIdHex(id))
	})

	return a
}

func (s ApplicationService) Create(app *models.Application) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if err := s.db.C(database.TableApplication).Insert(app); err != nil {
		return err
	}

	s.pool[app.ID] = app
	return s.watcher.Update(ApplicationWatcherChannel, app.ID.String())
}

func (s ApplicationService) Update(app *models.Application) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if err := s.db.C(database.TableApplication).UpdateId(app.ID, app); err != nil {
		return err
	}

	s.pool[app.ID] = app
	return s.watcher.Update(ApplicationWatcherChannel, app.ID.String())
}

func (s ApplicationService) Get(id bson.ObjectId) (*models.Application, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	app, ok := s.pool[id]
	if !ok {
		var err error

		app, err = s.loadToCache(id)
		if err != nil {
			return nil, err
		}
	}

	return app, nil
}

func (s ApplicationService) LoadSocialSettings() (*models.SocialSettings, error) {
	return &models.SocialSettings{
		LinkedTokenLength: 128,
		LinkedTTL:         3600,
	}, nil
}

func (s ApplicationService) LoadMfaConnection(connection string) ([]*models.MfaConnection, error) {
	conn := []*models.MfaConnection{
		{
			Name:    "Application",
			Type:    "otp",
			Channel: "auth1",
		},
	}
	return conn, nil
}

func (s ApplicationService) AddIdentityProvider(app *models.Application, ip *models.AppIdentityProvider) error {
	app.IdentityProviders = append(app.IdentityProviders, ip)

	return s.Update(app)
}

func (s ApplicationService) UpdateIdentityProvider(app *models.Application, ip *models.AppIdentityProvider) error {
	for index, provider := range app.IdentityProviders {
		if provider.ID == ip.ID {
			app.IdentityProviders[index] = ip
			return s.Update(app)
		}
	}

	return nil
}

func (s ApplicationService) loadToCache(id bson.ObjectId) (*models.Application, error) {
	app := &models.Application{}
	err := s.db.C(database.TableApplication).
		FindId(id).
		One(&app)

	if err != nil {
		return nil, errors.Wrapf(err, "Unable to load application with id %s", id.String())
	}

	s.pool[id] = app
	return app, nil
}