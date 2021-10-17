package authentication

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/SevenTV/Common/auth"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures"
	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/externalapis"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/helpers"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-querystring/query"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
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
			CreatedAt: time.Now(),
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
		token, _, err := auth.VerifyJWT(gCtx.Config().Credentials.JWTSecret, csrfToken)
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
		if csrfClaim.CreatedAt.Before(time.Now().Add(-time.Minute * 5)) {
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

		var grant *OAuth2AuthorizedResponse
		if err = externalapis.ReadRequestResponse(resp, &grant); err != nil {
			logrus.WithError(err).Error("ReadRequestResponse")
			return helpers.HttpResponse(c).SetMessage("Failed to decode data sent by the External Provider").SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
		}

		// Retrieve twitch user data
		users, err := externalapis.Twitch.GetUsers(gCtx, grant.AccessToken)
		if err != nil {
			logrus.WithError(err).Error("Twitch, GetUsers")
			return helpers.HttpResponse(c).SetMessage("Couldn't fetch user data from the External Provider").SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
		}
		if len(users) == 0 {
			return helpers.HttpResponse(c).SetMessage("No user data response from the External Provider").SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
		}
		twUser := users[0]

		// Create a new User
		ub := structures.NewUserBuilder().
			SetUsername(twUser.Login).
			SetEmail(twUser.Email)

		ucb := structures.NewUserConnectionBuilder().
			SetPlatform(structures.UserConnectionPlatformTwitch).
			SetLinkedAt(time.Now()).
			SetTwitchData(twUser).                                                        // Set twitch data
			SetGrant(grant.AccessToken, grant.RefreshToken, grant.ExpiresIn, grant.Scope) // Update the token grant

		// Write to database
		var userID primitive.ObjectID
		{
			// Upsert the connection
			var connection *structures.UserConnection
			doc := gCtx.Inst().Mongo.Collection(mongo.CollectionNameConnections).FindOneAndUpdate(ctx, bson.M{
				"data.id": twUser.ID,
			}, ucb.Update, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(1))
			if err = doc.Decode(&connection); err != nil && err != mongo.ErrNoDocuments {
				logrus.WithError(err).Error("mongo")
				return helpers.HttpResponse(c).SetMessage("Database Write Failed (connection, decode)").SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
			}
			// Add the connection to user object
			ub.AddConnection(connection.ID)

			// Find user
			var user *structures.User
			doc = gCtx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
				"connections": bson.M{
					"$in": []primitive.ObjectID{connection.ID},
				},
			})
			if err = doc.Decode(&user); err == mongo.ErrNoDocuments {
				// User doesn't yet exist: create it
				ub.SetDiscriminator("")
				ub.SetAvatarURL(twUser.ProfileImageURL)
				r, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameUsers).InsertOne(ctx, ub.User)
				if err != nil {
					logrus.WithError(err).Error("mongo")
					return helpers.HttpResponse(c).SetMessage("Database Write Failed (user, stat)").SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()

				}
				userID = r.InsertedID.(primitive.ObjectID)
			} else if err != nil {
				logrus.WithError(err).Error("mongo")
				return helpers.HttpResponse(c).SetMessage("Database Write Failed (user, stat)").SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
			} else {
				// User exists; update
				if err = gCtx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOneAndUpdate(ctx, bson.M{
					"_id": user.ID,
				}, ub.Update, options.FindOneAndUpdate().SetReturnDocument(1)).Decode(&user); err != nil {
					logrus.WithError(err).Error("mongo")
					return helpers.HttpResponse(c).SetMessage("Database Write Failed (user, stat)").SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
				}
			}
		}

		// Generate an access token for the user
		tokenTTL := time.Now().Add(time.Hour * 168)
		userToken, err := auth.SignJWT(gCtx.Config().Credentials.JWTSecret, &auth.JWTClaimUser{
			UserID:       userID.Hex(),
			TokenVersion: 0.0,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer: "7TV-API-REST",
				ExpiresAt: &jwt.NumericDate{
					Time: tokenTTL,
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Error("jwt")
			return helpers.HttpResponse(c).SetMessage(fmt.Sprintf("Token Sign Failure (%s)", err.Error())).SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
		}

		// Define a cookie
		c.Cookie(&fiber.Cookie{
			Name:     "access_token",
			Value:    userToken,
			Domain:   gCtx.Config().Http.CookieDomain,
			Expires:  tokenTTL,
			Secure:   gCtx.Config().Http.CookieSecure,
			HTTPOnly: true,
		})

		// Redirect to website's callback page
		params, _ = query.Values(&OAuth2CallbackAppParams{
			Token: userToken,
		})
		return c.Redirect(fmt.Sprintf("%s/oauth2?%s", gCtx.Config().WebsiteURL, params.Encode()))
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

type OAuth2AuthorizedResponse struct {
	TokenType    string   `json:"token_type"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	Scope        []string `json:"scope"`
	ExpiresIn    int      `json:"expires_in"`
}

type OAuth2CallbackAppParams struct {
	Token string `url:"token"`
}

var twitchScopes = []string{
	"user:read:email",
}

const TWITCH_CSRF_COOKIE_NAME = "csrf_token_tw"
