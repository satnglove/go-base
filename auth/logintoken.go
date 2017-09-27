package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var (
	errTokenNotFound = errors.New("login token not found")
)

// loginToken is an in-memory saved token referencing an account ID and an expiry date.
type loginToken struct {
	Token     string
	AccountID int
	Expiry    time.Time
}

// LoginTokenAuth implements passwordless login authentication flow using temporary in-memory stored tokens.
type LoginTokenAuth struct {
	token            map[string]loginToken
	mux              sync.RWMutex
	loginURL         string
	loginTokenLength int
	loginTokenExpiry time.Duration
}

// NewLoginTokenAuth configures and returns a LoginToken authentication instance.
func NewLoginTokenAuth() (*LoginTokenAuth, error) {
	a := &LoginTokenAuth{
		token:            make(map[string]loginToken),
		loginURL:         viper.GetString("auth_login_url"),
		loginTokenLength: viper.GetInt("auth_login_token_length"),
		loginTokenExpiry: viper.GetDuration("auth_login_token_expiry"),
	}
	return a, nil
}

// CreateToken creates an in-memory login token referencing account ID. It returns a token containing a random tokenstring and expiry date.
func (a *LoginTokenAuth) CreateToken(id int) loginToken {
	lt := loginToken{
		Token:     randStringBytes(a.loginTokenLength),
		AccountID: id,
		Expiry:    time.Now().Add(time.Minute * a.loginTokenExpiry),
	}
	a.add(lt)
	a.purgeExpired()
	return lt
}

// GetAccountID looks up the token by tokenstring and returns the account ID or error if token not found or expired.
func (a *LoginTokenAuth) GetAccountID(token string) (int, error) {
	lt, exists := a.get(token)
	if !exists || time.Now().After(lt.Expiry) {
		return 0, errTokenNotFound
	}
	a.delete(lt.Token)
	return lt.AccountID, nil
}

func (a *LoginTokenAuth) get(token string) (loginToken, bool) {
	a.mux.RLock()
	lt, ok := a.token[token]
	a.mux.RUnlock()
	return lt, ok
}

func (a *LoginTokenAuth) add(lt loginToken) {
	a.mux.Lock()
	a.token[lt.Token] = lt
	a.mux.Unlock()
}

func (a *LoginTokenAuth) delete(token string) {
	a.mux.Lock()
	delete(a.token, token)
	a.mux.Unlock()
}

func (a *LoginTokenAuth) purgeExpired() {
	for t, v := range a.token {
		if time.Now().After(v.Expiry) {
			a.delete(t)
		}
	}
}