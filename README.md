# ⚙️ SovraEquitara Backend API (Go/Fiber)

Backend API utama untuk platform **SovraEquitara**. Dibangun menggunakan bahasa pemrograman **Go (Golang)** dan web framework **Fiber v2** untuk memberikan performa API dengan konkurensi tinggi, penggunaan memori minimal, serta waktu respons secepat kilat. 

Database menggunakan **PostgreSQL** native (dijalankan di Docker) dengan ORM **GORM**.

---

## 🛠️ Tech Stack & Fitur Utama

- **Bahasa**: [Go (Golang) v1.21+](https://go.dev/)
- **Web Framework**: [Fiber v2](https://gofiber.io/) (Fast & Lightweight)
- **Database Relasional**: PostgreSQL dengan ORM **GORM**
- **Sistem Keamanan**:
  - Enkripsi password menggunakan hashing **Argon2**
  - Autentikasi token berbasis **JWT (JSON Web Tokens)**
  - Pengaman OTP (One-Time Password) untuk proses pendaftaran dan pemulihan kata sandi dengan deteksi *brute-force lockout*.
- **Real-time Event Stream**: **SSE (Server-Sent Events)** untuk pengumuman darurat dan sinkronisasi status laporan secara real-time.
- **Kecerdasan Buatan**: Integrasi **AI Assistant** dengan LLM lokal (LM Studio) / OpenAI API untuk panduan dan analisis laporan otomatis.

---

## 📂 Struktur Folder API

```bash
be/
├── cmd/
│   └── api/             # Entry point utama (main.go)
├── internal/
│   ├── config/          # Pengaturan env & konfigurasi sistem
│   ├── middleware/      # JWT auth guard, CORS, & guard tingkat role (Admin/Super Admin)
│   ├── model/           # Definisi skema struktur database GORM
│   ├── repository/      # Kueri database & manipulasi data PostgreSQL
│   └── handler/         # Logika penanganan HTTP request/response (handler.go)
├── backups/             # Direktori penyimpanan otomatis database backup
├── uploads/             # Media penyimpanan file gambar laporan & avatar profil
├── docker-compose.yaml  # Dockerized setup database PostgreSQL
└── setup.sql            # Skema lengkap SQL inisialisasi awal database
```

---

## 🔗 Dokumentasi Endpoint API

Semua rute di bawah diawali dengan prefix `/api`.

### 1. Autentikasi Publik (`/auth/*`)
| Method | Endpoint | Keterangan |
| :--- | :--- | :--- |
| `POST` | `/auth/register` | Mendaftarkan akun warga baru (Mengirim OTP ke email) |
| `POST` | `/auth/verify` | Memverifikasi OTP untuk mengaktifkan akun baru |
| `POST` | `/auth/resend-otp` | Mengirim ulang kode OTP registrasi |
| `POST` | `/auth/login` | Autentikasi Warga & Admin (Mendapatkan token JWT) |
| `POST` | `/auth/admin-login` | Autentikasi khusus Admin / Super Admin |
| `POST` | `/auth/forgot-password` | Request OTP untuk pemulihan kata sandi |
| `POST` | `/auth/verify-forgot-password-otp` | Validasi OTP pemulihan kata sandi |
| `POST` | `/auth/reset-password` | Mengganti password lama dengan password baru |

### 2. Data Publik (Bebas Akses)
| Method | Endpoint | Keterangan |
| :--- | :--- | :--- |
| `GET` | `/categories` | Mengambil semua kategori laporan (Infrastruktur, Lingkungan, dll) |
| `GET` | `/public-reports` | Mengambil daftar feed laporan warga publik |
| `GET` | `/reports/:id` | Mengambil detail satu laporan spesifik |
| `GET` | `/reports/:id/comments` | Mengambil daftar komentar dari suatu laporan |
| `GET` | `/leaderboard` | Mengambil 10 daftar warga dengan reputasi poin tertinggi |
| `GET` | `/profiles` | Mengambil ringkasan profil publik |
| `GET` | `/profiles/:id` | Mengambil detail profil publik warga |
| `GET` | `/events` | **SSE Stream**: Server-Sent Events untuk sinkronisasi update data |

### 3. Rute Warga Terproteksi (Butuh JWT Token di Authorization Header)
| Method | Endpoint | Keterangan |
| :--- | :--- | :--- |
| `GET` | `/my-reports` | Mengambil riwayat laporan milik sendiri |
| `POST` | `/reports` | Membuat laporan baru (Mendukung upload gambar & geolokasi) |
| `DELETE` | `/reports/:id` | Menghapus laporan yang dikirim (Hanya jika masih `PENDING`) |
| `POST` | `/reports/:id/comments` | Mengirim komentar baru pada suatu laporan |
| `POST` | `/reports/:id/vote` | Memberikan vote/dukungan pada laporan |
| `GET` | `/reports/:id/vote-status` | Memeriksa status vote milik sendiri di laporan tersebut |
| `PATCH`| `/reports/:id/approve-resolution` | Warga menyetujui resolusi perbaikan dari Admin |
| `PATCH`| `/reports/:id/reject-resolution` | Warga menolak resolusi (Mengembalikan status ke `VALID`) |
| `GET` | `/reports/stats` | Mengambil statistik personal laporan |
| `GET` | `/profile` | Mengambil info profil sendiri |
| `PUT` | `/profile` | Memperbarui nama, telepon, atau setelan profil |
| `POST` | `/profile/avatar` | Mengunggah foto profil (avatar) baru |
| `PUT` | `/profile/password` | Mengganti kata sandi dengan validasi OTP profil |
| `POST` | `/profile/password-otp` | Mengirim OTP keamanan untuk perubahan password |
| `GET` | `/notifications` | Mengambil daftar notifikasi personal |
| `POST` | `/ai-assistant` | Bertanya pada asisten AI berbasis riwayat laporan warga |
| `POST` | `/chat/send` | Mengirim pesan bantuan bantuan ke inbox Admin |
| `GET` | `/chat/messages` | Mengambil riwayat pesan bantuan miliknya |

### 4. Rute Khusus Admin (Butuh JWT + Role `admin`/`super_admin`)
| Method | Endpoint | Keterangan |
| :--- | :--- | :--- |
| `GET` | `/admin/reports` | Mengambil seluruh laporan di sistem untuk moderasi |
| `GET` | `/admin/saved-reports` | Mengambil daftar laporan yang ditandai/disimpan Admin |
| `POST` | `/admin/reports/:id/save` | Menandai atau menghapus tanda simpan laporan |
| `PATCH`| `/admin/reports/:id/verify` | Validasi laporan warga (`PENDING` ➔ `VALID`) |
| `PATCH`| `/admin/reports/:id/resolve` | Menyelesaikan perbaikan laporan (`VALID` ➔ `RESOLVED`) |
| `POST` | `/admin/ai-assistant` | Bertanya ke AI khusus administrasi kota |
| `GET` | `/admin/ai-assistant/threads` | Mengelola thread percakapan AI Admin |
| `POST` | `/admin/notifications` | Membuat notifikasi / broadcast baru ke sistem |
| `GET` | `/admin/chat/conversations` | Mengambil seluruh daftar chat masuk dari Warga |
| `POST`| `/admin/chat/conversations/:id/reply`| Membalas pesan chat warga |

### 5. Rute Khusus Super Admin (Butuh JWT + Role `super_admin`)
| Method | Endpoint | Keterangan |
| :--- | :--- | :--- |
| `PATCH`| `/superadmin/reports/:id/cancel` | Membatalkan laporan warga secara absolut (`REJECTED`) |
| `POST` | `/superadmin/database/backup` | Melakukan backup database PostgreSQL ke file `.sql` |
| `GET` | `/superadmin/profiles/:id/stats` | Melihat analitik aktivitas detail seorang user |
| `GET` | `/superadmin/admins` | Mengambil semua daftar staf Admin |
| `POST` | `/superadmin/admins` | Menambahkan akun Admin baru |
| `PUT` | `/superadmin/admins/:id` | Memperbarui data akun Admin |
| `DELETE`| `/superadmin/admins/:id` | Menghapus hak akses akun Admin |

---

## 🗄️ Desain Skema Database (PostgreSQL)

Database PostgreSQL mengelola 13 tabel relasional utama dengan performa tinggi:

- **`profiles`**: Pengelolaan data autentikasi mandiri (email, hash Argon2, nama, poin reputasi, dan peran).
- **`categories`**: Daftar kategori isu kota (Infrastruktur, Lingkungan, Fasilitas Umum, Keamanan).
- **`reports`**: Data utama laporan warga, memuat array `image_urls`, geolokasi (`latitude` & `longitude`), deskripsi, poin vote, dan status (`PENDING`, `VALID`, `RESOLVED`, `REJECTED`).
- **`comments` & `votes`**: Interaksi sosial warga terhadap laporan kota.
- **`saved_reports`**: Laporan yang disimpan admin untuk penanganan prioritas.
- **`conversations` & `messages`**: Saluran live chat helpdesk warga ke admin.
- **`notifications`**: Riwayat pengiriman informasi dan pemberitahuan penting.
- **`ai_threads` & `ai_messages`**: Riwayat interaksi asisten kecerdasan buatan.
- **`otps` & `forgot_password_otps`**: Sistem verifikasi OTP sementara dengan kolom pencegahan eksploitasi brute force (`failed_attempts`, `blocked_until`).

### ⚙️ Mekanisme Otomatis & Housekeeping Database
1. **Trigger `update_updated_at_column`**: Secara otomatis mengubah nilai kolom `updated_at` setiap terjadi pembaruan record pada tabel `profiles` dan `reports`.
2. **Background OTP Housekeeping**: Aplikasi backend menjalankan *goroutine process* berkala setiap 1 jam untuk menghapus kode OTP lama yang telah kedaluwarsa (> 10 menit) dari tabel `otps` dan `forgot_password_otps` agar database tetap bersih.

---

## ⚙️ Persiapan & Pengembangan Backend

### 1. Konfigurasi Environment (`.env`)
Buat file `.env` di direktori `/be` dengan struktur berikut:
```env
PORT=3000
JWT_SECRET=gunakan-kunci-rahasia-yang-sangat-kuat-disini
DATABASE_URL=postgresql://postgres:postgrespassword@localhost:5432/sovra_equitara?sslmode=disable

# Konfigurasi SMTP (Untuk Pengiriman Email OTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=emailanda@gmail.com
SMTP_PASS=passwordaplikasianda
SMTP_FROM=SovraEquitara <noreply@sovraequitara.com>
```

### 2. Jalankan PostgreSQL via Docker Compose
Jika belum aktif, jalankan container database:
```bash
docker-compose up -d
```
Skema SQL dasar dapat diimpor secara otomatis melalui `setup.sql` jika inisialisasi awal database diperlukan.

### 3. Instalasi Dependensi
```bash
go mod tidy
```

### 4. Menjalankan Server API Go
```bash
go run cmd/api/main.go
```
*API akan aktif di: `http://localhost:3000`.*

### 5. Menjalankan Pengujian API (Testing)
Untuk memastikan endpoint berjalan dengan baik, Anda dapat mencoba menjalankan:
```bash
go test ./...
```

---

© 2026 **SovraEquitara API Engine** &bull; *Robust Backend for Transparent Urban Governance.*
