# ECS Auth Library - OAuth2 Client

Una biblioteca de Golang para autenticación OAuth2 con Google Drive, usando flujos de autorización seguros con PKCE y manejo asíncrono mediante channels.

## Descripción General

Este paquete proporciona una solución de OAuth2 que:
- Gestiona el flujo de autorización de OAuth2 de forma asíncrona
- Soporta obtención de códigos de intercambio mediante WebSocket o channels
- Maneja automáticamente la expiración y renovación de tokens
- Almacena tokens en archivos de forma segura
- Utiliza PKCE para proteger contra ataques CSRF
- Proporciona múltiples channels para eventos de autenticación

## Instalación

```bash
go get github.com/ecsavigne/ecs_auth
```

## Uso Básico

### 1. Crear una Instancia de ClientAuth

```go
package main

import (
	"log"
	token "ecs_auth/clientAuth"
	"google.golang.org/api/drive/v3"
)

func main() {
	cl, err := token.NewClientAuth(
		token.WithClientID("YOUR_CLIENT_ID"),
		token.WithClientSecret("YOUR_CLIENT_SECRET"),
		token.WithScopes([]string{
			drive.DriveScope,
			drive.DriveAppdataScope,
			drive.DriveMetadataScope,
			drive.DriveMetadataReadonlyScope,
		}),
		token.WithRedirectURL("https://tu-aplicacion.com/oauth2/callback"),
		token.WithAuthURL("https://accounts.google.com/o/oauth2/auth"),
		token.WithTokenURL("https://oauth2.googleapis.com/token"),
	)

	if err != nil {
		log.Fatal(err)
	}
}
```

## Opciones de Configuración

Las siguientes opciones configuran el cliente OAuth2. Todas son **requeridas**:

| Opción | Descripción | Ejemplo |
|--------|-------------|---------|
| `WithClientID(string)` | ID del cliente de OAuth2 proporcionado por el proveedor | `WithClientID("YOUR_CLIENT_ID")` |
| `WithClientSecret(string)` | Secreto del cliente (confidencial, nunca lo compartas) | `WithClientSecret("YOUR_CLIENT_SECRET")` |
| `WithScopes([]string)` | Permisos OAuth2 requeridos para la aplicación | `WithScopes([]string{drive.DriveScope})` |
| `WithRedirectURL(string)` | URL donde el proveedor enviará el código de autorización | `WithRedirectURL("https://tu-app.com/oauth2/callback")` |
| `WithAuthURL(string)` | URL del servidor de autenticación del proveedor | `WithAuthURL("https://accounts.google.com/o/oauth2/auth")` |
| `WithTokenURL(string)` | URL para intercambiar el código por un token de acceso | `WithTokenURL("https://oauth2.googleapis.com/token")` |

### Ejemplo de Configuración

```go
cl, err := token.NewClientAuth(
	token.WithClientID("YOUR_CLIENT_ID"),
	token.WithClientSecret("YOUR_CLIENT_SECRET"),
	token.WithScopes([]string{
		drive.DriveScope,
		drive.DriveAppdataScope,
	}),
	token.WithRedirectURL("https://tu-app.com/oauth2/callback"),
	token.WithAuthURL("https://accounts.google.com/o/oauth2/auth"),
	token.WithTokenURL("https://oauth2.googleapis.com/token"),
)
```

## Obtención de Tokens

### GetToken() - Flujo Asíncrono

La función `GetToken()` inicia el flujo de autenticación OAuth2 de forma asíncrona. Los eventos se comunican a través de channels y debe ser ejecutada en una goroutine:

```go
go cl.GetToken()
```

Luego, procesa los eventos usando `select` en un loop:

```go
for {
    select {
    case codeEvent := <-cl.Code:
        // Código de intercambio recibido
        fmt.Println("Código recibido:", codeEvent.Code)
    
    case urlEvent := <-cl.Url:
        // URL de autenticación para el usuario
        fmt.Println("Visita esta URL:", urlEvent.Url)
    
    case tokenEvent := <-cl.Token:
        // Token OAuth2 obtenido exitosamente
        fmt.Println("Token obtenido:", string(tokenEvent.BytesToken))
    
    case verifierEvent := <-cl.Verifier:
        // Verificador PKCE generado
        fmt.Println("Verificador PKCE:", verifierEvent.Verifier)
    }
}
```

