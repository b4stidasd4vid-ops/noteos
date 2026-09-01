package verificacion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type appsScriptResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	PinEnviado string `json:"pinEnviado"`
}

// enviarCodigo le pide al Web App de Google Apps Script (appsScriptURL) que
// genere y mande un código al correo dado, y devuelve el código que
// efectivamente envió (el propio script lo genera; acá solo lo recibimos para
// poder validarlo después). La URL no está fija en el código: llega desde
// Firestore (config/servidor -> url_pstm) y se pasa como parámetro.
func enviarCodigo(appsScriptURL, correo string) (string, error) {
	payload, err := json.Marshal(map[string]string{"correo": correo})
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(appsScriptURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out appsScriptResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Status != "success" || out.PinEnviado == "" {
		return "", fmt.Errorf("apps script: %s", out.Message)
	}
	return out.PinEnviado, nil
}
