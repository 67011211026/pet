package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"pet-hotel/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BookingHandler struct {
	db *gorm.DB
}

func NewBookingHandler(db *gorm.DB) *BookingHandler {
	return &BookingHandler{db: db}
}

// ShowDashboard displays user's pets and their reservation history
func (h *BookingHandler) ShowDashboard(c *gin.Context) {
	userID := c.GetUint("user_id")

	var pets []models.Pet
	h.db.Where("user_id = ?", userID).Order("id DESC").Find(&pets)

	var bookings []models.Booking
	h.db.Preload("Pet").Preload("Room").
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&bookings)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"Title":       "หน้าแดชบอร์ด - Pet Hotel",
		"UserName":    c.GetString("user_name"),
		"UserEmail":   c.GetString("user_email"),
		"UserRole":    c.GetString("user_role"),
		"Pets":        pets,
		"Bookings":    bookings,
		"PetAdded":    c.Query("pet_added") == "true",
		"PetUpdated":  c.Query("pet_updated") == "true",
		"PetDeleted":  c.Query("pet_deleted") == "true",
		"BookSuccess": c.Query("booked") == "true",
	})
}

// ShowBookingForm renders Form 2: Room Reservation Form
func (h *BookingHandler) ShowBookingForm(c *gin.Context) {
	userID := c.GetUint("user_id")

	var pets []models.Pet
	h.db.Where("user_id = ?", userID).Find(&pets)

	var rooms []models.Room
	h.db.Where("status = ?", models.RoomStatusAvailable).Find(&rooms)

	c.HTML(http.StatusOK, "booking_form.html", gin.H{
		"Title":    "จองห้องฝากเลี้ยง - Pet Hotel",
		"UserName": c.GetString("user_name"),
		"UserRole": c.GetString("user_role"),
		"Pets":     pets,
		"Rooms":    rooms,
	})
}

type CreateBookingInput struct {
	PetID        uint   `form:"pet_id" binding:"required"`
	RoomID       uint   `form:"room_id" binding:"required"`
	CheckInDate  string `form:"check_in_date" binding:"required"`
	CheckOutDate string `form:"check_out_date" binding:"required"`
}

// CreateBooking processes Form 2 with ownership verification and room overlap prevention
func (h *BookingHandler) CreateBooking(c *gin.Context) {
	userID := c.GetUint("user_id")

	var input CreateBookingInput
	if err := c.ShouldBind(&input); err != nil {
		h.renderBookingError(c, "กรุณากรอกข้อมูลการจองให้ครบถ้วน")
		return
	}

	// 1. Verify Pet Ownership (IDOR Prevention: User cannot book for another user's pet)
	var pet models.Pet
	if err := h.db.Where("id = ? AND user_id = ?", input.PetID, userID).First(&pet).Error; err != nil {
		h.renderBookingError(c, "ไม่พบข้อมูลสัตว์เลี้ยงของคุณ กรุณาเลือกสัตว์เลี้ยงที่คุณเป็นเจ้าของ")
		return
	}

	// 2. Parse and Validate Dates
	checkIn, err := time.Parse("2006-01-02", input.CheckInDate)
	if err != nil {
		h.renderBookingError(c, "รูปแบบวันที่เข้าพักไม่ถูกต้อง")
		return
	}
	checkOut, err := time.Parse("2006-01-02", input.CheckOutDate)
	if err != nil {
		h.renderBookingError(c, "รูปแบบวันที่ออกไม่ถูกต้อง")
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	if checkIn.Before(today) {
		h.renderBookingError(c, "วันที่เข้าพักต้องไม่เป็นวันในอดีต")
		return
	}

	if !checkOut.After(checkIn) {
		h.renderBookingError(c, "วันที่ออกต้องเกิดขึ้นหลังวันที่เข้าพักอย่างน้อย 1 วัน")
		return
	}

	// 3. Verify Room Availability & Status
	var room models.Room
	if err := h.db.First(&room, input.RoomID).Error; err != nil || room.Status != models.RoomStatusAvailable {
		h.renderBookingError(c, "ห้องพักที่เลือกไม่พร้อมให้บริการในขณะนี้")
		return
	}

	// 4. Overlap Check: Ensure room is not already booked during these dates
	var conflictCount int64
	h.db.Model(&models.Booking{}).
		Where("room_id = ? AND status IN (?, ?)", room.ID, models.StatusPending, models.StatusApproved).
		Where("check_in_date < ? AND check_out_date > ?", checkOut, checkIn).
		Count(&conflictCount)

	if conflictCount > 0 {
		h.renderBookingError(c, fmt.Sprintf("ห้อง %s มีการจองในช่วงวันดังกล่าวแล้ว กรุณาเลือกวันอื่นหรือห้องอื่น", room.RoomNumber))
		return
	}

	// 5. Calculate Total Price
	days := math.Ceil(checkOut.Sub(checkIn).Hours() / 24)
	if days < 1 {
		days = 1
	}
	totalPrice := days * room.PricePerNight

	// 6. Save Booking
	booking := models.Booking{
		UserID:       userID,
		PetID:        pet.ID,
		RoomID:       room.ID,
		CheckInDate:  checkIn,
		CheckOutDate: checkOut,
		Status:       models.StatusPending,
		TotalPrice:   totalPrice,
	}

	if err := h.db.Create(&booking).Error; err != nil {
		h.renderBookingError(c, "เกิดข้อผิดพลาดในการบันทึกการจอง กรุณาลองใหม่อีกครั้ง")
		return
	}

	c.Redirect(http.StatusSeeOther, "/dashboard?booked=true")
}

// CancelBooking allows a user to cancel their own pending booking (IDOR check)
func (h *BookingHandler) CancelBooking(c *gin.Context) {
	userID := c.GetUint("user_id")
	bookingIDParam := c.Param("id")
	bookingID, err := strconv.ParseUint(bookingIDParam, 10, 64)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}

	// Ownership check: must match both ID and user_id
	res := h.db.Model(&models.Booking{}).
		Where("id = ? AND user_id = ? AND status = ?", bookingID, userID, models.StatusPending).
		Update("status", models.StatusRejected)

	if res.Error != nil || res.RowsAffected == 0 {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"Code":    403,
			"Message": "ไม่สามารถยกเลิกการจองนี้ได้ (อาจไม่ใช่การจองของคุณหรือได้รับการอนุมัติแล้ว)",
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/dashboard")
}

func (h *BookingHandler) renderBookingError(c *gin.Context, msg string) {
	userID := c.GetUint("user_id")
	var pets []models.Pet
	h.db.Where("user_id = ?", userID).Find(&pets)
	var rooms []models.Room
	h.db.Where("status = ?", models.RoomStatusAvailable).Find(&rooms)

	c.HTML(http.StatusBadRequest, "booking_form.html", gin.H{
		"Title":    "จองห้องฝากเลี้ยง - Pet Hotel",
		"UserName": c.GetString("user_name"),
		"UserRole": c.GetString("user_role"),
		"Pets":     pets,
		"Rooms":    rooms,
		"Error":    msg,
	})
}
