package ClientAuth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
)

var (
	scopes = []string{
		drive.DriveScope, drive.DriveAppdataScope,
		drive.DriveMetadataScope, drive.DriveMetadataReadonlyScope,
	}
	conf = &oauth2.Config{
		// ClientID:     "492799813423-5acrke2ei1lrcuf5fiqu21ugcgftlcmr.XXXXXX",
		// ClientSecret: "GOCSPX-XXXXXXXXXXX",
		// Scopes:       scopes,
		// // Url of the application callback where the server is sending the response of authorization include code
		// RedirectURL: "https://path/sh/oauth2/driver/callback",
		// Endpoint: oauth2.Endpoint{
		// 	// Url of authentication server
		// 	AuthURL:  "https://accounts.google.com/o/oauth2/auth",
		// 	TokenURL: "https://oauth2.googleapis.com/token",
		// },
	}
)

type CodeEvent struct {
	Code string
}

type UrlEvent struct {
	Url string
}

type VerifierEvent struct {
	Verifier string
}

type TokenEvent struct {
	Token       *oauth2.Token
	TokenSource oauth2.TokenSource
	BytesToken  []byte
}

type Options struct {
	authConfig *oauth2.Config
}

type Option func(*Options)

type ClientAuth struct {
	opts     Options
	Code     chan CodeEvent
	Url      chan UrlEvent
	Verifier chan VerifierEvent
	Token    chan TokenEvent
	state    string
	urlAuth  string
}

func defaultOptions() Options {
	return Options{
		authConfig: conf,
	}
}

func WithClientID(clientID string) Option {
	return func(opts *Options) {
		opts.authConfig.ClientID = clientID
	}
}

func WithClientSecret(clientSecret string) Option {
	return func(opts *Options) {
		opts.authConfig.ClientSecret = clientSecret
	}
}

func WithRedirectURL(redirectURL string) Option {
	return func(opts *Options) {
		opts.authConfig.RedirectURL = redirectURL
	}
}

func WithAuthURL(authURL string) Option {
	return func(opts *Options) {
		opts.authConfig.Endpoint.AuthURL = authURL
	}
}

func WithTokenURL(tokenURL string) Option {
	return func(opts *Options) {
		opts.authConfig.Endpoint.TokenURL = tokenURL
	}
}

func WithScopes(scopes []string) Option {
	return func(opts *Options) {
		opts.authConfig.Scopes = scopes
	}
}

func NewClientAuth(options ...Option) (*ClientAuth, error) {
	opts := defaultOptions()

	for _, opt := range options {
		opt(&opts)
	}

	if len(opts.authConfig.Scopes) == 0 {
		return nil, fmt.Errorf("no scopes provided")
	}

	if opts.authConfig == nil {
		return nil, fmt.Errorf("no auth config provided")
	}

	if opts.authConfig.ClientID == "" {
		return nil, fmt.Errorf("no client id provided")
	}

	if opts.authConfig.ClientSecret == "" {
		return nil, fmt.Errorf("no client secret provided")
	}

	if opts.authConfig.Endpoint.AuthURL == "" {
		return nil, fmt.Errorf("no auth url provided")
	}

	if opts.authConfig.RedirectURL == "" {
		return nil, fmt.Errorf("no redirect url provided")
	}

	if opts.authConfig.Endpoint.TokenURL == "" {
		return nil, fmt.Errorf("no token url provided")
	}

	auth := &ClientAuth{
		opts: opts,
	}

	auth.Code = make(chan CodeEvent, 1)
	auth.Url = make(chan UrlEvent, 1)
	auth.Verifier = make(chan VerifierEvent, 1)
	auth.Token = make(chan TokenEvent, 1)

	auth.state = randState()

	return auth, nil
}

func (self *ClientAuth) GetUrlAuth() {
	// use PKCE to protect against CSRF attacks
	// https://www.ietf.org/archive/id/draft-ietf-oauth-security-topics-22.html#name-countermeasures-6
	verifier := oauth2.GenerateVerifier()
	self.Verifier <- VerifierEvent{verifier}

	fmt.Println("State : ", self.GetState())

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL(self.GetState(), oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	self.Url <- UrlEvent{url}
}

func (self *ClientAuth) GetState() string {
	return self.state
}

// Guarda o token em um arquivo
func (*ClientAuth) SetTokenInFile(tok *oauth2.Token) {
	f, e := os.Create("token.json")
	if e != nil {
		fmt.Println("Error creating file: ", e)
	}
	defer f.Close()

	json.NewEncoder(f).Encode(tok)
}

func (*ClientAuth) TokenToBytes(tok *oauth2.Token) []byte {
	f := new(bytes.Buffer)

	json.NewEncoder(f).Encode(tok)

	return f.Bytes()
}

// Recupera o token de um arquivo
func (*ClientAuth) GetTokenOfFile(nameFile string) *oauth2.Token {
	f, e := os.Open(nameFile)
	if e != nil {
		return nil
	}
	defer f.Close()

	tok := &oauth2.Token{}
	json.NewDecoder(f).Decode(tok)
	return tok
}

func (*ClientAuth) GetTokenOfStr(tokenStr string) *oauth2.Token {
	f := bytes.NewBufferString(tokenStr)

	tok := &oauth2.Token{}
	json.NewDecoder(f).Decode(tok)

	return tok
}

// Verifica se o token nao expirou return true if token not expired
func (*ClientAuth) tokenNotExpired(tok *oauth2.Token) bool {
	if tok.Expiry.IsZero() {
		log.Fatalln("Expiry is zero")
		return true
	}

	log.Println("After: ", tok.Expiry.After(time.Now()))
	log.Println("Valid: ", tok.Valid())
	return tok.Valid() && tok.Expiry.After(time.Now())
}

// Gera um state aleatorio
func randState() string {
	return string(string([]byte(rand.Text())[0:10]))
}

// GetToken returns the token in bytes format, refreshing if necessary.
// The token is retrieved from a file named "token.json" in the current working directory.
// If the file does not exist, it will be created.
// If the token is nil or not valid, it will redirect the user to the consent page
// to ask for permission for the scopes specified above.
// The HTTP Client returned by conf.Client will refresh the token as necessary.
func (self *ClientAuth) GetToken() {
	var (
		tok *oauth2.Token = self.GetTokenOfFile("token.json")
		err error
		ctx = context.Background()
	)

	//  if not existe token or not valid
	if tok == nil {

		// url, verifier := self.GetUrlAuth()
		go self.GetUrlAuth()
		// Use the authorization code that is pushed to the redirect
		// URL. Exchange will do the handshake to retrieve the
		// initial access token. The HTTP Client returned by
		// conf.Client will refresh the token as necessary.
		var code string

		verifier := (<-self.Verifier).Verifier
		// Redirect user to consent page to ask for permission
		// for the scopes specified above.

		//  get code
		// code = funcGetCode()
		code = (<-self.Code).Code

		fmt.Println("Code 1: ", code)
		// Troca o code pelo token
		tok, err = conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
		if err != nil {
			fmt.Println(err)
		}
	} else if !self.tokenNotExpired(tok) {
		// tok = getTokenOfFile("token.json")

		// refresh token if is expired else return tokenSource
		tok, err = conf.TokenSource(ctx, tok).Token()
		if err != nil {
			log.Fatal(err)
		}
	}

	byteToken := self.TokenToBytes(tok)
	self.Token <- TokenEvent{Token: tok, BytesToken: byteToken, TokenSource: conf.TokenSource(ctx, tok)}
}
