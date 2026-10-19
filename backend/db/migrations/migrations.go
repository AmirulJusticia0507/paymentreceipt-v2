// Package migrations menyediakan seluruh berkas migrasi SQL yang di-embed ke
// dalam binary.
//
// Embedding dipilih supaya deploy tidak perlu menyalin berkas SQL terpisah ke
// dalam image - binary migrate sudah berdiri sendiri, dan versi migrasi yang
// dijalankan selalu persis sama dengan kode yang di-build (tidak ada risiko
// berkas SQL tertinggal atau tidak sinkron).
package migrations

import "embed"

// FS berisi seluruh berkas migrasi (*.up.sql dan *.down.sql) di direktori ini.
//
//go:embed *.sql
var FS embed.FS
