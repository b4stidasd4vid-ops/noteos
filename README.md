# noteos-server

Backend de autenticación de NoteOs. Habla con Supabase por su REST API
(PostgREST) usando la `service_role` key — nunca expuesta a la app Flutter.
También lee configuración (p. ej. la URL del Apps Script de correo / "SMTP")
desde Cloud Firestore.

## Correr en local

```
go mod tidy
go run .
```

Escucha en `http://localhost:8080` (o el `PORT` que definas en `.env`).

Para que el envío de códigos funcione en local sin tocar Firebase, define la
variable `APPS_SCRIPT_URL` con la URL del Web App de Google Apps Script. Sin
ella, el server intentará leer `url_pstm` desde Firestore (requiere las
credenciales de Firebase).

## Endpoints

- `POST /login` — `{"usuario": "...", "contrasena": "..."}`
- `POST /set-password` — `{"usuario": "...", "nueva_contrasena": "...", "correo_personal": "..."}`
  (correo_personal opcional; si viene, se guarda en `estudiantes.correo_personal`; solo válido si el usuario está en primer ingreso y verificó su código)
- `GET /estudiante?numero_documento=...` — busca en `estudiantes` y resuelve
  el nombre del curso vía `curso_id -> cursos.nombre`. Usado por la pantalla
  de registro/pre-login para autocompletar curso y correo institucional.
- `POST /verificacion/enviar` — `{"numero_documento": "...", "correo_personal": "..."}`.
  Envía un código de verificación. Exige al menos un correo de contacto: el
  `correo_personal` que manda el cliente o el `correo_estudiantil` que ya está
  en la BD. El código se manda al personal si existe, si no al institucional.
- `POST /verificacion/confirmar` — `{"numero_documento": "...", "codigo": "..."}`.
  Valida el código y habilita la creación de contraseña.

## Configuración de Firebase (leer url_pstm desde Firestore)

El servidor NO tiene la URL del Apps Script fija en el código. La lee en tiempo
de ejecución desde Cloud Firestore: documento `users/app`, campo `url_pstm`.
Documento de ejemplo:

```
users/app -> { "url_pstm": "https://script.google.com/macros/s/..." }
```

El acceso a Firestore se autentica con el Service Account de Firebase por una
de estas vías:

- Variable de entorno `FIREBASE_SERVICE_ACCOUNT_JSON` con el contenido del JSON
  del Service Account (recomendado en Render).
- O la variable `GOOGLE_APPLICATION_CREDENTIALS` apuntando a un archivo JSON en
  disco (usado por el SDK de Google / en entornos con filesystem persistente).

Nunca subas el JSON del Service Account al repo: está en `.gitignore`.

## Deploy en Render

1. Sube esta carpeta a su propio repo de Git.
2. En Render: New -> Web Service, runtime **Go**, build command
   `go build -o noteos-server .`, start command `./noteos-server`.
3. En "Environment", agrega `SUPABASE_URL` y `SUPABASE_SERVICE_ROLE_KEY`
   (Render define `PORT` solo). No subas el `.env` local — está en
   `.gitignore`.
4. Si quieres que el envío de códigos funcione leyendo `url_pstm` desde
   Firestore, agrega también `FIREBASE_SERVICE_ACCOUNT_JSON` (contenido del
   Service Account) y crea el documento `users/app` con el campo `url_pstm`
   en tu Firestore.
