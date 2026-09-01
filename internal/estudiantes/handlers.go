// Package estudiantes expone la consulta de datos de un estudiante por su
// número de documento, usada por la pantalla de registro/pre-login de la app
// para autocompletar curso y correo institucional.
package estudiantes

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"noteos-server/internal/supabase"
)

type Handler struct {
	DB *supabase.Client
}

func NewHandler(db *supabase.Client) *Handler {
	return &Handler{DB: db}
}

type estudianteRow struct {
	Nombre            string `json:"nombre"`
	NumeroDocumento   string `json:"numero_documento"`
	CursoID           *int64 `json:"curso_id"`
	CorreoEstudiantil string `json:"correo_estudiantil"`
	CorreoPersonal    string `json:"correo_personal"`
	PrimerIngreso     *bool  `json:"primer_ingreso"`
}

type cursoRow struct {
	Nombre string `json:"nombre"`
}

type Response struct {
	Nombre            string `json:"nombre,omitempty"`
	NumeroDocumento   string `json:"numero_documento,omitempty"`
	Curso             string `json:"curso,omitempty"`
	CorreoEstudiantil string `json:"correo_estudiantil,omitempty"`
	Error             string `json:"error,omitempty"`
}

// Buscar responde GET /estudiante?numero_documento=... con el nombre, el
// nombre del curso (resuelto vía estudiantes.curso_id -> cursos.nombre) y el
// correo institucional del estudiante.
func (h *Handler) Buscar(w http.ResponseWriter, r *http.Request) {
	numeroDocumento := strings.TrimSpace(r.URL.Query().Get("numero_documento"))
	if numeroDocumento == "" {
		writeJSON(w, http.StatusBadRequest, Response{Error: "numero_documento es obligatorio"})
		return
	}

	var estudiantes []estudianteRow
	query := url.Values{
		"numero_documento": {"eq." + numeroDocumento},
		"select":           {"nombre,numero_documento,curso_id,correo_estudiantil,primer_ingreso"},
		"limit":            {"1"},
	}
	if err := h.DB.Get("estudiantes", query, &estudiantes); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Error: "error consultando la base de datos"})
		return
	}
	if len(estudiantes) == 0 {
		writeJSON(w, http.StatusNotFound, Response{Error: "no se encontró un estudiante con esa identificación"})
		return
	}
	e := estudiantes[0]

	// Ya tiene contraseña creada: no es su primer ingreso, así que no se deja
	// pasar por el pre-registro (evita que otra persona se "re-registre" con
	// el documento de alguien que ya tiene cuenta).
	if e.PrimerIngreso != nil && !*e.PrimerIngreso {
		writeJSON(w, http.StatusConflict, Response{Error: "este usuario ya inició sesión antes; no puedes volver a registrarte"})
		return
	}

	curso := ""
	if e.CursoID != nil {
		var cursos []cursoRow
		cq := url.Values{
			"id":     {"eq." + strconv.FormatInt(*e.CursoID, 10)},
			"select": {"nombre"},
			"limit":  {"1"},
		}
		if err := h.DB.Get("cursos", cq, &cursos); err == nil && len(cursos) > 0 {
			curso = cursos[0].Nombre
		}
	}

	writeJSON(w, http.StatusOK, Response{
		Nombre:            e.Nombre,
		NumeroDocumento:   e.NumeroDocumento,
		Curso:             curso,
		CorreoEstudiantil: e.CorreoEstudiantil,
	})
}

func writeJSON(w http.ResponseWriter, status int, body Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
