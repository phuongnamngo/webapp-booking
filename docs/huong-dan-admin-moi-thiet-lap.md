# Hướng dẫn admin mới: Office hours, Seat types và gán loại ghế

Tài liệu dành cho **Org Admin / Space Admin** lần đầu cấu hình tổ chức. Trong ứng dụng web, các màn hình admin thường có đường dẫn dạng `/ui/admin/...` (tùy cách triển khai, phần tiền tố `/ui/` có thể là root của UI).

---

## 1. Thuật ngữ trong hệ thống

| Thuật ngữ (UI) | Ý nghĩa |
|-----------------|--------|
| **Location** (Địa điểm) | Một “khu” làm việc có **bản đồ** (kích thước, múi giờ), chứa nhiều ghế. |
| **Space** (Ghế / ô) | Một chỗ ngồi trên bản đồ — user book theo ô này. |
| **Seat type** (Loại ghế) | Khuôn mẫu đặt chỗ: theo **khoảng thời gian linh hoạt** hoặc theo **slot cố định**; gán cho từng Space. |
| **Office hours** (Giờ làm việc) | Khung giờ trong ngày mà booking và trạng thái “còn chỗ trong ngày” được tính — cấu hình **theo tổ chức** tại trang Cài đặt. |

**Lưu ý:** Trong code và API thường gọi là `space_type`; trên giao diện admin có thể hiển thị là **Seat type** — cùng một khái niệm.

---

## 2. Cấu hình Office hours (giờ làm việc)

### Vào đâu?

1. Đăng nhập tài khoản có quyền quản trị.
2. Mở menu admin → **Settings** (Cài đặt / Thiết lập tổ chức).  
   - Đường dẫn trang: **`/admin/settings`** (trên UI đầy đủ thường là `/ui/admin/settings`).

### Làm gì trên màn hình?

1. Tìm mục **Office hours** (Giờ làm việc).
2. Chọn **giờ bắt đầu** và **giờ kết thúc** trong ngày (ô nhập kiểu thời gian, ví dụ 08:00 → 17:00).
3. Cuộn xuống cuối form và bấm **Save** (Lưu) để ghi nhận cả Office hours và các thiết lập tổ chức khác trên cùng trang.

**Ý nghĩa:** Hệ thống dùng khoảng thời gian này để giới hạn booking và tính trạng thái “còn chỗ / hết chỗ trong ngày”. Gợi ý trên giao diện (tiếng Anh): *“Bookings and seat status are limited to this daily time range.”*

### Cài đặt liên quan nên biết (cùng trang Settings)

- **Minimum booking duration (hours)** — thời lượng tối thiểu **theo giờ** cho tổ chức.  
  - Ghế **chưa gán** Seat type: nếu giá trị này **0** hoặc không dùng, hệ thống có thể áp **fallback 30 phút** cho logic “còn book được / tối thiểu khi đặt” (đồng bộ với tài liệu [Hướng dẫn đặt chỗ và trạng thái ngày](./huong-dan-dat-cho-va-trang-thai-ngay.md)).
- **Daily basis booking**, **Max days in advance**, v.v. — ảnh hưởng cách user chọn ngày và độ dài booking; admin nên đọc nhãn từng ô và lưu sau khi chỉnh.

### Báo cáo booking hằng ngày (Daily booking report)

Trên cùng trang **Settings** (`/admin/settings`):

1. Bật **Enable daily booking report**.
2. Nhập **Report recipients** — danh sách email, phân tách bằng dấu phẩy (ví dụ: `ops@company.com, manager@company.com`).
3. Chọn **Report send time** — giờ gửi mỗi ngày theo múi giờ `Asia/Ho_Chi_Minh` (mặc định `08:00` nếu chưa đổi).
4. Bấm **Save** ở cuối form.
5. Dùng **Preview report** hoặc **Send test email** để kiểm tra nội dung **ngày hôm qua** (không cần cấu hình biến môi trường Docker/`server/.env` cho tính năng này).

---

## 3. Seat types (Loại ghế): tạo và chỉnh sửa

### Danh sách loại ghế

- Menu admin → mục quản lý **Seat types** (Loại ghế / Seat types).  
- Đường dẫn: **`/admin/seat-types/`**.

Trên bảng danh sách bạn thấy: **Tên**, **Chế độ đặt** (Booking mode), **Bật** (Enabled).

### Tạo loại ghế mới

1. Tại trang danh sách Seat types, bấm **Add** (Thêm).
2. Đi tới form tạo/sửa: **`/admin/seat-types/add`** (sau khi lưu lần đầu, hệ thống thường chuyển sang trang chi tiết có ID).

