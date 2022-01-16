package middleware

import (
	"strings"
	"time"

	"github.com/SevenTV/Common/auth"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/structures/v3/aggregations"
	"github.com/SevenTV/REST/src/global"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Auth(gCtx global.Context) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		// Parse token from header
		h := c.Get("Authorization")
		s := strings.Split(h, "Bearer ")
		if len(s) != 2 {
			return c.Status(401).JSON(&fiber.Map{"error": "Bad Authorization Header"})
		}
		t := s[1]

		// Verify the token
		_, claims, err := auth.VerifyJWT(gCtx.Config().Credentials.JWTSecret, strings.Split(t, "."))
		if err != nil {
			return c.Status(401).JSON(&fiber.Map{"error": err.Error()})
		}

		// User ID from parsed token
		u := claims["u"]
		if u == nil {
			return c.Status(401).JSON(&fiber.Map{"error": "Bad Token"})
		}
		userID, err := primitive.ObjectIDFromHex(u.(string))
		if err != nil {
			return c.Status(401).JSON(&fiber.Map{"error": err.Error()})
		}

		// Version of parsed token
		user := &structures.User{}
		v := claims["v"].(float64)

		pipeline := mongo.Pipeline{{{Key: "$match", Value: bson.M{"_id": userID}}}}
		pipeline = append(pipeline, aggregations.UserRelationRoles...)
		pipeline = append(pipeline, aggregations.UserRelationBans...)
		cur, err := gCtx.Inst().Mongo.Collection(structures.CollectionNameUsers).Aggregate(ctx, pipeline)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.Status(401).JSON(&fiber.Map{"error": "Token has Unknown Bound User"})
			}

			logrus.WithError(err).Error("mongo")
			return c.SendStatus(500)
		}
		cur.Next(ctx)
		if err := cur.Decode(user); err != nil {
			logrus.WithError(err).Error("mongo")
			return c.SendStatus(500)
		}

		_ = cur.Close(ctx)

		// Check bans
		for _, ban := range user.Bans {
			// Check for No Auth effect
			if ban.HasEffect(structures.BanEffectNoAuth) {
				return c.Status(fiber.StatusForbidden).JSON(&fiber.Map{
					"error": "You are banned",
					"ban": &fiber.Map{
						"reason":    ban.Reason,
						"expire_at": ban.ExpireAt.Format(time.RFC3339),
					},
				})
			}
			// Check for No Permissions effect
			if ban.HasEffect(structures.BanEffectNoPermissions) {
				user.Roles = []*structures.Role{structures.RevocationRole}

			}
		}
		defaultRoles := structures.DefaultRoles.Fetch(ctx, gCtx.Inst().Mongo, gCtx.Inst().Redis)
		user.AddRoles(defaultRoles...)

		if user.TokenVersion != v {
			return c.Status(401).JSON(&fiber.Map{"error": "Token Version Mismatch"})
		}

		c.Locals("user", user)
		return c.Next()
	}
}
