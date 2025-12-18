# Poros Kullanım Kılavuzu

**Poros** (Yunanca: Πόρος - "yol, geçit") modern, cross-platform bir ağ yol izleme aracıdır.

---

## İçindekiler

1. [Kurulum](#kurulum)
2. [Hızlı Başlangıç](#hızlı-başlangıç)
3. [Probe Metodları](#probe-metodları)
   - [ICMP Probe](#icmp-probe-varsayılan)
   - [UDP Probe](#udp-probe)
   - [TCP SYN Probe](#tcp-syn-probe)
   - [Paris Traceroute](#paris-traceroute)
4. [Trace Parametreleri](#trace-parametreleri)
5. [Ağ Ayarları](#ağ-ayarları)
6. [Çıktı Formatları](#çıktı-formatları)
   - [Text Çıktı](#text-çıktı-varsayılan)
   - [Verbose Tablo](#verbose-tablo-çıktısı)
   - [JSON Çıktı](#json-çıktısı)
   - [CSV Çıktı](#csv-çıktısı)
   - [HTML Rapor](#html-raporu)
7. [TUI (Terminal User Interface)](#tui-interaktif-arayüz)
8. [Zenginleştirme (Enrichment)](#zenginleştirme-enrichment)
   - [Reverse DNS](#reverse-dns-rdns)
   - [ASN Lookup](#asn-lookup)
   - [GeoIP Lookup](#geoip-lookup)
9. [Gelişmiş Kullanım](#gelişmiş-kullanım)
10. [Sorun Giderme](#sorun-giderme)
11. [Örnekler](#örnekler)

---

## Kurulum

### Go ile Kurulum
```bash
go install github.com/KilimcininKorOglu/poros/cmd/poros@latest
```

### Homebrew (macOS/Linux)
```bash
brew tap KilimcininKorOglu/tap
brew install poros
```

### Arch Linux (AUR)
```bash
yay -S poros       # Kaynak koddan
yay -S poros-bin   # Hazır binary
```

### Docker
```bash
docker run --cap-add=NET_RAW ghcr.io/kilimcininkoroglu/poros google.com
```

### Kaynak Koddan
```bash
git clone https://github.com/KilimcininKorOglu/poros.git
cd poros
make build
sudo ./bin/poros google.com
```

---

## Hızlı Başlangıç

### Temel Kullanım
```bash
# En basit kullanım - ICMP ile trace
poros google.com

# Çıktı örneği:
# traceroute to google.com (142.250.185.238), 30 hops max
#
#   1  router.local (192.168.1.1)  1.234 ms  1.456 ms  1.123 ms
#   2  10.0.0.1  5.678 ms  5.432 ms  5.555 ms  [AS15169 Google]
#   3  * * *
#   4  dns.google (8.8.8.8)  12.345 ms  12.123 ms  12.456 ms
#
# Trace complete. 4 hops, 12.31 ms total
```

### Yetki Gereksinimleri

| Platform | Gereksinim | Komut |
|----------|------------|-------|
| Linux | Root veya CAP_NET_RAW | `sudo poros target` veya `sudo setcap cap_net_raw+ep ./poros` |
| macOS | Root | `sudo poros target` |
| Windows | Administrator | Sağ tık → "Yönetici olarak çalıştır" |

---

## Probe Metodları

### ICMP Probe (Varsayılan)

ICMP Echo Request paketleri kullanır. En güvenilir yöntemdir.

```bash
# Açık belirtme (opsiyonel, varsayılan zaten ICMP)
poros -I google.com
poros --icmp google.com
```

**Özellikler:**
- ✅ En yaygın desteklenen
- ✅ Düşük overhead
- ❌ Bazı ağlarda engellenebilir

**Teknik Detaylar:**
- ICMP Type 8 (Echo Request) gönderir
- ICMP Type 0 (Echo Reply) veya Type 11 (Time Exceeded) bekler
- TTL her hop için 1'den başlayarak artırılır

---

### UDP Probe

Yüksek portlara UDP paketleri gönderir. ICMP engellendiğinde kullanışlıdır.

```bash
poros -U google.com
poros --udp google.com

# Özel port ile
poros -U -p 53 dns.google.com
poros -U --port 33434 target
```

**Özellikler:**
- ✅ ICMP engelli ağlarda çalışır
- ✅ NAT geçişinde daha başarılı
- ❌ Hedef port açık değilse ICMP Port Unreachable bekler

**Varsayılan Port:** 33434 (klasik traceroute portu)

**Port Aralığı:** Her probe için port 1 artırılır (33434, 33435, 33436...)

---

### TCP SYN Probe

TCP SYN paketleri gönderir. Firewall'ların HTTP/HTTPS trafiğine izin verdiği durumlarda kullanışlıdır.

```bash
poros -T google.com           # Port 80 (varsayılan)
poros -T -p 443 google.com    # HTTPS portu
poros --tcp --port 22 target  # SSH portu
```

**Özellikler:**
- ✅ Firewall-friendly (80, 443 portları genelde açık)
- ✅ Web sunucularına trace için ideal
- ❌ Daha fazla kaynak tüketir

**Yaygın Portlar:**
| Port | Servis | Kullanım |
|------|--------|----------|
| 80 | HTTP | Web sunucuları |
| 443 | HTTPS | Güvenli web |
| 22 | SSH | Sunucular |
| 53 | DNS | DNS sunucuları |

---

### Paris Traceroute

Load balancer'lara rağmen tutarlı yol izleme sağlar.

```bash
poros --paris google.com
poros --paris -U google.com   # Paris + UDP
```

**Neden Paris Traceroute?**

Klasik traceroute'ta her probe farklı bir "flow" olarak görülür ve load balancer farklı yollar seçebilir:

```
Klasik Traceroute:
  Probe 1 → Router A → Server 1
  Probe 2 → Router B → Server 2  (farklı yol!)
  Probe 3 → Router A → Server 1

Paris Traceroute:
  Probe 1 → Router A → Server 1
  Probe 2 → Router A → Server 1  (aynı yol!)
  Probe 3 → Router A → Server 1
```

**Teknik Detaylar:**
- Sabit flow identifier kullanır
- ICMP: Aynı ID, checksum ayarlaması
- UDP: Aynı kaynak/hedef port çifti

---

## Trace Parametreleri

### Maksimum Hop Sayısı (-m, --max-hops)

Trace'in maksimum kaç hop'ta duracağını belirler.

```bash
poros -m 15 google.com        # Max 15 hop
poros --max-hops 64 target    # Max 64 hop
```

**Varsayılan:** 30 hop  
**Aralık:** 1-255

---

### Probe Sayısı (-q, --queries)

Her hop için gönderilecek probe sayısı.

```bash
poros -q 1 google.com         # Hızlı trace (1 probe)
poros -q 5 google.com         # Daha güvenilir (5 probe)
poros --queries 10 target     # Detaylı istatistik
```

**Varsayılan:** 3 probe  
**Aralık:** 1-10

**İstatistik Etkisi:**
- 1 probe: Sadece tek RTT değeri
- 3 probe: Avg/Min/Max hesaplanabilir
- 5+ probe: Jitter (sapma) daha doğru

---

### Timeout (-w, --timeout)

Her probe için maksimum bekleme süresi.

```bash
poros -w 1s google.com        # 1 saniye timeout
poros -w 5s target            # 5 saniye (yavaş ağlar için)
poros --timeout 500ms target  # 500 milisaniye
```

**Varsayılan:** 3 saniye  
**Format:** `100ms`, `1s`, `1m` (Go duration formatı)

---

### Başlangıç Hop'u (-f, --first-hop)

Trace'in hangi TTL'den başlayacağını belirler.

```bash
poros -f 5 google.com         # İlk 4 hop'u atla
poros --first-hop 10 target   # 10. hop'tan başla
```

**Varsayılan:** 1  
**Kullanım Alanı:** Yerel ağ hop'larını atlamak için

---

### Sequential Mode (--sequential)

Varsayılan concurrent mode yerine sıralı mod kullanır.

```bash
poros --sequential google.com
```

**Concurrent (Varsayılan):**
- Tüm hop'lara paralel probe gönderir
- Çok daha hızlı (5-10x)
- Ağ üzerinde daha fazla yük

**Sequential:**
- Her hop'u sırayla probe eder
- Daha yavaş ama daha güvenilir
- Hassas ağlar için önerilir

---

## Ağ Ayarları

### IPv4/IPv6 Zorlama (-4, -6)

```bash
poros -4 google.com           # Sadece IPv4
poros -6 ipv6.google.com      # Sadece IPv6
poros --ipv4 target
poros --ipv6 target
```

**Varsayılan:** Sistem tercihi (genelde IPv4)

---

### Hedef Port (-p, --port)

UDP ve TCP probe'ları için hedef port.

```bash
poros -U -p 53 dns.google     # UDP port 53
poros -T -p 443 google.com    # TCP port 443
poros --tcp --port 8080 api   # TCP port 8080
```

**Varsayılanlar:**
- UDP: 33434
- TCP: 80

---

### Ağ Arayüzü (-i, --interface)

Belirli bir ağ arayüzünü kullanır.

```bash
poros -i eth0 google.com
poros --interface wlan0 target
```

**Kullanım:** Birden fazla NIC olan sistemlerde

---

### Kaynak IP (-s, --source)

Paketlerin kaynak IP adresini belirler.

```bash
poros -s 192.168.1.100 google.com
poros --source 10.0.0.5 target
```

**Kullanım:** Multi-homed sistemlerde

---

## Çıktı Formatları

### Text Çıktı (Varsayılan)

Klasik traceroute tarzı çıktı.

```bash
poros google.com
```

**Örnek Çıktı:**
```
traceroute to google.com (142.250.185.238), 30 hops max

  1  router.local (192.168.1.1)  1.234 ms  1.456 ms  1.123 ms
  2  10.0.0.1  5.678 ms  5.432 ms  5.555 ms  [AS15169 Google]
  3  * * *
  4  dns.google (8.8.8.8)  12.345 ms  12.123 ms  12.456 ms

Trace complete. 4 hops, 12.31 ms total
```

**Renk Kodlaması:**
- 🟢 Yeşil: Hızlı RTT (<50ms)
- 🟡 Sarı: Orta RTT (50-150ms)
- 🔴 Kırmızı: Yavaş RTT (>150ms)
- ⚪ Gri: Timeout (*)

---

### Verbose Tablo Çıktısı (-v)

Detaylı tablo formatında çıktı.

```bash
poros -v google.com
poros --verbose target
```

**Örnek Çıktı:**
```
┌─────┬─────────────────────┬─────────────────┬────────────┬────────────┬─────────┬────────────────────┐
│ Hop │ IP                  │ Hostname        │ Avg RTT    │ Loss       │ ASN     │ Location           │
├─────┼─────────────────────┼─────────────────┼────────────┼────────────┼─────────┼────────────────────┤
│ 1   │ 192.168.1.1         │ router.local    │ 1.27 ms    │ 0%         │ -       │ -                  │
│ 2   │ 10.0.0.1            │ -               │ 5.55 ms    │ 0%         │ AS15169 │ United States      │
│ 3   │ *                   │ -               │ -          │ 100%       │ -       │ -                  │
│ 4   │ 8.8.8.8             │ dns.google      │ 12.31 ms   │ 0%         │ AS15169 │ United States      │
└─────┴─────────────────────┴─────────────────┴────────────┴────────────┴─────────┴────────────────────┘
```

**Ek Bilgiler:**
- Min/Max/Avg RTT
- Packet loss yüzdesi
- ASN bilgisi
- Coğrafi konum

---

### JSON Çıktısı (-j, --json)

Makineler tarafından okunabilir JSON formatı.

```bash
poros -j google.com
poros --json google.com > trace.json

# Pretty print ile
poros --json google.com | jq .
```

**Örnek Çıktı:**
```json
{
  "target": "google.com",
  "resolved_ip": "142.250.185.238",
  "probe_method": "icmp",
  "max_hops": 30,
  "probe_count": 3,
  "start_time": "2025-12-18T12:00:00Z",
  "end_time": "2025-12-18T12:00:05Z",
  "completed": true,
  "hops": [
    {
      "hop": 1,
      "ip": "192.168.1.1",
      "hostname": "router.local",
      "rtt_ms": [1.234, 1.456, 1.123],
      "avg_rtt_ms": 1.271,
      "min_rtt_ms": 1.123,
      "max_rtt_ms": 1.456,
      "loss_percent": 0,
      "asn": null,
      "geo": null
    },
    {
      "hop": 2,
      "ip": "10.0.0.1",
      "hostname": null,
      "rtt_ms": [5.678, 5.432, 5.555],
      "avg_rtt_ms": 5.555,
      "min_rtt_ms": 5.432,
      "max_rtt_ms": 5.678,
      "loss_percent": 0,
      "asn": {
        "number": 15169,
        "name": "Google LLC",
        "country": "US"
      },
      "geo": {
        "country": "United States",
        "city": "Mountain View",
        "lat": 37.386,
        "lon": -122.084
      }
    }
  ],
  "summary": {
    "total_hops": 4,
    "responding_hops": 3,
    "total_time_ms": 12.31,
    "avg_rtt_ms": 6.38
  }
}
```

**Kullanım Alanları:**
- Scriptlerde işleme
- Log sistemlerine gönderme
- API entegrasyonları

---

### CSV Çıktısı (--csv)

Tablo verisi olarak CSV formatı.

```bash
poros --csv google.com
poros --csv google.com > trace.csv
```

**Örnek Çıktı:**
```csv
hop,ip,hostname,avg_rtt_ms,min_rtt_ms,max_rtt_ms,loss_percent,asn,asn_name,country,city
1,192.168.1.1,router.local,1.271,1.123,1.456,0,,,, 
2,10.0.0.1,,5.555,5.432,5.678,0,15169,Google LLC,US,Mountain View
3,*,,-1,-1,-1,100,,,,
4,8.8.8.8,dns.google,12.31,12.123,12.456,0,15169,Google LLC,US,
```

**Kullanım Alanları:**
- Excel/Google Sheets analizi
- Veritabanına import
- Raporlama

---

### HTML Raporu (--html)

Görsel HTML rapor dosyası oluşturur.

```bash
poros --html report.html google.com
poros -v --html detailed.html target

# Diğer formatlarla birlikte
poros --json --html report.html google.com
```

**Rapor Özellikleri:**
- 🌙 Modern dark theme tasarım
- 📊 Detaylı hop tablosu
- 📈 RTT renk kodlaması
- 📋 Özet istatistikler
- 🕐 Oluşturulma zamanı
- 📱 Responsive tasarım

**Örnek Rapor Bölümleri:**
1. **Header:** Hedef, IP, probe metodu
2. **Hop Table:** Tüm hop'lar detaylı
3. **Summary:** Toplam hop, ortalama RTT, completion
4. **Footer:** Poros branding, timestamp

---

## TUI (Interaktif Arayüz)

Terminal User Interface ile gerçek zamanlı trace izleme.

```bash
poros -t google.com
poros --tui google.com
```

### TUI Özellikleri

**Görsel Elemanlar:**
- Real-time hop tablosu
- Canlı RTT güncelleme
- Progress spinner
- Renk temalı gösterim

**Klavye Kısayolları:**
| Tuş | Fonksiyon |
|-----|-----------|
| `q` | Çıkış |
| `Ctrl+C` | İptal |
| `↑/↓` | Scroll |

**Renk Temaları:**
- **Dark** (varsayılan): Koyu arka plan
- **Light**: Açık arka plan
- **Minimal**: Sadece temel renkler

---

## Zenginleştirme (Enrichment)

### Reverse DNS (rDNS)

IP adreslerini hostname'lere çözer.

```bash
# Aktif (varsayılan)
poros google.com

# Devre dışı bırak
poros --no-rdns google.com
```

**Örnek:**
```
192.168.1.1 → router.local
8.8.8.8 → dns.google
```

---

### ASN Lookup

IP adreslerinin ait olduğu Autonomous System bilgisini gösterir.

```bash
# Aktif (varsayılan)
poros google.com

# Devre dışı bırak
poros --no-asn google.com
```

**Veri Kaynağı:** Team Cymru DNS

**Örnek:**
```
[AS15169 Google LLC]
[AS13335 Cloudflare]
[AS3356 Lumen Technologies]
```

---

### GeoIP Lookup

IP adreslerinin coğrafi konumunu gösterir.

```bash
# Aktif (varsayılan)
poros google.com

# Devre dışı bırak
poros --no-geoip google.com
```

**Veri Kaynağı:** ip-api.com

**Gösterilen Bilgiler:**
- Ülke
- Şehir
- Koordinatlar (JSON'da)

---

### Tüm Enrichment'ı Devre Dışı Bırakma

```bash
poros --no-enrich google.com
```

**Ne Zaman Kullanılır:**
- Hızlı trace gerektiğinde
- Gizlilik endişesi varsa
- API rate limit aşıldığında

---

## Gelişmiş Kullanım

### Birden Fazla Flag Kombinasyonu

```bash
# TCP probe, verbose, HTML rapor
poros -T -v --html report.html -p 443 google.com

# UDP Paris mode, 5 probe, JSON çıktı
poros -U --paris -q 5 --json target

# Hızlı trace: 1 probe, 1s timeout, no enrichment
poros -q 1 -w 1s --no-enrich google.com

# Detaylı trace: 10 probe, sequential, full enrichment
poros -q 10 --sequential -v google.com
```

### Script Entegrasyonu

```bash
#!/bin/bash
# Birden fazla hedefe trace

targets=("google.com" "cloudflare.com" "amazon.com")

for target in "${targets[@]}"; do
    echo "Tracing $target..."
    poros --json "$target" > "trace_${target}.json"
done
```

### Çıktıyı Filtreleme (jq ile)

```bash
# Sadece IP'leri al
poros --json google.com | jq '.hops[].ip'

# Ortalama RTT > 50ms olan hop'lar
poros --json google.com | jq '.hops[] | select(.avg_rtt_ms > 50)'

# ASN bazında gruplama
poros --json google.com | jq '[.hops[] | select(.asn != null)] | group_by(.asn.number)'
```

### Monitoring için Periyodik Trace

```bash
# Her 5 dakikada bir trace
while true; do
    timestamp=$(date +%Y%m%d_%H%M%S)
    poros --json google.com > "trace_${timestamp}.json"
    sleep 300
done
```

---

## Sorun Giderme

### "Permission denied" Hatası

**Sebep:** Raw socket oluşturmak için yetki gerekli.

**Çözümler:**

Linux:
```bash
# Seçenek 1: sudo ile çalıştır
sudo poros google.com

# Seçenek 2: Capability ekle (kalıcı)
sudo setcap cap_net_raw+ep /usr/local/bin/poros
```

macOS:
```bash
sudo poros google.com
```

Windows:
```
1. Başlat menüsüne sağ tıkla
2. "Windows PowerShell (Yönetici)" seç
3. poros.exe google.com
```

---

### "Network unreachable" Hatası

**Kontroller:**
```bash
# İnternet bağlantısını kontrol et
ping google.com

# DNS çözümlemesini kontrol et
nslookup google.com

# Routing tablosunu kontrol et
ip route  # Linux
netstat -rn  # macOS/Windows
```

---

### Tüm Hop'larda Timeout (*)

**Olası Sebepler:**
1. ICMP engellenmiş olabilir → UDP dene: `poros -U target`
2. Firewall kuralları → TCP dene: `poros -T -p 443 target`
3. Rate limiting → Timeout artır: `poros -w 5s target`

---

### Yavaş Trace

**Hızlandırma Yöntemleri:**
```bash
# Probe sayısını azalt
poros -q 1 target

# Timeout'u düşür
poros -w 1s target

# Enrichment kapat
poros --no-enrich target

# Hepsini birleştir
poros -q 1 -w 1s --no-enrich target
```

---

## Örnekler

### Web Sunucusu Analizi
```bash
# HTTPS bağlantısı simülasyonu
poros -T -p 443 -v google.com
```

### DNS Sunucusu Trace
```bash
# DNS trafiği simülasyonu
poros -U -p 53 8.8.8.8
```

### CDN Analizi
```bash
# Cloudflare edge'e trace
poros --paris -v cloudflare.com
```

### Karşılaştırmalı Analiz
```bash
# JSON ile karşılaştır
poros --json google.com > google.json
poros --json cloudflare.com > cloudflare.json
diff <(jq '.summary' google.json) <(jq '.summary' cloudflare.json)
```

### Raporlama
```bash
# Detaylı HTML rapor
poros -v --html network_report.html \
    -q 5 \
    --paris \
    target.example.com
```

---

## Komut Referansı

```
poros [flags] <target>

Probe Metodları:
  -I, --icmp           ICMP Echo probe kullan (varsayılan)
  -U, --udp            UDP probe kullan
  -T, --tcp            TCP SYN probe kullan
      --paris          Paris traceroute algoritması

Trace Parametreleri:
  -m, --max-hops int       Maksimum hop sayısı (varsayılan: 30)
  -q, --queries int        Her hop için probe sayısı (varsayılan: 3)
  -w, --timeout duration   Probe timeout süresi (varsayılan: 3s)
  -f, --first-hop int      Başlangıç hop'u (varsayılan: 1)
      --sequential         Sıralı mod kullan

Ağ Ayarları:
  -4, --ipv4           Sadece IPv4 kullan
  -6, --ipv6           Sadece IPv6 kullan
  -p, --port int       Hedef port (UDP/TCP) (varsayılan: 33434/80)
  -i, --interface      Ağ arayüzü
  -s, --source         Kaynak IP adresi

Çıktı Formatları:
  -v, --verbose        Detaylı tablo çıktısı
  -j, --json           JSON formatında çıktı
      --csv            CSV formatında çıktı
      --html string    HTML rapor dosyası oluştur
  -t, --tui            İnteraktif TUI modu
      --no-color       Renkli çıktıyı devre dışı bırak

Zenginleştirme:
      --no-enrich      Tüm zenginleştirmeyi kapat
      --no-rdns        Reverse DNS'i kapat
      --no-asn         ASN lookup'ı kapat
      --no-geoip       GeoIP lookup'ı kapat

Diğer:
  -h, --help           Yardım mesajını göster
      version          Versiyon bilgisini göster
```

---

## Sürüm Bilgisi

```bash
poros version
```

**Çıktı:**
```
Poros v1.0.0
  Commit: abc123
  Built:  2025-12-18T12:00:00Z
```

---

© 2025 Poros Contributors | MIT License
