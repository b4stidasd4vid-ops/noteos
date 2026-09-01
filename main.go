package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"noteos-server/internal/auth"
	"noteos-server/internal/estudiantes"
	"noteos-server/internal/firestoreconfig"
	"noteos-server/internal/supabase"
	"noteos-server/internal/verificacion"
)

func main() {
	_ = godotenv.Load() // en Render las env vars ya existen; localmente usa .env

	supabaseURL := os.Getenv("SUPABASE_URL")
	serviceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if supabaseURL == "" || serviceKey == "" {
		log.Fatal("faltan SUPABASE_URL y/o SUPABASE_SERVICE_ROLE_KEY en el entorno")
	}

	db := supabase.NewClient(supabaseURL, serviceKey)
	verifStore := verificacion.NewStore()

	// Config desde Firestore (p. ej. URL del Apps Script de correo, campo
	// "url_pstm"). Si no hay credenciales de Firebase, el resto del server
	// sigue funcionando y solo fallará el envío de códigos con un mensaje
	// claro.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	urlProvider, err := firestoreconfig.New(ctx)
	if err != nil {
		log.Printf("aviso: firestore no disponible (%v); el envío de códigos no funcionará", err)
		urlProvider = nil
	}

	h := auth.NewHandler(db, verifStore)
	he := estudiantes.NewHandler(db)
	hv := verificacion.NewHandler(verifStore, db, urlProvider)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("POST /set-password", h.SetPassword)
	mux.HandleFunc("GET /estudiante", he.Buscar)
	mux.HandleFunc("POST /verificacion/enviar", hv.Enviar)
	mux.HandleFunc("POST /verificacion/confirmar", hv.Confirmar)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("noteos-server escuchando en :%s", port)
	if err := http.ListenAndServe(":"+port, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

// withCORS permite llamadas desde Flutter Web durante desarrollo/pruebas.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
