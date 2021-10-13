package authentication

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/auth"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/helpers"
	"github.com/gofiber/fiber/v2"
	"github.com/google/go-querystring/query"
	"github.com/sirupsen/logrus"
)

func twitch(gCtx global.Context, router fiber.Router) {
	group := router.Group("twitch")

	group.Get("/", func(c *fiber.Ctx) error {
		// Generate a randomized value for a CSRF token
		csrfValue, err := utils.GenerateRandomString(64)
		if err != nil {
			logrus.WithError(err).Error("csrf, random bytes")
			return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SetMessage(err.Error()).SendAsError()
		}

		// Sign a JWT with the CSRF bytes
		csrfToken, err := auth.SignJWT(gCtx.Config().Credentials.JWTSecret, auth.JWTClaimOAuth2CSRF{
			State:     csrfValue,
			CreatedAt: time.Now().UnixMilli(),
		})
		if err != nil {
			logrus.WithError(err).Error("csrf, jwt")
			return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SetMessage(err.Error()).SendAsError()
		}

		// Set cookie
		c.Cookie(&fiber.Cookie{
			Name:     TWITCH_CSRF_COOKIE_NAME,
			Value:    csrfToken,
			Expires:  time.Now().Add(time.Minute * 5),
			Domain:   gCtx.Config().Http.CookieDomain,
			Secure:   gCtx.Config().Http.CookieSecure,
			HTTPOnly: true,
		})

		// Format querystring options for the redirection URL
		params, err := query.Values(&OAuth2URLParams{
			ClientID:     gCtx.Config().Platforms.Twitch.ClientID,
			RedirectURI:  gCtx.Config().Platforms.Twitch.RedirectURI,
			ResponseType: "code",
			Scope:        strings.Join(twitchScopes, " "),
			State:        csrfValue,
		})
		if err != nil {
			logrus.WithError(err).Error("querystring")
			return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SetMessage(err.Error()).SendAsError()
		}

		// Redirect the client
		return c.Redirect(fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?%s", params.Encode()))
	})

	group.Get("/callback", func(c *fiber.Ctx) error {
		ctx := c.Context()
		// Get state parameter
		state := c.Query("state")
		if state == "" {
			return helpers.HttpResponse(c).SetMessage("Missing State Parameter").SetStatus(helpers.HttpStatusCodeBadRequest).SendAsError()
		}

		// Retrieve the CSRF token from cookies
		csrfToken := strings.Split(c.Cookies(TWITCH_CSRF_COOKIE_NAME), ".")
		if len(csrfToken) != 3 {
			return helpers.HttpResponse(c).
				SetMessage(fmt.Sprintf("Bad State (found %d segments when 3 were expected)", len(csrfToken))).
				SetStatus(helpers.HttpStatusCodeBadRequest).
				SendAsError()
		}

		// Verify the token
		var csrfClaim *auth.JWTClaimOAuth2CSRF
		token, err := auth.VerifyJWT(gCtx.Config().Credentials.JWTSecret, csrfToken)
		if err != nil {
			logrus.WithError(err).Error("jwt")
			return helpers.HttpResponse(c).SetMessage(fmt.Sprintf("Invalid State: %s", err.Error())).SetStatus(helpers.HttpStatusCodeBadRequest).SendAsError()
		}
		{
			b, err := json.Marshal(token.Claims)
			if err != nil {
				logrus.WithError(err).Error("json")
				return helpers.HttpResponse(c).SetMessage(fmt.Sprintf("Invalid State: %s", err.Error())).SetStatus(helpers.HttpStatusCodeBadRequest).SendAsError()
			}

			if err = json.Unmarshal(b, &csrfClaim); err != nil {
				logrus.WithError(err).Error("json")
				return helpers.HttpResponse(c).SetMessage(fmt.Sprintf("Invalid State: %s", err.Error())).SetStatus(helpers.HttpStatusCodeBadRequest).SendAsError()
			}
		}

		// Validate the token
		// Check date matches
		if time.UnixMilli(csrfClaim.CreatedAt).Before(time.Now().Add(-time.Minute * 5)) {
			return helpers.HttpResponse(c).SetMessage("Expired State").SetStatus(helpers.HttpStatusCodeBadRequest).SendAsError()
		}

		// Check token value mismatch
		if state != csrfClaim.State {
			return helpers.HttpResponse(c).SetMessage("Mismatching State Value").SetStatus(helpers.HttpStatusCodeBadRequest).SendAsError()
		}

		// Remove the CSRF cookie
		c.Cookie(&fiber.Cookie{
			Name:     TWITCH_CSRF_COOKIE_NAME,
			Expires:  time.Now(),
			Domain:   gCtx.Config().Http.CookieDomain,
			Secure:   gCtx.Config().Http.CookieSecure,
			HTTPOnly: true,
		}) // We have now validated this request is authentic.

		// OAuth2 auhorization code for granting an access token
		code := c.Query("code")

		// Format querystring for our authenticated request to twitch
		params, err := query.Values(&OAuth2AuthorizationParams{
			ClientID:     gCtx.Config().Platforms.Twitch.ClientID,
			ClientSecret: gCtx.Config().Platforms.Twitch.ClientSecret,
			RedirectURI:  gCtx.Config().Platforms.Twitch.RedirectURI,
			Code:         code,
			GrantType:    "authorization_code",
		})
		if err != nil {
			logrus.WithError(err).Error("querystring")
			return helpers.HttpResponse(c).SetMessage(err.Error()).SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
		}

		// Prepare a HTTP request to Twitch to convert code to acccess token
		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://id.twitch.tv/oauth2/token?%s", params.Encode()), nil)
		if err != nil {
			logrus.WithError(err).Error("twitch")
			return helpers.HttpResponse(c).
				SetMessage("Internal Request to External Provider Failed").
				SetStatus(helpers.HttpStatusCodeInternalServerError).
				SendAsError()
		}

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logrus.WithError(err).Error("twitch")
			return helpers.HttpResponse(c).
				SetMessage("Internal Request Rejected by External Provider").SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
		}
		defer resp.Body.Close()
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			logrus.WithError(err).Error("ioutil, ReadAll")
			return helpers.HttpResponse(c).SetMessage("Unreadable Response From External Provider").SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
		}

		// todo: create / retrieve user

		return c.SendStatus(200)
	})
}

type OAuth2URLParams struct {
	ClientID     string `url:"client_id"`
	RedirectURI  string `url:"redirect_uri"`
	ResponseType string `url:"response_type"`
	Scope        string `url:"scope"`
	State        string `url:"state"`
}

type OAuth2AuthorizationParams struct {
	ClientID     string `url:"client_id"`
	ClientSecret string `url:"client_secret"`
	RedirectURI  string `url:"redirect_uri"`
	Code         string `url:"code"`
	GrantType    string `url:"grant_type"`
}

var twitchScopes = []string{
	"user:read:email",
}

const TWITCH_CSRF_COOKIE_NAME = "csrf_token_tw"
