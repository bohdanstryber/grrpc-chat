package main

import (
	"console-chat/chatpb"
	"fmt"
	"io"
	"log"
	"net"

	"google.golang.org/grpc"
)

type chatServiceServer struct {
	chatpb.UnimplementedChatServiceServer

	//store the chan pointers contained in a single channel
	//communicate between JoinChannel and SendMessage rpc calls
	channel map[string][]chan *chatpb.Message
}

func (s *chatServiceServer) JoinChannel(ch *chatpb.Channel, msgStream chatpb.ChatService_JoinChannelServer) error {
	msgChannel := make(chan *chatpb.Message)
	//the joining information stores in the channel map
	s.channel[ch.Name] = append(s.channel[ch.Name], msgChannel)

	for {
		//select between two channels
		select {
		case <-msgStream.Context().Done(): // channel is closed
			return nil
		case msg := <-msgChannel:
			fmt.Printf("GO ROUTINE (got message): %v \n", msg)

			// The server-side handler can send a stream of protobuf messages to the client through this parameter’s Send method.
			msgStream.Send(msg)
		}
	}
}

func (s *chatServiceServer) SendMessage(msgStream chatpb.ChatService_SendMessageServer) error {
	//send the incoming message to them for which JoinChannel go routine is listening to

	//receive the message
	msg, err := msgStream.Recv()

	if err == io.EOF {
		return nil
	}

	if err != nil {
		return err
	}

	ack := chatpb.MessageAck{Status: "SENT"}

	msgStream.SendAndClose(&ack)

	go func() {
		streams := s.channel[msg.Channel.Name]

		for _, msgChan := range streams {
			msgChan <- msg
		}
	}()

	return nil
}

func main() {
	fmt.Println("------------------")
	fmt.Println("--- SERVER APP ---")
	fmt.Println("------------------")

	lis, err := net.Listen("tcp", "localhost:5400")

	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)

	chatpb.RegisterChatServiceServer(grpcServer, &chatServiceServer{
		channel: make(map[string][]chan *chatpb.Message),
	})

	grpcServer.Serve(lis)
}
