# Hướng dẫn: Đặt chỗ, hủy booking và trạng thái ghế trong ngày

Tài liệu này mô tả cách hệ thống xử lý **hủy booking**, **trạng thái ghế trong ngày** (available / đã book một phần / hết chỗ) và **thời lượng tối thiểu** cho ghế chưa gán loại ghế (seat type), sau các cập nhật gần đây.

**Admin mới:** cách cấu hình Office hours, Seat types và gán loại ghế lên từng ô trên bản đồ địa điểm — xem [Hướng dẫn admin: thiết lập ban đầu](./huong-dan-admin-moi-thiet-lap.md).

---

## 1. Hủy booking sau khi đã tới giờ bắt đầu

### Người dùng thường

- Sau khi booking **đã tới hoặc đã qua giờ bắt đầu** (theo múi giờ của địa điểm), bạn **không thể tự hủy** booking đó nữa.
- Nếu cố hủy, ứng dụng sẽ báo lỗi (thông điệp kiểu: không thể hủy vì booking đã bắt đầu).

### Quản trị (Space Admin / Org Admin)

- Admin **vẫn có thể hủy** booking đã bắt đầu khi cần xử lý vận hành (ví dụ giải phóng chỗ, sửa lỗi đặt nhầm), trong phạm vi quyền hiện có của hệ thống.

### Giới hạn “hủy trước X giờ” (cài đặt tổ chức)

- Với booking **chưa** tới giờ bắt đầu, quy tắc **tối thiểu bao nhiêu giờ trước giờ vào** mới được hủy (nếu tổ chức bật) **vẫn áp dụng** như trước cho người dùng thường.

**Tóm lại:** “Đã tới giờ vào” là ranh chắn cho user thường; “X giờ trước khi vào” là ranh bổ sung cho booking **chưa** bắt đầu.

---

## 2. Trạng thái ghế trong ngày: available, đã book một phần, hết chỗ

Hệ thống xét **giờ làm việc** (Office hours), **thời điểm hiện tại** và **còn đủ khoảng thời gian để đặt một booking hợp lệ hay không**.

### “Hết chỗ” (full) khi nào?

Ghế được coi là **hết chỗ trong ngày** khi **ít nhất một** trong các trường hợp sau đúng:

- Đã book kín phần thời gian trong giờ làm việc, **hoặc**
- Còn lại trong ngày **không còn** khoảng thời gian nào đủ điều kiện để user đặt thêm (ví dụ chỉ còn 10 phút nhưng tối thiểu phải 30 phút).

### “Đã book một phần” (partially booked)

- Có booking trong ngày **và** vẫn còn **ít nhất một** lựa chọn đặt hợp lệ (slot hoặc khoảng thời gian) trong phần thời gian còn lại.

### Cùng ngày với “bây giờ”

- Với **ngày đang chọn là hôm nay**, hệ thống chỉ tính phần thời gian **từ lúc hiện tại** đến hết giờ làm việc (không còn coi là “còn book được” các khung đã qua trong ngày).

Nhờ vậy, ví dụ office 8h–17h, bây giờ 14h50, chỉ còn 10 phút tới 17h mà tối thiểu 30 phút thì ghế sẽ hiển thị **hết chỗ** thay vì “còn book một phần” gây hiểu nhầm.

---

## 3. Ghế chưa gán loại ghế (Seat type): thời lượng tối thiểu

Với ghế **chưa có** seat type (luồng chọn khoảng thời gian enter/leave như cũ):

- Nếu tổ chức đặt **thời lượng tối thiểu** (giờ) **lớn hơn 0**, hệ thống dùng đúng giá trị đó.
- Nếu **không đặt** hoặc đặt **bằng 0**, hệ thống dùng **mặc định 30 phút** làm mốc so sánh (đặt chỗ và tính “còn chỗ trong ngày” thống nhất).

**Gợi ý cho admin:** Nếu muốn cho phép booking ngắn hơn 30 phút cho loại ghế này, cần cấu hình **min booking duration** của tổ chức phù hợp (ví dụ 0,25 giờ = 15 phút nếu hệ thống hỗ trợ theo giờ), hoặc gán seat type phù hợp cho ghế.

---

## 4. Giao diện tìm kiếm / danh sách

- Màu và nhãn trạng thái trong ngày bám theo logic **còn book được hay không**, khớp với backend.
- Khi ghế **hết chỗ** chỉ vì **hết thời gian book được** (không có booking thật trên day-status), **không** hiển thị số badge giả như có thêm người đặt.

---

## 5. Admin cần kiểm tra gì sau khi triển khai

1. **User thường:** thử hủy một booking đã qua giờ vào → phải bị chặn, có thông báo rõ.
2. **Admin:** thử hủy cùng booking đó → được phép (trong phạm vi quyền).
3. **Ghế không seat type:** office hours cố định, đặt tình huống cuối ngày chỉ còn ít phút → trạng thái **hết chỗ** đúng kỳ vọng.

---
