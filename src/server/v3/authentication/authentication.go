package authentication

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/SevenTV/Common/auth"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/helpers"
	"github.com/SevenTV/REST/src/server/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Authentication(gCtx global.Context, router fiber.Router) {
	group := router.Group("/auth")
	twitch(gCtx, group)

	group.Get("/sign", func(c *fiber.Ctx) error {
		claim, err := json.Marshal(map[string]interface{}{
			"u": "XD",
			"x": []int{1, 2, 3},
		})
		if err != nil {
			logrus.WithError(err).Error("json")
			return c.SendStatus(500)
		}

		token, err := gCtx.Inst().Auth.Sign(gCtx.Config().NodeName, claim)
		if err != nil {
			logrus.WithError(err).Error("sign")
			return c.SendStatus(500)
		}

		return c.SendString(token)
	})

	group.Get("/verify", func(c *fiber.Ctx) error {
		t := c.Query("token")

		_, err := gCtx.Inst().Auth.Verify(t)
		if err != nil {
			logrus.WithError(err).Error("verify")
			return c.SendStatus(500)
		}

		return c.SendStatus(200)
	})

	// User Impersonation
	group.Get("/impersonate/:user", middleware.Auth(gCtx), func(c *fiber.Ctx) error {
		ctx := c.Context()
		// Get the actor
		actor, ok := c.Locals(helpers.UserKey).(*structures.User)
		if !ok {
			return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeUnauthorized).SendAsError()
		}
		if !actor.HasPermission(structures.RolePermissionManageStack) { // must be privileged
			return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeForbidden).SendAsError()
		}

		// Parse ID of the user to impersonate
		victimID, err := primitive.ObjectIDFromHex(c.Params("user"))
		if err != nil {
			return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SetMessage("Bad Object ID").SendAsError()
		}

		// Retrieve the victim's data
		victim := &structures.User{}
		if err = gCtx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
			"_id": victimID,
		}).Decode(victim); err != nil {
			if err == mongo.ErrNoDocuments {
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SetMessage("Unknown User").SendAsError()
			}
			return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
		}
		fmt.Println(victimID, victim)

		// Create a token that authenticates as the victim
		token, err := auth.SignJWT(gCtx.Config().Credentials.JWTSecret, &auth.JWTClaimUser{
			UserID:       victimID.Hex(),
			TokenVersion: victim.TokenVersion,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer: "7TV-API-REST",
				ExpiresAt: &jwt.NumericDate{
					Time: time.Now().Add(time.Hour * 1), // token is valid for 1 hour
				},
			},
		})
		if err != nil {
			return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendAsError()
		}

		// Send response
		j, _ := json.Marshal(ImpersonateUserResponse{
			Token: token,
			User:  victim,
		})
		return helpers.HttpResponse(c).SetStatus(fiber.StatusOK).Send(j)
	})
}

type ImpersonateUserResponse struct {
	Token string           `json:"token"`
	User  *structures.User `json:"user"`
}
