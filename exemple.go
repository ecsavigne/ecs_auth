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
	cl, err := token.NewClientAuth(
		token.WithClientID("492799813423-5acrke2ei1lrcuf5fiqu21ugcgftlcmr.apps.googleusercontent.com"),
		token.WithClientSecret("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"),
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
	go cl.GetToken()

	for {
		select {
		case token := <-cl.Token:
			fmt.Println()
			fmt.Println("Token: ", string(token.BytesToken))
		case url := <-cl.Url:
			fmt.Println("Url: ", url.Url)
			// Use the authorization code that is pushed to the redirec
		}
	}

	fmt.Println("Visit the URL for the auth dialog: ", (<-cl.Url).Url)

}

func main() {
	NewServiceDrive()
}
