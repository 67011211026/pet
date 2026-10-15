package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"pet-hotel/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PetHandler struct {
	db *gorm.DB
}

func NewPetHandler(db *gorm.DB) *PetHandler {
	return &PetHandler{db: db}
}

// ShowPetForm renders Form 1: Pet Registration Form
func (h *PetHandler) ShowPetForm(c *gin.Context) {
	c.HTML(http.StatusOK, "pet_form.html", gin.H{
		"Title":    "ลงทะเบียนสัตว์เลี้ยง - Pet Hotel",
		"UserName": c.GetString("user_name"),
		"UserRole": c.GetString("user_role"),
		"IsEdit":   false,
	})
}

type CreatePetInput struct {
	Name           string `form:"name" binding:"required"`
	PetType        string `form:"pet_type" binding:"required"`
	Breed          string `form:"breed"`
	AllergiesNotes string `form:"allergies_notes"`
}

// CreatePet handles Form 1 submission
func (h *PetHandler) CreatePet(c *gin.Context) {
	userID := c.GetUint("user_id")

	var input CreatePetInput
	if err := c.ShouldBind(&input); err != nil {
		c.HTML(http.StatusBadRequest, "pet_form.html", gin.H{
			"Error":    "กรุณากรอกชื่อและประเภทสัตว์เลี้ยงให้ครบถ้วน",
			"UserName": c.GetString("user_name"),
			"UserRole": c.GetString("user_role"),
			"IsEdit":   false,
		})
		return
	}

	pet := models.Pet{
		UserID:         userID, // Securely bound to current authenticated user
		Name:           strings.TrimSpace(input.Name),
		PetType:        input.PetType,
		Breed:          strings.TrimSpace(input.Breed),
		AllergiesNotes: strings.TrimSpace(input.AllergiesNotes),
	}

	if err := h.db.Create(&pet).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "pet_form.html", gin.H{
			"Error":    "เกิดข้อผิดพลาดในการบันทึกข้อมูลสัตว์เลี้ยง",
			"UserName": c.GetString("user_name"),
			"IsEdit":   false,
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/dashboard?pet_added=true")
}

// ShowEditPetForm renders Form 1 in edit mode with strict Ownership Check (IDOR Prevention)
func (h *PetHandler) ShowEditPetForm(c *gin.Context) {
	userID := c.GetUint("user_id")
	petIDParam := c.Param("id")
	petID, err := strconv.ParseUint(petIDParam, 10, 64)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}

	var pet models.Pet
	// CRITICAL IDOR PREVENTION: Enforce both pet ID AND user ID match
	result := h.db.Where("id = ? AND user_id = ?", petID, userID).First(&pet)
	if result.Error != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"Code":    404,
			"Message": "ไม่พบข้อมูลสัตว์เลี้ยง หรือคุณไม่มีสิทธิ์เข้าถึงข้อมูลของสัตว์เลี้ยงตัวนี้ (IDOR Protected)",
		})
		return
	}

	c.HTML(http.StatusOK, "pet_form.html", gin.H{
		"Title":    "แก้ไขข้อมูลสัตว์เลี้ยง",
		"UserName": c.GetString("user_name"),
		"UserRole": c.GetString("user_role"),
		"Pet":      pet,
		"IsEdit":   true,
	})
}

// UpdatePet updates pet data with strict Ownership Check (IDOR Prevention)
func (h *PetHandler) UpdatePet(c *gin.Context) {
	userID := c.GetUint("user_id")
	petIDParam := c.Param("id")
	petID, err := strconv.ParseUint(petIDParam, 10, 64)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}

	var input CreatePetInput
	if err := c.ShouldBind(&input); err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"Code":    400,
			"Message": "ข้อมูลไม่ถูกต้อง กรุณากรอกข้อมูลให้ครบถ้วน",
		})
		return
	}

	// CRITICAL IDOR PREVENTION: Updates only if ID and user_id match
	res := h.db.Model(&models.Pet{}).
		Where("id = ? AND user_id = ?", petID, userID).
		Updates(map[string]interface{}{
			"name":            strings.TrimSpace(input.Name),
			"pet_type":        input.PetType,
			"breed":           strings.TrimSpace(input.Breed),
			"allergies_notes": strings.TrimSpace(input.AllergiesNotes),
		})

	if res.Error != nil || res.RowsAffected == 0 {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"Code":    403,
			"Message": "ไม่สามารถแก้ไขได้ เนื่องจากคุณไม่ได้เป็นเจ้าของสัตว์เลี้ยงตัวนี้ (IDOR Protected)",
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/dashboard?pet_updated=true")
}

// DeletePet deletes pet data with strict Ownership Check (IDOR Prevention)
func (h *PetHandler) DeletePet(c *gin.Context) {
	userID := c.GetUint("user_id")
	petIDParam := c.Param("id")
	petID, err := strconv.ParseUint(petIDParam, 10, 64)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}

	// CRITICAL IDOR PREVENTION: Deletes only if ID and user_id match
	res := h.db.Where("id = ? AND user_id = ?", petID, userID).Delete(&models.Pet{})
	if res.Error != nil || res.RowsAffected == 0 {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"Code":    403,
			"Message": "ไม่สามารถลบได้ เนื่องจากคุณไม่ได้เป็นเจ้าของสัตว์เลี้ยงตัวนี้ (IDOR Protected)",
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/dashboard?pet_deleted=true")
}
