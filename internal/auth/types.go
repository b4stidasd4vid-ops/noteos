package auth

import "time"

// Estudiante refleja las columnas de auth agregadas a public.estudiantes.
// El login de estudiantes ya no usa una tabla "usuarios" separada: el
// nombre o el correo institucional actúan como identificador.
type Estudiante struct {
	ID                int64      `json:"id"`
	Nombre            string     `json:"nombre"`
	NumeroDocumento   string     `json:"numero_documento"`
	CorreoEstudiantil string     `json:"correo_estudiantil"`
	ContrasenaHash    *string    `json:"contraseña_hash"`
	PrimerIngreso     *bool      `json:"primer_ingreso"`
	IntentosFallidos  *int       `json:"intentos_fallidos"`
	TiempoBloqueo     *time.Time `json:"tiempo_bloqueo"`
}

type LoginRequest struct {
	Usuario    string `json:"usuario"`
	Contrasena string `json:"contrasena"`
}

type LoginResponse struct {
	OK                bool   `json:"ok,omitempty"`
	PrimerIngreso     bool   `json:"primer_ingreso,omitempty"`
	Rol               string `json:"rol,omitempty"`
	Nombre            string `json:"nombre,omitempty"`
	EstudianteID      *int64 `json:"estudiante_id,omitempty"`
	Error             string `json:"error,omitempty"`
	IntentosRestantes int    `json:"intentos_restantes,omitempty"`
	ReintentaEnSeg    int64  `json:"reintenta_en_segundos,omitempty"`
}

type SetPasswordRequest struct {
	Usuario         string `json:"usuario"`
	NuevaContrasena string `json:"nueva_contrasena"`
	CorreoPersonal  string `json:"correo_personal"`
}

type SetPasswordResponse struct {
	OK    bool   `json:"ok,omitempty"`
	Error string `json:"error,omitempty"`
}
