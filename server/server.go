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
	channel map[string][]chan *chatpb.Message
}

func (s *chatServiceServer) JoinChannel(ch *chatpb.Channel, msgStream chatpb.ChatService_JoinChannelServer) error {
	msgChannel := make(chan *chatpb.Message)
	s.channel[ch.Name] = append(s.channel[ch.Name], msgChannel)

	for {
		select {
		case <-msgStream.Context().Done(): // channel is closed
			return nil
		case msg := <-msgChannel:
			fmt.Printf("GO ROUTINE (got message): %v \n", msg)
			msgStream.Send(msg)
		}
	}
}

func (s *chatServiceServer) SendMessage(msgStream chatpb.ChatService_SendMessageServer) error {
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
