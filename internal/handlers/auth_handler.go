package handlers

import (
	"net/http"
	"strings"

	"pet-hotel/internal/config"
	"pet-hotel/internal/middleware"
	"pet-hotel/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

// ShowLoginPage renders the login HTML template
func (h *AuthHandler) ShowLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"Title": "เข้าสู่ระบบ - Pet Hotel",
	})
}

// ShowRegisterPage renders the register HTML template
func (h *AuthHandler) ShowRegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"Title": "สมัครสมาชิก - Pet Hotel",
	})
}

type RegisterInput struct {
	Name     string `form:"name" json:"name" binding:"required"`
	Email    string `form:"email" json:"email" binding:"required,email"`
	Password string `form:"password" json:"password" binding:"required,min=6"`
}

// Register creates a new user with Bcrypt password hashing
func (h *AuthHandler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBind(&input); err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Error": "กรุณากรอกข้อมูลให้ครบถ้วน รหัสผ่านต้องมีความยาวอย่างน้อย 6 ตัวอักษร",
		})
		return
	}

	// Check if email already exists
	var existing models.User
	if err := h.db.Where("email = ?", strings.ToLower(input.Email)).First(&existing).Error; err == nil {
		c.HTML(http.StatusConflict, "register.html", gin.H{
			"Error": "อีเมลนี้ถูกใช้งานในระบบแล้ว กรุณาใช้อีเมลอื่น",
		})
		return
	}

	// Bcrypt password hashing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"Error": "เกิดข้อผิดพลาดในการเข้ารหัสรหัสผ่าน",
		})
		return
	}

	newUser := models.User{
		Name:         input.Name,
		Email:        strings.ToLower(input.Email),
		PasswordHash: string(hashedPassword),
		Role:         models.RoleUser,
	}

	if err := h.db.Create(&newUser).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"Error": "ไม่สามารถบันทึกข้อมูลได้ กรุณาลองใหม่อีกครั้ง",
		})
		return
	}

	// Redirect to login page with success message
	c.Redirect(http.StatusSeeOther, "/login?registered=true")
}

type LoginInput struct {
	Email    string `form:"email" json:"email" binding:"required,email"`
	Password string `form:"password" json:"password" binding:"required"`
}

// Login validates user credentials and issues a secure JWT cookie
func (h *AuthHandler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBind(&input); err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"Error": "กรุณากรอกอีเมลและรหัสผ่านให้ถูกต้อง",
		})
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", strings.ToLower(input.Email)).First(&user).Error; err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Error": "อีเมลหรือรหัสผ่านไม่ถูกต้อง",
		})
		return
	}

	// Verify Bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Error": "อีเมลหรือรหัสผ่านไม่ถูกต้อง",
		})
		return
	}

	// Generate JWT
	token, err := middleware.GenerateJWT(&user, h.cfg)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"Error": "เกิดข้อผิดพลาดในการสร้างเซสชัน",
		})
		return
	}

	// Set HttpOnly, SameSite cookie (Secure: true in prod)
	isProd := h.cfg.Environment == "release"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", token, 86400, "/", "", isProd, true)

	// Direct user according to role
	if user.Role == models.RoleAdmin {
		c.Redirect(http.StatusSeeOther, "/admin")
	} else {
		c.Redirect(http.StatusSeeOther, "/dashboard")
	}
}

// Logout clears the session cookie
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/login")
}
