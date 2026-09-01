package verificacion

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"noteos-server/internal/firestoreconfig"
	"noteos-server/internal/supabase"
)

type Handler struct {
	Store *Store
	DB    *supabase.Client
	URL   *firestoreconfig.URLProvider
}

func NewHandler(store *Store, db *supabase.Client, urlProvider *firestoreconfig.URLProvider) *Handler {
	return &Handler{Store: store, DB: db, URL: urlProvider}
}

type estudianteRow struct {
	ID                int64  `json:"id"`
	NumeroDocumento   string `json:"numero_documento"`
	CorreoEstudiantil string `json:"correo_estudiantil"`
	CorreoPersonal    string `json:"correo_personal"`
	PrimerIngreso     *bool  `json:"primer_ingreso"`
}

type respuesta struct {
	OK    bool   `json:"ok,omitempty"`
	Error string `json:"error,omitempty"`
}

type enviarRequest struct {
	NumeroDocumento string `json:"numero_documento"`
	CorreoPersonal  string `json:"correo_personal"`
}

// Enviar responde POST /verificacion/enviar. Busca al estudiante por
// numero_documento, confirma que sigue en primer ingreso, y le pide al
// mailer que mande el código al correo personal (si vino en la petición) o,
// si no, al correo institucional.
func (h *Handler) Enviar(w http.ResponseWriter, r *http.Request) {
	var req enviarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, respuesta{Error: "cuerpo inválido"})
		return
	}
	req.NumeroDocumento = strings.TrimSpace(req.NumeroDocumento)
	if req.NumeroDocumento == "" {
		writeJSON(w, http.StatusBadRequest, respuesta{Error: "numero_documento es obligatorio"})
		return
	}

	var rows []estudianteRow
	q := url.Values{
		"numero_documento": {"eq." + req.NumeroDocumento},
		"select":           {"id,numero_documento,correo_estudiantil,correo_personal,primer_ingreso"},
		"limit":            {"1"},
	}
	if err := h.DB.Get("estudiantes", q, &rows); err != nil {
		writeJSON(w, http.StatusInternalServerError, respuesta{Error: "error consultando la base de datos"})
		return
	}
	if len(rows) == 0 {
		writeJSON(w, http.StatusNotFound, respuesta{Error: "no se encontró un estudiante con esa identificación"})
		return
	}
	e := rows[0]
	if e.PrimerIngreso != nil && !*e.PrimerIngreso {
		writeJSON(w, http.StatusConflict, respuesta{Error: "este usuario ya inició sesión antes"})
		return
	}

	// Se exige que haya al menos un medio de contacto: el correo personal que
	// manda el cliente (se guarda de una vez para poder reenviar el código sin
	// que el cliente lo vuelva a mandar) o el correo institucional que ya está
	// en la base de datos. Si no hay ninguno, no se puede enviar el código.
	req.CorreoPersonal = strings.TrimSpace(req.CorreoPersonal)
	if req.CorreoPersonal != "" && req.CorreoPersonal != e.CorreoPersonal {
		patchQ := url.Values{"id": {"eq." + strconv.FormatInt(e.ID, 10)}}
		if err := h.DB.Patch("estudiantes", patchQ, map[string]any{"correo_personal": req.CorreoPersonal}); err != nil {
			writeJSON(w, http.StatusInternalServerError, respuesta{Error: "error consultando la base de datos"})
			return
		}
		e.CorreoPersonal = req.CorreoPersonal
	}

	correoDestino := e.CorreoPersonal
	if correoDestino == "" {
		correoDestino = e.CorreoEstudiantil
	}
	if correoDestino == "" {
		writeJSON(w, http.StatusBadRequest, respuesta{Error: "hace falta un correo personal o institucional para enviar el código"})
		return
	}

	if h.URL == nil {
		writeJSON(w, http.StatusInternalServerError, respuesta{Error: "el servidor no está configurado para enviar códigos"})
		return
	}
	appsScriptURL, err := h.URL.URL(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, respuesta{Error: "no se pudo obtener la URL del correo: " + err.Error()})
		return
	}

	codigo, err := enviarCodigo(appsScriptURL, correoDestino)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, respuesta{Error: "no se pudo enviar el código: " + err.Error()})
		return
	}
	h.Store.Guardar(e.NumeroDocumento, codigo)

	writeJSON(w, http.StatusOK, respuesta{OK: true})
}

type confirmarRequest struct {
	NumeroDocumento string `json:"numero_documento"`
	Codigo          string `json:"codigo"`
}

// Confirmar responde POST /verificacion/confirmar. Si el código coincide y
// no venció, deja al estudiante habilitado para crear su contraseña
// (auth.Handler.SetPassword consulta el mismo Store).
func (h *Handler) Confirmar(w http.ResponseWriter, r *http.Request) {
	var req confirmarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, respuesta{Error: "cuerpo inválido"})
		return
	}
	req.NumeroDocumento = strings.TrimSpace(req.NumeroDocumento)
	req.Codigo = strings.TrimSpace(req.Codigo)
	if req.NumeroDocumento == "" || req.Codigo == "" {
		writeJSON(w, http.StatusBadRequest, respuesta{Error: "numero_documento y codigo son obligatorios"})
		return
	}

	if err := h.Store.Confirmar(req.NumeroDocumento, req.Codigo); err != nil {
		writeJSON(w, http.StatusBadRequest, respuesta{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, respuesta{OK: true})
}

func writeJSON(w http.ResponseWriter, status int, body respuesta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
