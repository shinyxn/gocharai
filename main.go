package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/harmony-ai-solutions/CharacterAI-Golang/cai"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var (
	client    *whatsmeow.Client
	caiClient *cai.GoCAI
	token     string = "yourtokenhere"
	character string = "yourcharacteridhere"
	isPlus    bool   = false
)

func eventHandler(evt interface{}) {
	// mundut chat data
	chatData, errChat := caiClient.Chat.GetChat(character)
	if errChat != nil {
		if strings.Contains(errChat.Error(), "404") {
			// chat kalih ai dereng wonten, ndamel rumiyin
			chatData, errChat = caiClient.Chat.NewChat(character)
			if errChat != nil {
				fmt.Println(fmt.Errorf("unable to create chat, error: %q", errChat))
				os.Exit(3)
			}
		} else {
			fmt.Println(fmt.Errorf("unable to fetch chat data, error: %q", errChat))
			os.Exit(2)
		}
	}
	// ningali partisipan ai
	var aiParticipant *cai.ChatParticipant
	for _, participant := range chatData.Participants {
		if !participant.IsHuman {
			aiParticipant = participant
			break
		}
	}

	switch v := evt.(type) {
	case *events.Message:
		if !v.Info.IsFromMe && !v.Info.IsGroup {
			if v.Message.GetConversation() != "" {
				textmsg := fmt.Sprintf("%s: %s", v.Info.PushName, v.Message.GetConversation())
				fmt.Println("-- User --", textmsg)

				messageResult, errMessage := caiClient.Chat.SendMessage(chatData.ExternalID, aiParticipant.User.Username, textmsg, nil)
				if errMessage != nil {
					fmt.Println(fmt.Errorf("unable to send message. Error: %v", errMessage))
				}

				if len(messageResult.Replies) > 0 {
					firstReply := messageResult.Replies[0]

					client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
						Conversation: proto.String(firstReply.Text),
					})

					res := fmt.Sprintf("%v: %v", aiParticipant.Name, firstReply.Text)
					fmt.Println(res)
				}

			}
		}

		if v.Info.IsGroup {
			if v.Message.GetConversation() != "" {
				textmsg := fmt.Sprintf("%s: %s", v.Info.PushName, v.Message.GetConversation())
				fmt.Println("-- Group -- ", textmsg)

				messageResult, errMessage := caiClient.Chat.SendMessage(chatData.ExternalID, aiParticipant.User.Username, textmsg, nil)
				if errMessage != nil {
					fmt.Println(fmt.Errorf("unable to send message. Error: %v", errMessage))
				}

				if len(messageResult.Replies) > 0 {
					firstReply := messageResult.Replies[0]

					client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
						Conversation: proto.String(firstReply.Text),
					})

					res := fmt.Sprintf("%v: %v", aiParticipant.Name, firstReply.Text)
					fmt.Println(res)
				}

			}
		}
	}
}

func main() {
	// ndamel client charai
	var errClient error
	caiClient, errClient = cai.NewGoCAI(token, isPlus)
	if errClient != nil {
		fmt.Println(fmt.Errorf("unable to create client, error: %q", errClient))
		os.Exit(1)
	}

	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:golangai.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		// ganok ID, nggawe login anyar
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("QR code:", evt.Code)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// nek wes login, connect tok
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
