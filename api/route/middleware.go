package route

import (
	"crypto/rsa"
	"errors"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/caitunai/dotconfig/api/base"
	"github.com/caitunai/dotconfig/db"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	jwtSecret             []byte
	publicKey             *rsa.PublicKey
	oauthCallbackPath     = "oauth/path/to/callback"
	configCenterIsEnabled bool
	configCenterAuth      gin.HandlerFunc
)

const unauthorizedMessage = "unauthorized"

func InitMiddleware() {
	configureConfigCenterAccess(
		viper.GetBool("configcenter.enabled"),
		viper.GetString("configcenter.username"),
		viper.GetString("configcenter.password"),
	)
	jwtSecret = []byte(viper.GetString("auth.api.secret"))
	publicKeyByte, err := os.ReadFile(viper.GetString("oauth.publicKeyPath"))
	if err != nil {
		log.Error().Err(err).Msg("read oauth public key failed")
		return
	}
	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(publicKeyByte)
	if err != nil {
		log.Error().Err(err).Msg("parse oauth public key failed")
		return
	}
}

func configureConfigCenterAccess(enabled bool, username, password string) {
	configCenterIsEnabled = enabled
	configCenterAuth = nil
	if enabled {
		configCenterAuth = gin.BasicAuth(gin.Accounts{username: password})
	}
}

func configCenterEnabled(c *base.Context) {
	if !configCenterIsEnabled {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Next()
}

func configCenterNoStore(c *base.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Vary", "Authorization")
	c.Next()
}

// This middleware can be used to verify the login status of real users.
func authorized(c *base.Context) {
	uid := c.GetUint("uid")
	if uid == 0 {
		login(c)
		c.Abort()
		return
	}

	c.Next()
}

// This middleware can be used to verify the authorization status of a JWT token for an API.
func apiAuthorized(c *base.Context) {
	apiUser := c.GetAPIUser()
	if apiUser == nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			base.KeyStatus:  http.StatusForbidden,
			base.KeyMessage: unauthorizedMessage,
			base.KeyData:    gin.H{},
		})
		return
	}
	c.Next()
}

func AttemptAuth() base.HandlerFunc {
	return func(c *base.Context) {
		if isConfigCenterPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		var uid uint64
		id, _ := c.DecodeCookie("session_id")
		if id != "" {
			uid, _ = strconv.ParseUint(id, 10, 64)
		}
		if uid == 0 {
			bearerToken := getBearerToken(c.GetHeader("Authorization"))
			if bearerToken != "" {
				var accountID uint64
				token, err := jwt.Parse(bearerToken, func(token *jwt.Token) (any, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
						return jwtSecret, nil
					}
					if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
						return nil, errors.New("sign method error")
					}
					if publicKey == nil {
						return nil, errors.New("jwt public key not found")
					}
					return publicKey, nil
				})
				if err != nil {
					log.Error().Err(err).Msg("parse bearer token error")
					accountID = 0
				} else {
					sub, err := token.Claims.GetSubject()
					if err != nil {
						log.Error().Err(err).Msg("get token id error")
					}
					accountID, err = strconv.ParseUint(sub, 10, 64)
					if err != nil {
						audiences, err := token.Claims.GetAudience()
						if err == nil && slices.Contains(audiences, viper.GetString("auth.api.audience")) {
							issuer, err := token.Claims.GetIssuer()
							if err == nil && slices.Contains(viper.GetStringSlice("auth.api.issuers"), issuer) {
								c.SetAPIUser(&base.APIUser{
									User:   sub,
									Issuer: issuer,
								})
							}
						}
					}
				}
				if accountID > 0 {
					if c.IsDatabaseEnabled() {
						u, err := db.RegisterUser(c.Request.Context(), uint(accountID))
						if err != nil {
							uid = 0
						} else if u != nil {
							c.SetUser(u)
							uid = uint64(u.ID)
						}
					} else {
						uid = accountID
					}
				}
			}
		}
		if uid == 0 && !c.IsWechatMiniProgram() {
			// is WeChat H5 but not WeChat program
			ag := strings.ToLower(c.GetHeader("user-agent"))
			isWechat := strings.Contains(ag, "micromessenger")
			isCallback := strings.Contains(c.Request.URL.Path, oauthCallbackPath)
			if isWechat && !isCallback {
				login(c)
				c.Abort()
				return
			}
		}

		c.Set("uid", uint(uid))
		c.Next()
	}
}

func isConfigCenterPath(path string) bool {
	return path == "/config-center" || strings.HasPrefix(path, "/config-center/")
}

func getBearerToken(authorization string) string {
	scheme, token, ok := strings.Cut(strings.TrimSpace(authorization), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func AuthorizedAllowSpider() base.HandlerFunc {
	return func(c *base.Context) {
		ag := strings.ToLower(c.GetHeader("user-agent"))
		if strings.Contains(ag, "twitterbot") ||
			c.Request.Method == "GET" ||
			c.Request.Method == "HEAD" ||
			c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		// 处理登陆逻辑
		authorized(c)
	}
}

func login(c *base.Context) {
	c.Unauthorized("you should login", gin.H{
		"result": "you are not implement the login function",
	})
}
