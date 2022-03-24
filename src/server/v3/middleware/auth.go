package middleware

import (
	"strings"
	"time"

	"github.com/SevenTV/Common/auth"
	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/structures/v3/aggregations"
	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Auth(gCtx global.Context) rest.Middleware {
	return func(ctx *rest.Ctx) rest.APIError {
		// Parse token from header
		h := utils.B2S(ctx.Request.Header.Peek("Authorization"))
		s := strings.Split(h, "Bearer ")
		if len(s) != 2 {
			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": "Bad Authorization Header"})
		}
		t := s[1]

		// Verify the token
		claims := &auth.JWTClaimUser{}
		_, err := auth.VerifyJWT(gCtx.Config().Credentials.JWTSecret, strings.Split(t, "."), claims)
		if err != nil {
			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": err.Error()})
		}

		// User ID from parsed token
		if claims.UserID == "" {
			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": "Bad Token"})
		}
		userID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": err.Error()})
		}

		// Version of parsed token
		user := &structures.User{}

		pipeline := mongo.Pipeline{{{Key: "$match", Value: bson.M{"_id": userID}}}}
		pipeline = append(pipeline, aggregations.UserRelationRoles...)
		pipeline = append(pipeline, aggregations.UserRelationBans...)
		cur, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameUsers).Aggregate(ctx, pipeline)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return errors.ErrUnauthorized().SetFields(errors.Fields{"message": "Token has Unknown Bound User"})
			}

			logrus.WithError(err).Error("mongo")
			return errors.ErrInternalServerError()
		}
		cur.Next(ctx)
		if err := cur.Decode(user); err != nil {
			logrus.WithError(err).Error("mongo")
			return errors.ErrInternalServerError()
		}

		_ = cur.Close(ctx)

		if user.TokenVersion != claims.TokenVersion {
			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": "Token Version Mismatch"})
		}

		// Check bans
		for _, ban := range user.Bans {
			// Check for No Auth effect
			if ban.Effects.Has(structures.BanEffectNoAuth) {
				return errors.ErrInsufficientPrivilege().
					SetDetail("You are banned").
					SetFields(errors.Fields{
						"ban_reason":      ban.Reason,
						"ban_expire_date": ban.ExpireAt.Format(time.RFC3339),
					})
			}
			// Check for No Permissions effect
			if ban.Effects.Has(structures.BanEffectNoPermissions) {
				user.Roles = []*structures.Role{structures.RevocationRole}
			}
		}

		defaultRoles, _ := gCtx.Inst().Query.Roles(ctx, bson.M{"default": true})
		user.AddRoles(defaultRoles...)

		ctx.SetActor(user)
		return nil
	}
}
