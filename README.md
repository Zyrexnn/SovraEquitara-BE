# ⚙️ SovraEquitara Backend API

Mesin utama platform **SovraEquitara**, dibangun menggunakan **Go (Golang)** dan framework **Fiber** untuk menangani logika bisnis, autentikasi, dan persistensi data dengan performa tinggi.

---

## 🛠️ Tech Stack
- **Language**: [Go v1.21+](https://go.dev/)
- **Framework**: [Fiber v2](https://gofiber.io/)
- **Database**: [PostgreSQL](https://www.postgresql.org/)
- **Authentication**: JWT (JSON Web Token)
- **Email Service**: SMTP (untuk Reset Password)

---

## ⚙️ Persiapan Pengembangan

### 1. Prasyarat
- **Go**: v1.21 atau lebih baru.
- **PostgreSQL**: Server database yang aktif.

### 2. Database Setup
Buat database baru di PostgreSQL dengan nama `sovra_equitara`. Tabel akan otomatis di-migrasi saat pertama kali server dijalankan (jika menggunakan ORM/Auto-migration).

### 3. Konfigurasi Environment
Buat file `.env` di direktori `be/` dan sesuaikan nilainya:
```env
PORT=3000
JWT_SECRET=rahasia-super-kuat-anda
DATABASE_URL=postgresql://user:password@localhost:5432/sovra_equitara?sslmode=disable

# Konfigurasi SMTP (Opsional untuk Reset Password)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=email-anda@gmail.com
SMTP_PASS=password-aplikasi-anda
SMTP_FROM=SovraEquitara <noreply@sovraequitara.com>
```

### 4. Instalasi Dependensi
```bash
go mod tidy
```

### 5. Jalankan Server
```bash
go run main.go
```
API akan aktif di: `http://localhost:3000`

---

## 🔗 Endpoint Utama
- `POST /api/register` - Pendaftaran warga baru.
- `POST /api/login` - Autentikasi & perolehan token JWT.
- `GET /api/public-reports` - Melihat feed laporan kota.
- `POST /api/reports` - Mengirim laporan baru (Perlu JWT).
- `GET /api/leaderboard` - Melihat 10 warga teladan teratas.

---

## 🚀 Deployment
Kompilasi menjadi binary executable:
```bash
go build -o sovra-api main.go
```

© 2026 **SovraEquitara** &bull; *Membangun Kota dengan Data.*