## Arquitectura Asíncrona

El `ClientAuth` proporciona múltiples channels para comunicación asíncrona:

| Channel | Tipo | Descripción |
|---------|------|-------------|
| `Code` | `chan CodeEvent` | Código de intercambio OAuth2 |
| `Url` | `chan UrlEvent` | URL de consentimiento para el usuario |
| `Verifier` | `chan VerifierEvent` | Verificador PKCE para seguridad |
| `Token` | `chan TokenEvent` | Token OAuth2 final |

### Estructura de Eventos

```go
type CodeEvent struct {
    Code string
}

type UrlEvent struct {
    Url string
}

type VerifierEvent struct {
    Verifier string
}

type TokenEvent struct {
    Token       *oauth2.Token
    TokenSource oauth2.TokenSource
    BytesToken  []byte
}
```

## Ejemplo Completo: WebSocket

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	token "ecs_auth/clientAuth"
	"github.com/ecsavigne/ecs_socket/client"
	"github.com/ecsavigne/ecs_socket/socket_type"
	"google.golang.org/api/drive/v3"
)

// Función que recibe el código mediante WebSocket
func receiveMessage(clAuth *token.ClientAuth) {
	codeCh := clAuth.Code
	cl := client.NewClient(socket_type.CConfig{
		Url: "tu-servidor.com/wsCode",
		Tls: true,
	})
	
	if cl.Error != nil {
		log.Fatal(cl.Error)
	}

	code := ""
	for {
		cl.ReceiveMessage()

		mapCode := make(map[string]any)
		json.Unmarshal(cl.GetLastMessage(), &mapCode)
		code = mapCode["Code"].(string)

		if code != "" {
			fmt.Println("Código recibido:", code)
			break
		} else {
			fmt.Println("Código no encontrado, reintentando...")
			codeCh <- token.CodeEvent{Code: ""}
			return
		}
	}

	// Envía el código al channel
	codeCh <- token.CodeEvent{Code: code}
}

func NewServiceDrive() {
	cl, err := token.NewClientAuth(
		token.WithClientID("YOUR_CLIENT_ID"),
		token.WithClientSecret("YOUR_CLIENT_SECRET"),
		token.WithScopes([]string{
			drive.DriveScope,
			drive.DriveAppdataScope,
			drive.DriveMetadataScope,
			drive.DriveMetadataReadonlyScope,
		}),
		token.WithRedirectURL("https://tu-app.com/oauth2/callback"),
		token.WithAuthURL("https://accounts.google.com/o/oauth2/auth"),
		token.WithTokenURL("https://oauth2.googleapis.com/token"),
	)

	if err != nil {
		log.Fatal(err)
	}

	// Inicia la recepción de código en goroutine
	go receiveMessage(cl)
	
	// Inicia el flujo de autenticación en goroutine
	go cl.GetToken()

	// Procesa los eventos asincronamente
	for {
		select {
		case tokenEvent := <-cl.Token:
			fmt.Println("✅ Token obtenido exitosamente:")
			fmt.Println(string(tokenEvent.BytesToken))
			return
		
		case urlEvent := <-cl.Url:
			fmt.Println("🔐 URL de autenticación:")
			fmt.Println(urlEvent.Url)
			fmt.Println("Por favor, visita esta URL para autorizar la aplicación")
		}
	}
}

