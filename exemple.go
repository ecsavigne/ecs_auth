package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	token "github.com/ecsavigne/ecs_auth/clientAuth"
	"google.golang.org/api/drive/v3"

	"github.com/ecsavigne/ecs_socket/client"
	"github.com/ecsavigne/ecs_socket/socket_type"
)

func receiveMessage(clAuth *token.ClientAuth) {
	codeCh := clAuth.Code
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
			codeCh <- token.CodeEvent{Code: ""}
			return
		}
	}

	codeCh <- token.CodeEvent{Code: code}
}

func NewServiceDrive() {
	// cl, err := token.NewClientAuth(
	// 	token.WithClientID(""),
	// 	token.WithClientSecret(""),
	// 	token.WithScopes([]string{"whatsapp_business_messaging", "whatsapp_business_management"}),
	// 	token.WithRedirectURL("https://oficial.crmsocialhub.com.br/sh/oauth2/facebook/callback"),
	// 	token.WithAuthURL("https://www.facebook.com/v24.0/dialog/oauth"),
	// 	token.WithTokenURL("https://graph.facebook.com/v24.0/oauth/access_token"),
	// )
	cl, err := token.NewClientAuth(
		token.WithClientID(os.Getenv("CLIENt_ID")),
		token.WithClientSecret(os.Getenv("CLIENT_SECRET")),
		token.WithScopes([]string{
			drive.DriveScope, drive.DriveAppdataScope,
			drive.DriveMetadataScope, drive.DriveMetadataReadonlyScope,
		}),
		token.WithRedirectURL("https://oficial.crmsocialhub.com.br/sh/oauth2/driver/callback"),
		token.WithAuthURL("https://accounts.google.com/o/oauth2/auth"),
		token.WithTokenURL("https://oauth2.googleapis.com/token"),
	)

	if err != nil {
		log.Fatal(err)
	}

	go receiveMessage(cl)
	go cl.GetToken(true)

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