### Các trường quan trọng trên form

| Trường | Gợi ý sử dụng |
|--------|----------------|
| **Name** | Tên dễ hiểu (ví dụ: “Bàn linh hoạt 30p”, “Phòng họp slot sáng”). |
| **Booking mode** | **Flexible time** — user chọn enter/leave; cần **Minimum duration (minutes)**. **Fixed slots** — user chọn một trong các **time slot** đã định nghĩa (nhãn, giờ bắt đầu/kết thúc, bật/tắt, thứ tự). |
| **Minimum duration (minutes)** | Chủ yếu cho chế độ **Flexible time** (tối thiểu bao nhiêu phút mỗi lần đặt). |
| **Enabled** | Tắt nếu tạm không dùng loại này; ghế đang gán loại bị tắt có thể rơi về luồng “legacy” tùy cấu hình — nên kiểm tra sau khi tắt. |

### Time slots (chỉ khi chọn Fixed slots)

1. Chọn **Booking mode** = Fixed slots.
2. Phần **Time slots**: thêm từng dòng — **Tên**, **Start time**, **End time**, **Enabled**, **Sort order**.
3. Slot phải nằm trong **Office hours** khi user đặt; nên thiết kế slot khớp giờ làm việc (ví dụ 08:00–12:00, 13:00–17:00).

4. Bấm **Save** để lưu loại ghế và các slot.

### Sửa / xóa

- Từ danh sách, bấm vào một dòng để mở **`/admin/seat-types/{id}`**.
- Có nút **Delete** (nếu được phép) và **Save** như form chỉnh sửa.

---

## 4. Gán Seat type cho từng ghế trên “khu” (Location / bản đồ)

Trong sản phẩm này, **“khu” hoặc area** mà user hay nói tương ứng với **Location** (địa điểm có bản đồ), không phải một thực thể tên “Area” riêng trong menu.

### Bước 4.1 — Mở địa điểm

1. Menu admin → **Locations** (Địa điểm). Đường dẫn: **`/admin/locations/`**.
2. Chọn (hoặc tạo mới) một location để vào **`/admin/locations/{id}`**.

### Bước 4.2 — Chọn ghế (Space) cần gán loại

Có hai cách thường dùng:

1. **Trên bản đồ:** bấm vào ô ghế (khối RND) để chọn — ô được chọn thường có viền/trạng thái “selected”.
2. **Trong bảng danh sách ghế:** bấm vào **một dòng** tương ứng ghế — hệ thống mở **cửa sổ chỉnh sửa chi tiết ghế** (modal).

### Bước 4.3 — Chọn Seat type và lưu

1. Trong modal chi tiết ghế, tìm dropdown **Seat type** (nhãn có thể là “Seat type” / “Loại ghế”).
2. Chọn một loại trong danh sách, hoặc chọn **None / Trống** nếu muốn ghế đó dùng luồng **chọn khoảng thời gian cổ điển** (không theo slot / không theo min duration của loại ghế — vẫn chịu Office hours và rule tổ chức).
3. Bấm **Save** trên trang location để ghi toàn bộ thay đổi bản đồ và danh sách ghế xuống server.

**Mẹo:** Sau khi lưu, mỗi ghế đã có ID sẽ có **link đặt chỗ** (booking link) hiển thị trong bảng — có thể copy gửi user hoặc kiểm thử.

---

## 5. Thứ tự gợi ý khi triển khai môi trường mới

1. **Settings** → Office hours + timezone mặc định (nếu có trên cùng trang) + min booking duration theo nhu cầu.  
2. **Seat types** → tạo đủ các loại (linh hoạt / slot) và bật **Enabled**.  
3. **Locations** → vẽ/chỉnh ghế trên map → gán **Seat type** từng ghế → **Save**.  
4. Đăng nhập user thử **Search** theo location và book thử một vài kịch bản (slot, linh hoạt, ghế không loại).

---

## 6. Liên kết nhanh (đường dẫn trong app)

| Nội dung | Đường dẫn |
|----------|-----------|
| Cài đặt tổ chức + Office hours | `/admin/settings` |
| Danh sách Seat types | `/admin/seat-types/` |
| Thêm Seat type | `/admin/seat-types/add` |
| Sửa Seat type | `/admin/seat-types/{id}` |
| Danh sách Locations | `/admin/locations/` |
| Sửa một Location (bản đồ + ghế) | `/admin/locations/{id}` |

---