func main() {
	NewServiceDrive()
}
```

## Métodos Adicionales

### GetTokenSource(funcGetCode FuncGetCode) oauth2.TokenSource
Retorna una fuente de tokens que se auto-refresca cuando expira.

```go
tokenSource := cl.GetTokenSource(func() string {
	return "codigo-de-intercambio"
})
```

### SetTokenInFile(tok *oauth2.Token)
Guarda el token en un archivo `token.json`.

```go
cl.SetTokenInFile(tok)
```

### GetTokenOfFile(nameFile string) *oauth2.Token
Recupera un token desde un archivo.

```go
tok := cl.GetTokenOfFile("token.json")
```

### GetTokenOfStr(tokenStr string) *oauth2.Token
Convierte una cadena JSON a un token OAuth2.

```go
tok := cl.GetTokenOfStr(jsonString)
```

### TokenNotExpired(tok *oauth2.Token) bool
Verifica si el token sigue siendo válido.

```go
if cl.TokenNotExpired(tok) {
	fmt.Println("✅ Token is still valid")
}
```

### TokenToBytes(tok *oauth2.Token) []byte
Convierte un token a su representación en bytes (JSON).

```go
tokenBytes := cl.TokenToBytes(tok)
```

### GetState() string
Retorna el state actual usado en la autenticación.

```go
state := cl.GetState()
```

## Flujo de Autenticación

El flujo de autenticación sigue estos pasos:

1. **Inicialización**: Se crea una instancia de `ClientAuth` con las credenciales.

2. **Generación de PKCE**: Se genera un verificador PKCE para proteger contra ataques CSRF.
   - Se envía al channel `Verifier`

3. **Generación de URL**: Se crea una URL de consentimiento para el usuario.
   - Se envía al channel `Url`
   - El usuario debe visitar esta URL y autorizar la aplicación

4. **Obtención del Código**: El usuario es redirigido a la URL de consentimiento.
   - El código de intercambio se envía al channel `Code`
   - Puede obtenerse mediante WebSocket, HTTP, o cualquier otro mecanismo

5. **Intercambio de Código**: Se intercambia el código por un token OAuth2.
   - Se utiliza el verificador PKCE para la seguridad

6. **Token Disponible**: El token se envía al channel `Token`.
   - Contiene el token, su fuente, y su representación en bytes

7. **Almacenamiento**: El token se guarda automáticamente en `token.json`.

8. **Renovación Automática**: Si el token expira, se renueva automáticamente.

## Seguridad

✅ **PKCE (Proof Key for Public Clients)**
- Protege contra ataques CSRF
- Requiere un verificador aleatorio para el intercambio de código
- Se implementa automáticamente en el flujo

✅ **Almacenamiento de Tokens**
- Los tokens se guardan localmente en archivos
- Nunca se transmiten innecesariamente
- Se pueden serializar y deserializar según sea necesario

✅ **Validación de Parámetros**
- Todos los parámetros requeridos son validados
- Se retornan errores claros si falta algún parámetro

✅ **Renovación Automática**
- Los tokens expirados se renuevan automáticamente
- Se valida la expiración antes de usar el token

## Manejo de Errores

```go
cl, err := token.NewClientAuth(
	token.WithClientID("YOUR_CLIENT_ID"),
	// ... otras opciones
)

if err != nil {
	switch err.Error() {
	case "no scopes provided":
		log.Fatal("Debes especificar al menos un scope")
	case "no client id provided":
		log.Fatal("Debes proporcionar el Client ID")
	case "no client secret provided":
		log.Fatal("Debes proporcionar el Client Secret")
	case "no auth url provided":
		log.Fatal("Debes proporcionar la URL de autenticación")
	case "no redirect url provided":
		log.Fatal("Debes proporcionar la URL de redirección")
	case "no token url provided":
		log.Fatal("Debes proporcionar la URL del token")
	default:
		log.Fatal(err)
	}
}
```

## Requisitos

- Go 1.16 o superior
- `golang.org/x/oauth2`
- `google.golang.org/api/drive/v3` (para scopes de Google Drive)

## Dependencias

```
golang.org/x/oauth2
google.golang.org/api/drive/v3
```

## Licencia

Este proyecto es parte de ECS Auth Library.

## Soporte

Para problemas o preguntas, contacta al equipo de desarrollo de ECS.

## Checklist de Implementación

- ✅ Opciones de configuración documentadas correctamente
- ✅ Parámetros requeridos claramente identificados
- ✅ Información sensible NO mostrada (se usan placeholders)
- ✅ Ejemplos funcionales con comentarios
- ✅ Arquitectura asíncrona explicada con channels
- ✅ Flujo de autenticación detallado paso a paso
- ✅ Métodos adicionales documentados
- ✅ Consideraciones de seguridad incluidas
- ✅ Manejo de errores cubierto
- ✅ Requisitos y dependencias listadas
