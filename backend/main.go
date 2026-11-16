package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"gis-app/backend/internal/database"
	"gis-app/backend/internal/routes"

	"github.com/joho/godotenv"
)

func main() {
	// Baca konfigurasi .env.
	if err := godotenv.Load(); err != nil {
		log.Printf("Gagal membaca .env: %v", err)
		return
	}

	// Hubungkan PostgreSQL.
	db, err := database.Connect()
	if err != nil {
		log.Printf("Koneksi PostgreSQL gagal: %v", err)
		return
	}
	defer db.Close()

	log.Println("PostgreSQL berhasil terhubung")

	// Ambil port dari .env.
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "9000"
	}

	// Daftarkan seluruh route dan konfigurasi server.
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           routes.Setup(db),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("API berjalan di http://127.0.0.1:%s", port)

	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		log.Printf("Server gagal berjalan: %v", err)
	}
}
