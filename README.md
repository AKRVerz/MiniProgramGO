# Datadog APM Demo — Panduan Lengkap

Project ini terdiri dari 2 service Go (`checkout-service` → `payment-service`) yang
sudah di-instrumentasi dengan Datadog APM (`dd-trace-go`). Tujuannya: mendemonstrasikan
4 golden signals (RPS, Latency, Error Rate, Distributed Tracing) secara live dan
terkontrol, supaya enak dipakai untuk screencast.

---

## 1. Arsitektur

```
load-generator.sh ──> checkout-service (:8081) ──> payment-service (:8082)
                              │                              │
                              └──────── trace context ───────┘
                                   (1 trace, 2 services)
                                          │
                                          v
                                  Datadog Agent (:8126 APM, :8125 DogStatsD)
                                          │
                                          v
                                     Datadog SaaS
```

- `checkout-service`: pintu masuk request (mensimulasikan API gateway / front service).
- `payment-service`: service "berat", punya endpoint `/config` untuk mengatur
  latency tambahan & probabilitas error secara live — tanpa restart.
- Karena trace context diteruskan lewat HTTP header (`httptrace.WrapClient`),
  satu request `/checkout` akan menghasilkan **satu trace** yang melintasi
  dua service — inilah yang akan kamu tunjukkan sebagai distributed tracing.

---

## 2. Setup di Server Fresh (dari nol)

Panduan ini mengasumsikan server Ubuntu/Debian yang masih bersih, belum ada
Docker sama sekali. Jalankan semua perintah ini lewat SSH ke server kamu.

### a. Update sistem & install Docker

```bash
sudo apt update && sudo apt upgrade -y

# Install Docker Engine + Compose plugin lewat script resmi Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
rm get-docker.sh
```

### b. Supaya tidak perlu `sudo` tiap pakai `docker`

```bash
sudo usermod -aG docker $USER
```
**Logout dari SSH, lalu login lagi** (wajib, supaya keanggotaan grup baru
ini aktif). Verifikasi:
```bash
docker ps
```
Kalau muncul tabel kosong tanpa minta password, berarti sudah benar.

### c. Pindahkan project ke server

Pilih salah satu cara:

**Cara 1 — upload langsung (paling simpel untuk sekali pakai)**
Dari laptop kamu (bukan dari server), kirim folder project lewat `scp`:
```bash
scp -r datadog-demo/ <user>@<ip_server>:~/
```

**Cara 2 — lewat Git** (lebih rapi kalau kamu sudah push ke repo)
```bash
git clone <url_repo_kamu>
cd datadog-demo
```

### d. Cek port yang sudah dipakai di server (penting di server fresh yang sudah ada service lain)

```bash
sudo ss -tulnp | grep -E ':8081|:8082|:8126'
```
Kalau kosong (tidak ada output), berarti port-port itu masih bebas dan aman
dipakai. Kalau server kamu sudah punya Datadog Agent terpasang langsung di
host, ini tidak masalah — `docker-compose.yml` di project ini sengaja
**tidak** mem-publish port agent ke host, jadi tidak akan bentrok.

### e. Konfigurasi environment

```bash
cd ~/datadog-demo
cp .env.example .env
nano .env
```
Isi `DD_API_KEY` (dari Datadog → Organization Settings → API Keys) dan
`DD_SITE` (sesuai region akun kamu — cek dari URL saat login Datadog, contoh
`us5.datadoghq.com` kalau URL-nya `app.us5.datadoghq.com`). Simpan (`Ctrl+O`,
`Enter`, `Ctrl+X` di nano).

### f. Jalankan

```bash
docker compose up --build
```

Biarkan terminal ini tetap terbuka (logs akan terus muncul di sini). Buka
**SSH session/tab baru** untuk lanjut ke langkah tes.

### g. Tes

Di terminal/tab baru (SSH ke server yang sama):
```bash
curl http://localhost:8081/healthz   # checkout-service
curl http://localhost:8082/healthz   # payment-service
curl http://localhost:8081/checkout  # 1 full request lintas 2 service
```
Kalau hasilnya `payment processed`, semuanya sudah tersambung dengan benar.
Lanjutkan ke **APM → Services** di Datadog untuk verifikasi trace masuk.

### h. (Opsional) Jalankan di background

Kalau sudah yakin semuanya jalan normal dan kamu mau lepas dari sesi SSH
tanpa mematikan container:
```bash
# Ctrl+C dulu untuk hentikan mode "attached" di langkah f, lalu:
docker compose up --build -d
docker compose logs -f   # untuk lihat log lagi kapan saja
```

---

## 3. Demo 4 Golden Signals (untuk screencast)

### a. RPS (Requests Per Second)
```bash
chmod +x load-generator.sh
./load-generator.sh 10 180   # ~10 req/detik selama 3 menit
```
Buka **APM → Services → checkout-service** di Datadog, lihat grafik
**Request Rate** naik real-time. Coba ubah angka RPS (`5` lalu `20`) supaya
grafiknya kelihatan jelas naik-turun saat kamu jelaskan di video.

### b. Latency (p50/p95/p99)
Sambil traffic jalan, naikkan latency `payment-service`:
```bash
curl -X POST "http://localhost:8082/config?latency_ms=800&error_rate=0"
```
Lihat di **APM → Services → payment-service**, grafik Latency Distribution /
p50, p95, p99 ikut naik. Balikin ke normal:
```bash
curl -X POST "http://localhost:8082/config?latency_ms=0&error_rate=0"
```
Di video, jelaskan bedanya: p50 = median (separuh request lebih cepat dari ini),
p95/p99 = "ekor" — request paling lambat yang sering jadi indikator masalah
nyata walau rata-rata kelihatan baik-baik saja.

### c. Error Rate
```bash
curl -X POST "http://localhost:8082/config?latency_ms=0&error_rate=0.4"
```
Ini bikin ~40% request gagal (HTTP 500). Lihat **Error Rate** naik di service
`payment-service`, dan trace yang gagal akan ditandai merah di **APM → Traces**.

### d. Distributed Tracing
Buka **APM → Traces**, klik salah satu trace dari `checkout-service`. Kamu akan
lihat flame graph dengan span:
```
checkout-service: GET /checkout
  └─ payment-service: payment.process
       └─ db.query (simulasi)
```
Ini bukti end-to-end tracing — tunjukkan dan jelaskan flame graph ini di video,
termasuk cara melacak bottleneck-nya ada di span mana.

---
