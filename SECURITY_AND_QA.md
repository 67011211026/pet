# คู่มือความปลอดภัยและการเตรียมตัวตอบผู้ตรวจงาน (Security & Q&A Defense Guide)

เอกสารนี้รวบรวมหลักการออกแบบด้านความปลอดภัย (10 คะแนน) และแนวทางการตอบคำถามเชิงเทคนิค (5 คะแนน) สำหรับระบบ **Pet Hotel & Daycare Reservation System**

---

## 1. การปฏิบัติตามมาตรฐานความปลอดภัย (Security Standards - 10 คะแนน)

### 1.1 การป้องกัน SQL Injection
* **กลไกที่ใช้:** ใช้งาน **GORM / Parameterized Queries** ในทุกฟังก์ชันของชั้น Repository
* **หลักการทำงาน:** โค้ดจะไม่ใช้การต่อสตริง (`fmt.Sprintf("SELECT * FROM ... " + input)`) แต่จะส่ง Query Structure และ Parameters แยกออกจากกันผ่าน Placeholder `?` 
* **ผลลัพธ์:** แม้ผู้ไม่ประสงค์ดีจะกรอก `' OR '1'='1` หรือ SQL payload อื่นๆ ฐานข้อมูลจะตีความเป็นเพียงข้อความ String ธรรมดา ไม่สามารถ Execute เป็นคำสั่ง SQL ได้

### 1.2 การป้องกัน Broken Access Control / IDOR (Insecure Direct Object References)
* **กลไกที่ใช้:**
  1. ยืนยันตัวตนผ่าน JWT / Cookie Session และเก็บ `user_id` ไว้ใน Gin Context
  2. เมื่อผู้ใช้เรียกดู แก้ไข หรือลบข้อมูล (เช่น สัตว์เลี้ยง หรือการจอง) ระบบจะดึง `user_id` จาก Context มาเป็นเงื่อนไขร่วมเสมอ:
     ```go
     db.Where("id = ? AND user_id = ?", resourceID, authUserID).First(&pet)
     ```
  3. **การป้องกัน User A ขโมยจองห้องให้สัตว์เลี้ยงของ User B:** ใน Handler `CreateBooking` มีการตรวจสอบว่า `pet_id` ที่เลือกต้องมี `user_id == current_user_id` เสมอ
* **ผลลัพธ์:** ป้องกันไม่ให้ User A แอบเปลี่ยน URL Parameter หรือ Payload ID เพื่อแอบเข้าถึงหรือแก้ไขข้อมูลของ User B ได้ 100%

### 1.3 การป้องกัน XSS (Cross-Site Scripting)
* **กลไกที่ใช้:**
  1. ใช้ **Go `html/template`** ซึ่งมีกลไก **Context-Aware Auto-Escaping** ทำการแปลงอักขระพิเศษ เช่น `<`, `>`, `&`, `"`, `'` เป็น HTML Entities อัตโนมัติ
  2. กำหนด Security Response Headers ใน Middleware:
     * `X-Content-Type-Options: nosniff`
     * `X-Frame-Options: DENY` (ป้องกัน Clickjacking)
     * `X-XSS-Protection: 1; mode=block`

### 1.4 การป้องกัน CSRF และการรักษาความปลอดภัยของ Cookie
* **กลไกที่ใช้:**
  * เก็บ Authentication Token ในคุกกี้ที่เปิดใช้งาน:
    * `HttpOnly = true`: ป้องกันไม่ให้ JavaScript ฝั่ง Client เข้าถึง Token ได้ (ลดผลกระทบหากเกิด XSS)
    * `SameSite = Lax`: เบราว์เซอร์จะไม่ส่งคุกกี้นี้ไปกับ Cross-Site Request ที่เป็นอันตราย ป้องกันการโจมตีประเภท CSRF
    * `Secure = true` (ใน Production): ส่งผ่าน HTTPS เท่านั้น

### 1.5 การป้องกัน Brute Force Attack (Rate Limiting)
* **กลไกที่ใช้:**
  * ติดตั้ง **Token Bucket Algorithm** (`golang.org/x/time/rate`) ดักจับเฉพาะเส้นทาง `/login` และ `/register`
  * จำกัดไม่เกิน 10 requests ต่อนาทีต่อ IP และมี Burst ไม่เกิน 5 ครั้ง
  * หากเกินกำหนด เซิร์ฟเวอร์จะตัดการทำงานทันทีพร้อมส่งคืน **HTTP Status 429 (Too Many Requests)** และแนบ Header `Retry-After`

---

## 2. แนวทางการตอบคำถามผู้ตรวจงาน (Technical Q&A - 5 คะแนน)

### คำถามที่ 1: กลไกการทำงานของ Password Hashing ด้วย Bcrypt เป็นอย่างไร? ทำไมถึงปลอดภัยกว่า MD5 หรือ SHA-256?
> **แนวทางการตอบ:**
> 1. **การทำงาน:** Bcrypt ทำงานบนอัลกอริทึม Blowfish Cipher โดยขั้นตอนแรกจะทำการสร้าง **Salt ขนาด 16-byte แบบสุ่ม** รวมเข้ากับรหัสผ่าน จากนั้นทำการคำนวณซ้ำหลายรอบตามค่า **Work Factor (Cost)** ที่ตั้งไว้ (ในโปรเจกต์นี้ใช้ Default Cost = 10 หมายถึงวนลูป \(2^{10} = 1024\) รอบ)
> 2. **ความปลอดภัย:** 
>    * ไม่มีการเก็บรหัสผ่านจริง (One-way Hashing)
>    * Salt ช่วยป้องกันการโจมตีด้วย Rainbow Table (ตารางแฮชสำเร็จรูป)
>    * ต่างจาก MD5 หรือ SHA-256 ซึ่งเป็นแฮชที่คำนวณเร็วมาก (Fast Hash) ออกแบบมาเพื่อตรวจความถูกต้องของไฟล์ แต่ Bcrypt เป็น Slow Key Derivation Function ที่กินเวลา CPU เล็กน้อย (เช่น 100ms) ทำให้แฮกเกอร์ไม่สามารถใช้การ์ดจอ (GPU) ยิง Brute Force หลายล้านครั้งต่อวินาทีได้

