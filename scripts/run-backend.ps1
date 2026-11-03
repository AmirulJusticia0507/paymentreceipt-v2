# scripts/run-backend.ps1
# Menjalankan backend Go dengan otomatis memuat variabel dari backend/.env

$rootDir = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $rootDir "backend\.env"

if (Test-Path $envFile) {
    Write-Host "Memuat konfigurasi dari backend/.env..." -ForegroundColor Cyan
    Get-Content $envFile | ForEach-Object {
        $line = $_.Trim()
        if ($line -and -not $line.StartsWith('#') -and $line.Contains('=')) {
            $parts = $line.Split('=', 2)
            $name = $parts[0].Trim()
            $val = $parts[1].Trim()
            # Hapus tanda kutip jika ada
            if (($val.StartsWith('"') -and $val.EndsWith('"')) -or ($val.StartsWith("'") -and $val.EndsWith("'"))) {
                $val = $val.Substring(1, $val.Length - 2)
            }
            [System.Environment]::SetEnvironmentVariable($name, $val, 'Process')
        }
    }
} else {
    Write-Warning "File backend/.env tidak ditemukan!"
}

Set-Location (Join-Path $rootDir "backend")
Write-Host "Menjalankan server backend Go pada http://localhost:8080..." -ForegroundColor Green
go run ./cmd/api
