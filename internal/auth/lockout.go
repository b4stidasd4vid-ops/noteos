package auth

import (
	"net/url"
	"strconv"
	"time"

	"noteos-server/internal/supabase"
)

const (
	maxIntentos     = 3
	bloqueoDuracion = 5 * time.Minute
)

// bloqueada reporta si el estudiante sigue bloqueado en este instante. Si el
// bloqueo ya venció, lo trata como si no existiera (el siguiente intento lo
// limpia solo, al pasar por registrarIntentoFallido/resetIntentos).
func bloqueada(e *Estudiante) (bool, time.Duration) {
	if e.TiempoBloqueo == nil {
		return false, 0
	}
	restante := time.Until(*e.TiempoBloqueo)
	if restante <= 0 {
		return false, 0
	}
	return true, restante
}

// registrarIntentoFallido suma un intento fallido y, si llega al máximo,
// bloquea la cuenta por bloqueoDuracion. Devuelve los intentos restantes
// antes del bloqueo (0 si quedó bloqueada).
func registrarIntentoFallido(client *supabase.Client, e *Estudiante) (int, error) {
	actuales := 0
	if e.IntentosFallidos != nil {
		actuales = *e.IntentosFallidos
	}
	intentos := actuales + 1
	fields := map[string]any{"intentos_fallidos": intentos}

	restantes := maxIntentos - intentos
	if intentos >= maxIntentos {
		fields["tiempo_bloqueo"] = time.Now().Add(bloqueoDuracion)
		fields["intentos_fallidos"] = 0
		restantes = 0
	}

	query := url.Values{"id": {"eq." + strconv.FormatInt(e.ID, 10)}}
	if err := client.Patch("estudiantes", query, fields); err != nil {
		return 0, err
	}
	return restantes, nil
}

// resetIntentos limpia el contador de intentos y cualquier bloqueo, típico
// tras un login exitoso.
func resetIntentos(client *supabase.Client, e *Estudiante) error {
	query := url.Values{"id": {"eq." + strconv.FormatInt(e.ID, 10)}}
	return client.Patch("estudiantes", query, map[string]any{
		"intentos_fallidos": 0,
		"tiempo_bloqueo":    nil,
	})
}