---

### คำถามที่ 2: วิธีการตรวจสอบและป้องกันไม่ให้ User A เข้าถึงข้อมูลของ User B ในชั้นโค้ด Go ทำอย่างไร?
> **แนวทางการตอบ:**
> 1. ในชั้น **Auth Middleware** เมื่อถอดรหัส JWT Token ผ่าน จะนำ `user_id` เก็บลงใน `gin.Context` ด้วยคำสั่ง `c.Set("user_id", claims.UserID)`
> 2. ในชั้น **Handler** จะดึง `userID := c.GetUint("user_id")` ออกมาใช้งาน **ห้ามรับ user_id จาก Input Form หรือ URL Parameter เด็ดขาด**
> 3. ในชั้น **Repository / Database Query** จะใช้คำสั่ง SQL หรือ GORM ที่ผูกเงื่อนไขคู่กันเสมอ:
>    ```go
>    res := db.Where("id = ? AND user_id = ?", targetID, userID).Delete(&models.Pet{})
>    if res.RowsAffected == 0 {
>        // ไม่พบข้อมูลหรือไม่มีสิทธิ์ -> ส่งคืน 403 Forbidden หรือ 404 Not Found
>    }
>    ```
> 4. ทำให้ต่อให้ User A เดา ID ของ User B ได้ แต่เมื่อนำมาเทียบกับ Token แล้วไม่ตรง ฐานข้อมูลจะไม่ทำการเปลี่ยนแปลงใดๆ

---

### คำถามที่ 3: การแยก Routing และ Middleware ระหว่างสิทธิ์ User และ Admin ทำงานอย่างไร?
> **แนวทางการตอบ:**
> เราใช้ **Route Grouping** ของ Gin Framework:
> 1. **Public Routes (`/login`, `/register`):** มีเพียง Middleware `RateLimitLogin()` เพื่อจำกัดความถี่
> 2. **User Protected Routes (`/dashboard`, `/pets/*`, `/bookings/*`):** มี Middleware `AuthRequired()` ตรวจสอบว่ามี Token ที่ถูกต้องหรือไม่
> 3. **Admin Routes (`/admin`, `/admin/bookings/*`, `/admin/rooms/*`):** นอกจาก `AuthRequired()` แล้ว จะผ่าน Middleware พิเศษคือ `RequireRole(models.RoleAdmin)` เพื่อตรวจค่า Role ใน Token Claim หากไม่ใช่ `admin` จะส่งคืน `403 Forbidden` ทันที และตัดวงจรการทำงานด้วย `c.Abort()` ทำให้ไม่หลุดเข้าถึงข้อมูล Dashboard ของ Admin

---

### คำถามที่ 4: การส่งคืน HTTP Status Code ที่ถูกต้องตามสถานการณ์ มีหลักเกณฑ์อย่างไรบ้าง?
> **แนวทางการตอบ:**
> * **200 OK:** ดำเนินการสำเร็จและแสดงผลข้อมูล (เช่น แสดงหน้า Dashboard, ดึงรายชื่อห้อง)
> * **201 Created:** สร้าง Resource ใหม่ลงฐานข้อมูลสำเร็จ (เมื่อทำ REST API)
> * **303 See Other / 302 Found:** Redirect ผู้ใช้ไปยังหน้าถัดไปหลังจากส่ง Form สำเร็จ (เช่น เมื่อ Login ผ่านให้ redirect ไปหน้า Dashboard)
> * **400 Bad Request:** ข้อมูลที่ Client ส่งมาไม่ถูกต้องตามเงื่อนไข (เช่น วันที่ Check-out เกิดก่อน Check-in หรือกรอกฟอร์มไม่ครบ)
> * **401 Unauthorized:** ไม่ได้ส่ง Token มา หรือ Token หมดอายุ / ถอดรหัสไม่สำเร็จ
> * **403 Forbidden:** ยืนยันตัวตนผ่านแล้ว แต่ไม่มีสิทธิ์เข้าถึง Resource นั้น (เช่น User ทั่วไปพยายามเข้าถึง URL ของ Admin หรือพยายามลบสัตว์เลี้ยงของคนอื่น)
> * **404 Not Found:** ไม่พบหน้าที่ร้องขอ หรือไม่พบ Resource ในฐานข้อมูล
> * **409 Conflict:** เกิดความขัดแย้งของข้อมูล (เช่น สมัครสมาชิกด้วยอีเมลที่มีอยู่แล้วในระบบ หรือห้องพักถูกจองซ้อนทับ)
> * **429 Too Many Requests:** ร้องขอมาถี่เกินไป (โดนตัดการทำงานจาก Rate Limiter)
> * **500 Internal Server Error:** ข้อผิดพลาดฝั่งเซิร์ฟเวอร์ เช่น ไม่สามารถเชื่อมต่อฐานข้อมูลได้
