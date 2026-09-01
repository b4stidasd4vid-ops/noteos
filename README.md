# noteos-server

Backend de autenticación de NoteOs. Habla con Supabase por su REST API
(PostgREST) usando la `service_role` key — nunca expuesta a la app Flutter.

## Correr en local

```
go mod tidy
go run .
```

Escucha en `http://localhost:8080` (o el `PORT` que definas en `.env`).

## Endpoints

- `POST /login` — `{"usuario": "...", "contrasena": "..."}`
- `POST /set-password` — `{"usuario": "...", "nueva_contrasena": "...", "correo_personal": "..."}` (correo_personal opcional; si viene, se guarda en `estudiantes.correo_personal`)
  (solo válido si el usuario está en primer ingreso)
- `GET /estudiante?numero_documento=...` — busca en `estudiantes` y resuelve
  el nombre del curso vía `curso_id -> cursos.nombre`. Usado por la pantalla
  de registro/pre-login para autocompletar curso y correo institucional.

## Deploy en Render

1. Sube esta carpeta a su propio repo de Git.
2. En Render: New -> Web Service, runtime **Go**, build command
   `go build -o noteos-server .`, start command `./noteos-server`.
3. En "Environment", agrega `SUPABASE_URL` y `SUPABASE_SERVICE_ROLE_KEY`
   (Render define `PORT` solo). No subas el `.env` local — está en
   `.gitignore`.
