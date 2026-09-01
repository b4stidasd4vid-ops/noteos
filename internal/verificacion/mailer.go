package verificacion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// appsScriptURL apunta al Web App de Google Apps Script que manda el correo
// (usa GmailApp.sendEmail). Es una solución temporal para probar el envío
// mientras se monta el servicio definitivo.
const appsScriptURL = "https://script.google.com/macros/s/AKfycbywif72Rjxm2WKYEG47kJeAHlKUreL5_sORgM927F2001MuuxQBeqWVWAV8ezGElfEv/exec"

type appsScriptResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	PinEnviado string `json:"pinEnviado"`
}

// enviarCodigo le pide al Apps Script que genere y mande un código al
// correo dado, y devuelve el código que efectivamente envió (el propio
// script lo genera; acá solo lo recibimos para poder validarlo después).
func enviarCodigo(correo string) (string, error) {
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
