# scripts/run-tunnel.ps1
# Menjalankan Cloudflare Tunnel gratis untuk mengekspos backend Go (port 8080) ke internet

Write-Host "===========================================================" -ForegroundColor Cyan
Write-Host " Memulai Cloudflare Tunnel (Gratis) untuk Backend Go" -ForegroundColor Cyan
Write-Host " Target: http://localhost:8080" -ForegroundColor Cyan
Write-Host "===========================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Perhatikan baris log di bawah ini." -ForegroundColor Yellow
Write-Host "Salin tautan HTTPS yang berakhiran: .trycloudflare.com" -ForegroundColor Green
Write-Host "Contoh: https://contoh-acak.trycloudflare.com" -ForegroundColor Green
Write-Host ""

cloudflared tunnel --url http://localhost:8080
