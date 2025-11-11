# Cross-Platform Build Guide

Bu döküman Vigilon projesini farklı platformlar için nasıl derleyeceğinizi gösterir.

## Hızlı Başlangıç

### Tek Komutla Tüm Platformlar

```bash
./build-all.sh
```

Bu script otomatik olarak şunları oluşturur:
- `vigilon-server-linux-amd64` (Server - Linux)
- `vigilon-agent-linux-amd64` (Agent - Linux)
- `vigilon-agent-linux-arm64` (Agent - Raspberry Pi)
- `vigilon-agent-windows-amd64.exe` (Agent - Windows)

## Gereksinimler

### Temel Gereksinimler
- Go 1.24+
- GCC (native compiler)

### Cross-Compilation için Gereksinimler

#### Windows için (MinGW)
```bash
# Ubuntu/Debian
sudo apt-get install gcc-mingw-w64-x86-64

# macOS
brew install mingw-w64
```

#### ARM64 için (Raspberry Pi)
```bash
# Ubuntu/Debian
sudo apt-get install gcc-aarch64-linux-gnu

# macOS
# Homebrew ile ARM64 cross-compiler kurulabilir
```

## Manuel Build

### Linux AMD64 (Mevcut Platform)

```bash
# Server
CGO_ENABLED=1 go build -ldflags="-s -w" -o vigilon-server cmd/server/main.go

# Agent
CGO_ENABLED=1 go build -ldflags="-s -w" -o vigilon-agent cmd/agent/main.go
```

### Linux ARM64 (Raspberry Pi)

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
  go build -ldflags="-s -w" -o vigilon-agent-linux-arm64 cmd/agent/main.go
```

### Windows AMD64

```bash
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
  go build -ldflags="-s -w" -o vigilon-agent-windows-amd64.exe cmd/agent/main.go
```

## Makefile Komutları

```bash
# Yerel platform için build
make build

# Sadece server
make server

# Sadece agent
make agent

# Tüm platformlar (cross-compile gerektirir)
make build-all

# Linux AMD64
make build-linux

# Windows AMD64
make build-windows

# ARM64 (Raspberry Pi)
make build-arm

# Temizlik
make clean
```

## Build Script Detayları

### build-all.sh Özellikleri

1. **Otomatik Platform Algılama**: Sisteminizde hangi cross-compiler'lar varsa onları kullanır
2. **Versiyon Yönetimi**: `VERSION` environment variable ile versiyon belirleyebilirsiniz
3. **Otomatik Checksum**: SHA256 hash'leri oluşturur
4. **Web Binary Kopyalama**: Agent binary'lerini `web/static/bin/` klasörüne kopyalar (install script için)
5. **Renkli Output**: Hangi adımların tamamlandığını gösterir

### Versiyon ile Build

```bash
VERSION=1.1.0 ./build-all.sh
```

### Sadece Belirli Binary'ler

```bash
# Sadece Linux
CGO_ENABLED=1 go build -o dist/vigilon-server-linux-amd64 cmd/server/main.go

# Sadece ARM64 agent
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
  go build -o dist/vigilon-agent-linux-arm64 cmd/agent/main.go
```

## Docker ile Build (Önerilir)

Docker kullanarak tüm cross-compiler'lar otomatik olarak kurulu gelir:

```bash
# Multi-platform Docker build
docker buildx build --platform linux/amd64,linux/arm64 -t vigilon:latest .

# Veya GitHub Actions workflow'u kullan (otomatik)
git tag v1.1.0
git push origin v1.1.0  # Otomatik olarak tüm platformlar için build alır
```

## Çıktı Dizini

Binary'ler `./dist/` klasörüne kaydedilir:

```
dist/
├── vigilon-server-linux-amd64
├── vigilon-agent-linux-amd64
├── vigilon-agent-linux-arm64
├── vigilon-agent-windows-amd64.exe
└── checksums.txt
```

## Optimizasyon Flags

Kullanılan build flags:
- `-s`: Symbol table'ı kaldır (boyut azaltır)
- `-w`: DWARF debug bilgisini kaldır (boyut azaltır)
- `-X main.version=X.X.X`: Versiyon bilgisi ekle

## CGO Neden Gerekli?

Vigilon, SQLite3 veritabanı için `github.com/mattn/go-sqlite3` kullanır. Bu paket C kütüphanesine bağımlıdır ve CGO gerektirir.

### CGO Olmadan Build?

Eğer CGO kullanmadan build almak isterseniz, SQLite yerine pure-Go bir veritabanı kullanmanız gerekir (örn: BoltDB, Badger).

## Sorun Giderme

### "cgo: C compiler not found"

```bash
# GCC yükleyin
sudo apt-get install build-essential
```

### "x86_64-w64-mingw32-gcc: not found"

```bash
# MinGW yükleyin (Windows cross-compile için)
sudo apt-get install gcc-mingw-w64-x86-64
```

### "aarch64-linux-gnu-gcc: not found"

```bash
# ARM64 cross-compiler yükleyin
sudo apt-get install gcc-aarch64-linux-gnu
```

### Binary çok büyük

Binary boyutunu azaltmak için:

```bash
# Build sonrası sıkıştırma
upx --best --lzma vigilon-server-linux-amd64

# Veya strip kullanın
strip vigilon-server-linux-amd64
```

## GitHub Actions ile Otomatik Build

Projede GitHub Actions workflow'ları mevcut:

1. **Tag push yaptığınızda** (örn: `v1.1.0`):
   - Tüm platformlar için binary derlenir
   - GitHub Release oluşturulur
   - Binary'ler release'e eklenir
   - `web/static/bin/` güncellenir

2. **Her commit'te**:
   - Kod test edilir
   - Build kontrolü yapılır
   - Linting çalışır

## Binary Dağıtım

### GitHub Releases
- Tag push edilince otomatik oluşur
- Kullanıcılar direkt indirebilir

### Web Server
- `web/static/bin/` klasöründeki binary'ler
- One-line installer tarafından kullanılır:
  ```bash
  curl -fsSL http://server:8090/install.sh?token=TOKEN | sudo bash
  ```

### Docker Registry
- Docker Hub'a otomatik push edilir
- `docker pull harungecit/vigilon-server:latest`
- `docker pull harungecit/vigilon-agent:latest`

## Örnek Build Akışı

```bash
# 1. Dependency'leri kontrol et
go mod tidy

# 2. Test et
go test ./...

# 3. Build al
./build-all.sh

# 4. Test et
./dist/vigilon-server-linux-amd64 -version

# 5. Checksums kontrol et
cd dist && sha256sum -c checksums.txt

# 6. Git commit & tag
git add .
git commit -m "build: Release v1.1.0"
git tag v1.1.0
git push origin main --tags

# 7. GitHub Actions otomatik build alır ve release oluşturur
```

## Platform Notları

### Linux
- ✅ Tam destek (Server + Agent)
- ✅ systemd service files
- ✅ Native CGO support

### Windows
- ✅ Agent desteği
- ⚠️ Server test edilmedi (Linux tavsiye edilir)
- ✅ Windows Service desteği

### Raspberry Pi (ARM64)
- ✅ Agent desteği
- ✅ systemd service
- ✅ Düşük resource kullanımı

### macOS
- ⚠️ Build alınabilir ama test edilmedi
- CGO cross-compile daha karmaşık
