package cosmetics

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	"github.com/SevenTV/REST/src/server/v2/model"
	"github.com/SevenTV/REST/src/server/v3/middleware"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

// Config implements rest.Route
func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/cosmetics",
		Method: rest.GET,
		Children: []rest.Route{
			newAvatars(r.Ctx),
		},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 150, []string{"s-maxage=300"}),
		},
	}
}

// Get Cosmetics
// @Summary Get Cosmetics
// @Description Get all active cosmetics and the users assigned to them
// @Tags cosmetics
// @Param user_identifier query string false "one of 'object_id', 'twitch_id' or 'login'"
// @Produce json
// @Success 200 {object} model.CosmeticsMap
// @Router /cosmetics [get]
func (r *Route) Handler(ctx *rest.Ctx) errors.APIError {
	// identifier type argument
	idType := utils.B2S(ctx.QueryArgs().Peek("user_identifier"))

	if !utils.Contains([]string{"object_id", "twitch_id", "login"}, idType) {
		return errors.ErrInvalidRequest().SetDetail("Query Param 'user_identifier' must be 'object_id', 'twitch_id' or 'login'")
	}

	// Compose cache key
	cacheKey := r.Ctx.Inst().Redis.ComposeKey("rest", fmt.Sprintf("cache:cosmetics:%s", idType))

	// Return existing cache?
	d, err := r.Ctx.Inst().Redis.Get(ctx, cacheKey)
	if err == nil && d != "" {
		result := &model.CosmeticsMap{}
		if err := json.Unmarshal(utils.S2B(d), result); err != nil {
			logrus.WithError(err).Error("failed to return cache of /v2/cosmetics")
		}
		return ctx.JSON(rest.OK, result)
	}

	// Fetch roles
	roles, _ := r.Ctx.Inst().Query.Roles(ctx, bson.M{})
	roleMap := make(map[primitive.ObjectID]*structures.Role)
	for _, r := range roles {
		roleMap[r.ID] = r
	}

	// Let's make a pipeline
	pipeline := mongo.Pipeline{
		{{Key: "$sort", Value: bson.M{"priority": -1}}},
		{{Key: "$match", Value: bson.M{
			"disabled": bson.M{"$not": bson.M{"$eq": true}},
			"kind": bson.M{"$in": []structures.EntitlementKind{
				structures.EntitlementKindRole,
				structures.EntitlementKindBadge,
				structures.EntitlementKindPaint,
			}},
		}}},
		// Lookup cosmetics
		{{
			Key: "$group",
			Value: bson.M{
				"_id": nil,
				"entitlements": bson.M{
					"$push": "$$ROOT",
				},
			},
		}},
		// Lookup: Users
		{{
			Key: "$lookup",
			Value: mongo.Lookup{
				From:         mongo.CollectionNameUsers,
				LocalField:   "entitlements.user_id",
				ForeignField: "_id",
				As:           "users",
			},
		}},
		{{Key: "$project", Value: bson.M{
			"cosmetics":                  1,
			"entitlements._id":           1,
			"entitlements.kind":          1,
			"entitlements.data":          1,
			"entitlements.user_id":       1,
			"users.connections.id":       1,
			"users.connections.platform": 1,
			"users.username":             1,
			"users._id":                  1,
			"users.role_ids":             1,
		}}},
	}

	// Run the aggregation
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).Aggregate(ctx, pipeline)
	if err != nil {
		logrus.WithError(err).Error("mongo, failed to spawn cosmetic entitlements aggregation")
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}

	// Decode data
	data := &aggregatedCosmeticsResult{}
	cur.Next(ctx)
	if err = multierror.Append(cur.Decode(data), cur.Close(ctx)).ErrorOrNil(); err != nil {
		logrus.WithError(err).Error("mongo, failed to decode aggregated cosmetic entitlements")
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}

	// We will now recompose the data into
	// an API v2 /cosmetics response

	// Map cosmetics
	cosmetics := []*structures.Cosmetic{}
	cur, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCosmetics).Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.M{"priority": -1}),
	)
	if err != nil {
		logrus.WithError(err).Error("mongo, failed to fetch cosmetics data")
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}
	if err = cur.All(ctx, &cosmetics); err != nil {
		logrus.WithError(err).Error("mongo, failed to decode cosmetics data")
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}
	cosMap := make(map[primitive.ObjectID]*structures.Cosmetic)
	for _, cos := range cosmetics {
		cosMap[cos.ID] = cos
	}

	// Structure entitlements by kind
	// kind:ent_id:[]ent
	ents := make(map[structures.EntitlementKind]map[primitive.ObjectID]*structures.Entitlement)
	for _, ent := range data.Entitlements {
		m := ents[ent.Kind]
		if m == nil {
			ents[ent.Kind] = map[primitive.ObjectID]*structures.Entitlement{}
			m = ents[ent.Kind]
		}
		m[ent.ID] = ent
	}

	// Map users with their roles
	userMap := make(map[primitive.ObjectID]*structures.User)
	userCosmetics := make(map[primitive.ObjectID][2]bool) // [0]: badge, [1] paint
	for _, u := range data.Users {
		userMap[u.ID] = u
		userCosmetics[u.ID] = [2]bool{false, false}
	}
	for _, ent := range ents[structures.EntitlementKindRole] {
		u := userMap[ent.UserID]
		rol := ent.GetData().ReadRole()
		if u != nil && utils.Contains(u.RoleIDs, rol.ObjectReference) {
			continue
		}
		u.RoleIDs = append(u.RoleIDs, rol.ObjectReference)
	}

	usersToIdentifiers := func(ul []*structures.User) []string {
		s := make([]string, len(ul))
		switch idType {
		case "object_id":
			for i, u := range ul {
				s[i] = u.ID.Hex()
			}
		case "login":
			for i, u := range ul {
				s[i] = u.Username
			}
		case "twitch_id":
			for i, u := range ul {
				for _, con := range u.Connections {
					if con.Platform == structures.UserConnectionPlatformTwitch {
						s[i] = con.ID
						break
					}
				}
			}
		}
		return s
	}

	// Create the final result
	result := &model.CosmeticsMap{
		Badges: []*model.CosmeticBadge{},
		Paints: []*model.CosmeticPaint{},
	}
	for _, ent := range ents[structures.EntitlementKindBadge] {
		entd := ent.GetData().ReadItem()
		cos := cosMap[entd.ObjectReference]
		u := userMap[ent.UserID]
		uc := userCosmetics[u.ID]
		if uc[0] || !entd.Selected {
			continue // user already has a badge
		}

		if entd.RoleBinding == nil || utils.Contains(u.RoleIDs, *entd.RoleBinding) {
			cos.Users = append(cos.Users, u)
			uc[0] = true
			userCosmetics[u.ID] = uc
		}
	}
	for _, ent := range ents[structures.EntitlementKindPaint] {
		entd := ent.GetData().ReadItem()
		cos := cosMap[entd.ObjectReference]
		u := userMap[ent.UserID]
		uc := userCosmetics[u.ID]
		if uc[1] || !entd.Selected {
			continue // user already has a paint
		}

		if entd.RoleBinding == nil || utils.Contains(u.RoleIDs, *entd.RoleBinding) {
			cos.Users = append(cos.Users, u)
			uc[1] = true
			userCosmetics[u.ID] = uc
		}
	}

	for _, cos := range cosmetics {
		if len(cos.Users) == 0 {
			continue // skip if cosmetic has no users
		}
		switch cos.Kind {
		case structures.CosmeticKindBadge:
			badge := cos.ReadBadge()
			urls := make([][2]string, 3)
			for i := 1; i <= 3; i++ {
				a := [2]string{}
				a[0] = strconv.Itoa(i)
				a[1] = fmt.Sprintf("https://%s/badge/%s/%dx", r.Ctx.Config().CdnURL, badge.ID.Hex(), i)
				urls[i-1] = a
			}
			result.Badges = append(result.Badges, &model.CosmeticBadge{
				ID:      cos.ID.Hex(),
				Name:    cos.Name,
				Tooltip: badge.Tooltip,
				URLs:    urls,
				Users:   usersToIdentifiers(cos.Users),
				Misc:    false,
			})
		case structures.CosmeticKindNametagPaint:
			paint := cos.ReadPaint()
			stops := make([]model.CosmeticPaintGradientStop, len(paint.Stops))
			dropShadows := make([]model.CosmeticPaintDropShadow, len(paint.DropShadows))
			for i, stop := range paint.Stops {
				stops[i] = model.CosmeticPaintGradientStop{
					At:    stop.At,
					Color: stop.Color,
				}
			}
			for i, shadow := range paint.DropShadows {
				dropShadows[i] = model.CosmeticPaintDropShadow{
					OffsetX: shadow.OffsetX,
					OffsetY: shadow.OffsetY,
					Radius:  shadow.Radius,
					Color:   shadow.Color,
				}
			}
			result.Paints = append(result.Paints, &model.CosmeticPaint{
				ID:          paint.ID.Hex(),
				Name:        cos.Name,
				Users:       usersToIdentifiers(cos.Users),
				Function:    string(paint.Function),
				Color:       paint.Color,
				Stops:       stops,
				Repeat:      paint.Repeat,
				Angle:       paint.Angle,
				Shape:       paint.Shape,
				ImageURL:    paint.ImageURL,
				DropShadows: dropShadows,
			})
		}
	}

	// Set Cache
	{
		j, err := json.Marshal(result)
		if err != nil {
			logrus.WithError(err).Error("failed to encode cache data for /v2/cosmetics")
		} else if err = r.Ctx.Inst().Redis.SetEX(ctx, cacheKey, j, time.Minute*10); err != nil {
			logrus.WithError(err).Error("failed to save cache of /v2/cosmetics")
		}
	}

	return ctx.JSON(rest.OK, result)
}

type aggregatedCosmeticsResult struct {
	Entitlements []*structures.Entitlement `bson:"entitlements"`
	Users        []*structures.User        `bson:"users"`
}
