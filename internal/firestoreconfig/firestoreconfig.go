// Package firestoreconfig lee desde Cloud Firestore la configuración del
// servidor (p. ej. la URL del Apps Script de correo / "SMTP"), almacenada en
// el documento users/app.
//
// Las credenciales del Service Account pueden venir por dos vías:
//   - variable de entorno FIREBASE_SERVICE_ACCOUNT_JSON (contenido del JSON),
//   - archivo apuntado por GOOGLE_APPLICATION_CREDENTIALS (recomendado por el
//     SDK de Google; Render la inyecta en producción).
package firestoreconfig

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

const (
	coleccion = "users"
	documento = "app"
	campoURL  = "url_pstm"

	envOverride    = "APPS_SCRIPT_URL"
	envSAJSON      = "FIREBASE_SERVICE_ACCOUNT_JSON"
	envSAPath      = "GOOGLE_APPLICATION_CREDENTIALS"
	requisitoOrigen = "APPS_SCRIPT_URL o FIREBASE_SERVICE_ACCOUNT_JSON / GOOGLE_APPLICATION_CREDENTIALS"
)

// URLProvider resuelve la URL del Apps Script de correo, con caché en
// memoria para no consultar Firestore en cada envío.
type URLProvider struct {
	client *firestore.Client
	mu     sync.RWMutex
	url    string
}

// New inicializa el acceso a Cloud Firestore. Si no hay credenciales
// disponibles devuelve un error. ctx debe tener un timeout razonable.
func New(ctx context.Context) (*URLProvider, error) {
	opts, err := clientOptions()
	if err != nil {
		return nil, err
	}
	app, err := firebase.NewApp(ctx, nil, opts...)
	if err != nil {
		return nil, err
	}
	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}
	return &URLProvider{client: client}, nil
}

// clientOptions construye las opciones de autenticación: si hay una
// variable de entorno con el contenido del JSON, se usa; si no, se delega en
// el mecanismo por defecto de Google (GOOGLE_APPLICATION_CREDENTIALS /
// metadata server).
func clientOptions() ([]option.ClientOption, error) {
	if raw := strings.TrimSpace(os.Getenv(envSAJSON)); raw != "" {
		return []option.ClientOption{option.WithCredentialsJSON([]byte(raw))}, nil
	}
	if path := strings.TrimSpace(os.Getenv(envSAPath)); path != "" {
		return nil, nil // el SDK lee el archivo por defecto
	}
	return nil, fmt.Errorf("no hay credenciales de Firebase: define %s", requisitoOrigen)
}

// Close libera el cliente de Firestore.
func (p *URLProvider) Close() error {
	if p == nil || p.client == nil {
		return nil
	}
	return p.client.Close()
}

// URL devuelve la URL del Apps Script de correo, aplicando, en orden:
//  1. APPS_SCRIPT_URL (override por variable de entorno, útil en desarrollo
//     local sin credenciales),
//  2. el valor cacheado en memoria,
//  3. la lectura desde Firestore (users/app -> url_pstm).
func (p *URLProvider) URL(ctx context.Context) (string, error) {
	if env := strings.TrimSpace(os.Getenv(envOverride)); env != "" {
		return env, nil
	}

	p.mu.RLock()
	cached := p.url
	p.mu.RUnlock()
	if cached != "" {
		return cached, nil
	}

	// Lectura desde Firestore con copia fresca del contexto con timeout.
	lecturaCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	snap, err := p.client.Collection(coleccion).Doc(documento).Get(lecturaCtx)
	if err != nil {
		// ante un error transitorio, si ya teníamos una URL cacheada la
		// seguimos usando en lugar de romper el envío.
		p.mu.RLock()
		cached = p.url
		p.mu.RUnlock()
		if cached != "" {
			log.Printf("firestoreconfig: error refrescando url_pstm (%v); usando URL cacheada", err)
			return cached, nil
		}
		return "", fmt.Errorf("leyendo %s/%s: %w", coleccion, documento, err)
	}

	url, _ := snap.Data()[campoURL].(string)
	url = strings.TrimSpace(url)
	if url == "" {
		return "", fmt.Errorf("el campo %s está vacío en %s/%s", campoURL, coleccion, documento)
	}

	p.mu.Lock()
	p.url = url
	p.mu.Unlock()
	return url, nil
}
