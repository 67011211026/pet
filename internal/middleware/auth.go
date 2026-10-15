package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"pet-hotel/internal/config"
	"pet-hotel/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint        `json:"user_id"`
	Email  string      `json:"email"`
	Name   string      `json:"name"`
	Role   models.Role `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT token for a user
func GenerateJWT(user *models.User, cfg *config.Config) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// AuthRequired verifies JWT from Cookie or Authorization header
func AuthRequired(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""

		// 1. Try reading from HttpOnly Cookie first (Browser navigation)
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			tokenString = cookie
		}

		// 2. Or from Authorization header (API / Mobile client)
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			respondUnauthorized(c)
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			// Clear invalid cookie
			c.SetCookie("auth_token", "", -1, "/", "", false, true)
			respondUnauthorized(c)
			return
		}

		// Save user details into Gin Context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// RequireRole checks that the user has at least one of the required roles
func RequireRole(allowedRoles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		userRole := val.(models.Role)
		for _, role := range allowedRoles {
			if userRole == role {
				c.Next()
				return
			}
		}

		// If user doesn't have sufficient role
		if strings.Contains(c.GetHeader("Accept"), "text/html") {
			c.HTML(http.StatusForbidden, "error.html", gin.H{
				"Code":    403,
				"Message": "Forbidden: You do not have permission to access this page.",
			})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
		}
		c.Abort()
	}
}

func respondUnauthorized(c *gin.Context) {
	if strings.Contains(c.GetHeader("Accept"), "text/html") {
		c.Redirect(http.StatusSeeOther, "/login")
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: valid token required"})
	}
	c.Abort()
}
