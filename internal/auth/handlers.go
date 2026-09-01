package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"noteos-server/internal/supabase"
	"noteos-server/internal/verificacion"
)

type Handler struct {
	DB         *supabase.Client
	VerifStore *verificacion.Store
}

func NewHandler(db *supabase.Client, verifStore *verificacion.Store) *Handler {
	return &Handler{DB: db, VerifStore: verifStore}
}

// buscarEstudiante trae la fila de estudiantes cuyo nombre o correo
// institucional coincide con "usuario" (el login acepta cualquiera de los
// dos en un mismo campo), o nil si no existe ninguna.
func (h *Handler) buscarEstudiante(usuario string) (*Estudiante, error) {
	var rows []Estudiante
	query := url.Values{
		"or":     {"(nombre.eq." + usuario + ",correo_estudiantil.eq." + usuario + ")"},
		"select": {"*"},
		"limit":  {"1"},
	}
	if err := h.DB.Get("estudiantes", query, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, LoginResponse{Error: "cuerpo inválido"})
		return
	}
	req.Usuario = strings.TrimSpace(req.Usuario)
	if req.Usuario == "" || req.Contrasena == "" {
		writeJSON(w, http.StatusBadRequest, LoginResponse{Error: "usuario y contraseña son obligatorios"})
		return
	}

	e, err := h.buscarEstudiante(req.Usuario)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, LoginResponse{Error: "error consultando la base de datos"})
		return
	}
	if e == nil {
		writeJSON(w, http.StatusUnauthorized, LoginResponse{Error: "credenciales inválidas"})
		return
	}

	if bloqueado, restante := bloqueada(e); bloqueado {
		writeJSON(w, http.StatusLocked, LoginResponse{
			Error:          "cuenta bloqueada por demasiados intentos fallidos",
			ReintentaEnSeg: int64(restante.Seconds()),
		})
		return
	}

	if e.PrimerIngreso == nil || *e.PrimerIngreso {
		writeJSON(w, http.StatusOK, LoginResponse{PrimerIngreso: true, Rol: "estudiante"})
		return
	}

	if e.ContrasenaHash == nil || bcrypt.CompareHashAndPassword([]byte(*e.ContrasenaHash), []byte(req.Contrasena)) != nil {
		restantes, err := registrarIntentoFallido(h.DB, e)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, LoginResponse{Error: "error consultando la base de datos"})
			return
		}
		if restantes <= 0 {
			writeJSON(w, http.StatusLocked, LoginResponse{
				Error:          "cuenta bloqueada por demasiados intentos fallidos",
				ReintentaEnSeg: int64(bloqueoDuracion.Seconds()),
			})
			return
		}
		writeJSON(w, http.StatusUnauthorized, LoginResponse{
			Error:             "credenciales inválidas",
			IntentosRestantes: restantes,
		})
		return
	}

	if err := resetIntentos(h.DB, e); err != nil {
		writeJSON(w, http.StatusInternalServerError, LoginResponse{Error: "error consultando la base de datos"})
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		OK:           true,
		Rol:          "estudiante",
		Nombre:       e.Nombre,
		EstudianteID: &e.ID,
	})
}

func (h *Handler) SetPassword(w http.ResponseWriter, r *http.Request) {
	var req SetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SetPasswordResponse{Error: "cuerpo inválido"})
		return
	}
	req.Usuario = strings.TrimSpace(req.Usuario)
	if len(req.NuevaContrasena) < 6 {
		writeJSON(w, http.StatusBadRequest, SetPasswordResponse{Error: "la contraseña debe tener al menos 6 caracteres"})
		return
	}

	req.CorreoPersonal = strings.TrimSpace(req.CorreoPersonal)
	if req.CorreoPersonal != "" && !correoValido(req.CorreoPersonal) {
		writeJSON(w, http.StatusBadRequest, SetPasswordResponse{Error: "el correo personal no es válido"})
		return
	}

	e, err := h.buscarEstudiante(req.Usuario)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SetPasswordResponse{Error: "error consultando la base de datos"})
		return
	}
	if e == nil {
		writeJSON(w, http.StatusUnauthorized, SetPasswordResponse{Error: "credenciales inválidas"})
		return
	}
	if e.PrimerIngreso != nil && !*e.PrimerIngreso {
		writeJSON(w, http.StatusBadRequest, SetPasswordResponse{Error: "este usuario ya tiene contraseña"})
		return
	}

	if h.VerifStore == nil || !h.VerifStore.EstaVerificado(e.NumeroDocumento) {
		writeJSON(w, http.StatusForbidden, SetPasswordResponse{Error: "primero debes verificar el código enviado a tu correo"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NuevaContrasena), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SetPasswordResponse{Error: "error generando la contraseña"})
		return
	}

	query := url.Values{"id": {"eq." + strconv.FormatInt(e.ID, 10)}}
	fields := map[string]any{
		"contraseña_hash": string(hash),
		"primer_ingreso":  false,
	}
	if req.CorreoPersonal != "" {
		fields["correo_personal"] = req.CorreoPersonal
	}
	if err := h.DB.Patch("estudiantes", query, fields); err != nil {
		writeJSON(w, http.StatusInternalServerError, SetPasswordResponse{Error: "error consultando la base de datos"})
		return
	}
	h.VerifStore.Limpiar(e.NumeroDocumento)

	writeJSON(w, http.StatusOK, SetPasswordResponse{OK: true})
}

// correoValido reporta si s tiene formato básico de correo
// (algo@dominio.tld). No intenta validar el dominio real.
func correoValido(s string) bool {
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	dominio := s[at+1:]
	punto := strings.LastIndexByte(dominio, '.')
	return punto > 0 && punto < len(dominio)-1 && !strings.ContainsAny(s, " \t")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
