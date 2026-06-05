package main

import (
	"encoding/json"
	"fmt"
	"log"

	token "github.com/ecsavigne/ecs_auth/clientAuth"
	"github.com/joho/godotenv"

	"github.com/ecsavigne/ecs_socket/client"
	"github.com/ecsavigne/ecs_socket/socket_type"
)

func receiveMessage(clAuth *token.ClientAuth) {
	// codeCh := clAuth.Code
	cl := client.NewClient(socket_type.CConfig{
		Url: "oficial.crmsocialhub.com.br/wsCode",
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

		// fmt.Println("CodeStrMap: ", mapCode)
		if code != "" {
			break
		} else {
			fmt.Println("Code not found")
			code = ""
			return
		}
	}

	clAuth.SetCode(code)
}

func NewServiceDrive() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error al cargar el archivo .env")
	}

	cl, err := token.NewClientAuth(
		token.WithClientID("765398235585526"),
		token.WithClientSecret("7520958c0de34fdbddbdce0ed7822293"),
		token.WithScopes([]string{"whatsapp_business_messaging", "whatsapp_business_management", "whatsapp_business_management"}),
		token.WithRedirectURL("https://oficial.crmsocialhub.com.br/sh/oauth2/facebook/callback"),
		// token.WithAuthURL("https://www.facebook.com/v24.0/dialog/oauth"),
		token.WithTokenURL("https://graph.facebook.com/v24.0/oauth/access_token"),
	)
	// cl, err := token.NewClientAuth(
	// 	token.WithClientID(os.Getenv("CLIENT_ID")),
	// 	token.WithClientSecret(os.Getenv("CLIENT_SECRET")),
	// 	token.WithScopes([]string{
	// 		drive.DriveScope, drive.DriveAppdataScope,
	// 		drive.DriveMetadataScope, drive.DriveMetadataReadonlyScope,
	// 	}),
	// 	token.WithRedirectURL("https://oficial.crmsocialhub.com.br/sh/oauth2/driver/callback"),
	// 	token.WithAuthURL("https://accounts.google.com/o/oauth2/auth"),
	// 	token.WithTokenURL("https://oauth2.googleapis.com/token"),
	// )
	if err != nil {
		log.Fatal(err)
	}

	cl.SetCode("AQD80PLJZEpCwNoqxWRvb4rurRR-46rIlZrJ8JMPWsdyHgR2E12kGq0V6z3W1-lnDd-R_4qwitBqU4ywNCfDuMmfzBbKUiC87mC2RhoAQUE8ax4bhunhFipm3uXj_t8Ob9ZAgMD8QKu-Pwqx2qZ2oY-X0S2eiL8YlnjjzfmlIJrzA6QSoIgd-liAtBdQWPREAoEGqGnd4NFnBEW-N3Xq5vdEMdAG37CCfEd-fKuoNmhg3V-WTUC4EQ-wvSDaxsOCId0CgLoD3TWFGSuHya5MXPabl6xFKAFk6Umg9ltNnrS_mKh2CBaPja0HIpVRmpCPyMAHBhcG9UXbvKOPOT4BAQTRCGVg-4PcZq_ntqyCjc2n2z6YeQqfPrR6nIhimFUVtRT8_Uftby5xU0Q2Hqu2g-wMnVQ6r9XZVcPJI0z3hC6weT0r2q-lhhixMEz7LOkFj2uWtbRXPIRHEXLKI6lsWTY5")

	go receiveMessage(cl)
	go cl.GetToken(false)

	for {
		select {
		case token := <-cl.Token:
			fmt.Println()
			fmt.Println("Token: ", string(token.BytesToken))
			return
		case url := <-cl.Url:
			fmt.Println("Url: ", url.Url)
			// Use the authorization code that is pushed to the redirec
		}
	}
}

func main() {
	NewServiceDrive()
}
