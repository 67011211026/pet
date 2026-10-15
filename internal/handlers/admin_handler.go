package handlers

import (
	"net/http"
	"strconv"

	"pet-hotel/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// ShowAdminDashboard displays all bookings, room statuses, and metrics
func (h *AdminHandler) ShowAdminDashboard(c *gin.Context) {
	var bookings []models.Booking
	h.db.Preload("User").Preload("Pet").Preload("Room").
		Order("id DESC").
		Find(&bookings)

	var rooms []models.Room
	h.db.Order("id ASC").Find(&rooms)

	// Summary statistics
	var pendingCount, approvedCount, totalPets, totalUsers int64
	h.db.Model(&models.Booking{}).Where("status = ?", models.StatusPending).Count(&pendingCount)
	h.db.Model(&models.Booking{}).Where("status = ?", models.StatusApproved).Count(&approvedCount)
	h.db.Model(&models.Pet{}).Count(&totalPets)
	h.db.Model(&models.User{}).Where("role = ?", models.RoleUser).Count(&totalUsers)

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"Title":         "Admin Dashboard - Pet Hotel",
		"UserName":      c.GetString("user_name"),
		"UserEmail":     c.GetString("user_email"),
		"UserRole":      c.GetString("user_role"),
		"Bookings":      bookings,
		"Rooms":         rooms,
		"PendingCount":  pendingCount,
		"ApprovedCount": approvedCount,
		"TotalPets":     totalPets,
		"TotalUsers":    totalUsers,
		"Updated":       c.Query("updated") == "true",
	})
}

// UpdateBookingStatus changes booking status (approve/reject/completed)
func (h *AdminHandler) UpdateBookingStatus(c *gin.Context) {
	bookingIDParam := c.Param("id")
	bookingID, err := strconv.ParseUint(bookingIDParam, 10, 64)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}

	status := c.PostForm("status")
	validStatus := map[models.BookingStatus]bool{
		models.StatusPending:   true,
		models.StatusApproved:  true,
		models.StatusRejected:  true,
		models.StatusCompleted: true,
	}

	if !validStatus[models.BookingStatus(status)] {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"Code":    400,
			"Message": "สถานะการจองไม่ถูกต้อง",
		})
		return
	}

	h.db.Model(&models.Booking{}).Where("id = ?", bookingID).Update("status", status)

	c.Redirect(http.StatusSeeOther, "/admin?updated=true")
}

// ToggleRoomStatus toggles room status between available and maintenance
func (h *AdminHandler) ToggleRoomStatus(c *gin.Context) {
	roomIDParam := c.Param("id")
	roomID, err := strconv.ParseUint(roomIDParam, 10, 64)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}

	var room models.Room
	if err := h.db.First(&room, roomID).Error; err != nil {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}

	newStatus := models.RoomStatusAvailable
	if room.Status == models.RoomStatusAvailable {
		newStatus = models.RoomStatusMaintenance
	}

	h.db.Model(&room).Update("status", newStatus)

	c.Redirect(http.StatusSeeOther, "/admin?updated=true")
}
