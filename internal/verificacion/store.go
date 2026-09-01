// Package verificacion valida códigos de verificación enviados por correo,
// usados para confirmar identidad antes de dejar crear una contraseña en el
// pre-registro (evita que alguien se registre con el número de documento de
// otro estudiante). El código en sí lo genera y envía un servicio externo
// (ver mailer.go); acá solo se guarda cuál fue el código válido y se valida.
package verificacion

import (
	"errors"
	"sync"
	"time"
)

const (
	vigencia           = 3 * time.Minute
	vigenciaVerificado = 5 * time.Minute
)

var (
	ErrNoEnviado  = errors.New("no se ha enviado un código para este usuario")
	ErrExpirado   = errors.New("el código expiró, solicita uno nuevo")
	ErrIncorrecto = errors.New("código incorrecto")
)

type entrada struct {
	codigo     string
	expira     time.Time
	verificado bool
}

// Store guarda en memoria los códigos vigentes, indexados por número de
// documento. No persiste en la base de datos: si el server se reinicia, los
// códigos pendientes se pierden y el estudiante debe pedir uno nuevo.
type Store struct {
	mu       sync.Mutex
	entradas map[string]entrada
}

func NewStore() *Store {
	return &Store{entradas: map[string]entrada{}}
}

// Guardar registra codigo como el código vigente para numeroDocumento,
// reemplazando cualquier código pendiente anterior.
func (s *Store) Guardar(numeroDocumento, codigo string) {
	s.mu.Lock()
	s.entradas[numeroDocumento] = entrada{codigo: codigo, expira: time.Now().Add(vigencia)}
	s.mu.Unlock()
}

// Confirmar valida el código para numeroDocumento. Si es correcto y no
// venció, marca la entrada como verificada (con una vigencia propia, para
// que EstaVerificado siga dando true mientras se completa la creación de
// contraseña) y devuelve nil.
func (s *Store) Confirmar(numeroDocumento, codigo string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.entradas[numeroDocumento]
	if !ok {
		return ErrNoEnviado
	}
	if time.Now().After(e.expira) {
		delete(s.entradas, numeroDocumento)
		return ErrExpirado
	}
	if e.codigo != codigo {
		return ErrIncorrecto
	}

	e.verificado = true
	e.expira = time.Now().Add(vigenciaVerificado)
	s.entradas[numeroDocumento] = e
	return nil
}

// EstaVerificado reporta si numeroDocumento confirmó su código
// recientemente (dentro de vigenciaVerificado).
func (s *Store) EstaVerificado(numeroDocumento string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.entradas[numeroDocumento]
	if !ok || !e.verificado {
		return false
	}
	if time.Now().After(e.expira) {
		delete(s.entradas, numeroDocumento)
		return false
	}
	return true
}

// Limpiar borra la entrada de numeroDocumento (usar tras completar el
// registro, para no dejar un "verificado" reutilizable).
func (s *Store) Limpiar(numeroDocumento string) {
	s.mu.Lock()
	delete(s.entradas, numeroDocumento)
	s.mu.Unlock()
}
